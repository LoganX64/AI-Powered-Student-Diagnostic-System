package liveview

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"

	"github.com/gin-gonic/gin"
)

// wsFixture builds a tenant graph plus the two WS handlers, and a second tenant
// used to prove tenant scoping. No websocket handshake happens: every assertion
// below is on a guard that returns before upgrader.Upgrade.

type wsFixture struct {
	DB *sql.DB

	StudentHandler *StudentWSHandler
	ViewerHandler  *ViewerWSHandler
	Hub            *Hub

	TenantID     int
	AdminUserID  int
	CoachUserID  int
	CoachID      int
	StudentID    int
	AssignmentID int

	OtherTenantID  int
	OtherCoachID   int
	OtherStudentID int
}

func newWSFixture(t *testing.T) *wsFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.OpenTestDB(t)

	hub := NewHub(nil)
	f := &wsFixture{
		DB:             db,
		Hub:            hub,
		StudentHandler: NewStudentWSHandler(hub, repository.NewStudentRepo(db), repository.NewAssignmentRepo(db)),
		ViewerHandler:  NewViewerWSHandler(hub, repository.NewStudentRepo(db), repository.NewAssignmentRepo(db), repository.NewCoachRepo(db)),
	}

	f.TenantID = testutil.CreateTenant(t, db)
	f.AdminUserID = testutil.CreateAdmin(t, db, f.TenantID)
	f.CoachUserID, f.CoachID = testutil.CreateCoach(t, db, f.TenantID)
	subjectID := testutil.CreateSubject(t, db, f.TenantID, "Math")
	testID := testutil.CreateTest(t, db, f.TenantID, subjectID, f.CoachID, 60)
	f.StudentID = testutil.CreateStudent(t, db, f.TenantID, f.CoachID, testutil.UniqueCode(t, "wsstu"), "WS Student")
	f.AssignmentID = testutil.CreateAssignment(t, db, f.StudentID, testID, f.CoachID)

	f.OtherTenantID = testutil.CreateTenant(t, db)
	_, f.OtherCoachID = testutil.CreateCoach(t, db, f.OtherTenantID)
	otherSubject := testutil.CreateSubject(t, db, f.OtherTenantID, "Other")
	testutil.CreateTest(t, db, f.OtherTenantID, otherSubject, f.OtherCoachID, 60)
	f.OtherStudentID = testutil.CreateStudent(t, db, f.OtherTenantID, f.OtherCoachID, testutil.UniqueCode(t, "wsoth"), "Other Student")

	return f
}

func wsCtx(role string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	if role != "" {
		c.Set("role", role)
	}
	return c, w
}

// --- StudentLiveStream guards ---

func TestStudentLiveStreamMissingSession(t *testing.T) {
	f := newWSFixture(t)
	c, w := wsCtx("student") // no student_id in context
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(f.AssignmentID)}}

	f.StudentHandler.StudentLiveStream(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestStudentLiveStreamNonStudentRole(t *testing.T) {
	f := newWSFixture(t)
	c, w := wsCtx("admin")
	c.Set("student_id", f.StudentID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(f.AssignmentID)}}

	f.StudentHandler.StudentLiveStream(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestStudentLiveStreamInvalidAssignmentID(t *testing.T) {
	f := newWSFixture(t)
	c, w := wsCtx("student")
	c.Set("student_id", f.StudentID)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	f.StudentHandler.StudentLiveStream(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestStudentLiveStreamMissingAssignment(t *testing.T) {
	f := newWSFixture(t)
	c, w := wsCtx("student")
	c.Set("student_id", f.StudentID)
	c.Params = gin.Params{{Key: "id", Value: "999999999"}}

	f.StudentHandler.StudentLiveStream(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404", w.Code)
	}
}

// TestStudentLiveStreamForeignAssignment covers the ownership guard: the student
// in context does not own the assignment.
func TestStudentLiveStreamForeignAssignment(t *testing.T) {
	f := newWSFixture(t)
	otherSubject := testutil.CreateSubject(t, f.DB, f.OtherTenantID, "Other 2")
	otherTest := testutil.CreateTest(t, f.DB, f.OtherTenantID, otherSubject, f.OtherCoachID, 30)
	otherAssignment := testutil.CreateAssignment(t, f.DB, f.OtherStudentID, otherTest, f.OtherCoachID)

	c, w := wsCtx("student")
	c.Set("student_id", f.StudentID)
	c.Set("tenant_id", f.TenantID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(otherAssignment)}}

	f.StudentHandler.StudentLiveStream(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

// --- ViewerLiveStream guards ---

func TestViewerLiveStreamRequiresAdminOrCoach(t *testing.T) {
	f := newWSFixture(t)
	c, w := wsCtx("student")
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.AdminUserID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(f.StudentID)}}

	f.ViewerHandler.ViewerLiveStream(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestViewerLiveStreamInvalidStudentID(t *testing.T) {
	f := newWSFixture(t)
	c, w := wsCtx("admin")
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.AdminUserID)
	c.Params = gin.Params{{Key: "id", Value: "xyz"}}

	f.ViewerHandler.ViewerLiveStream(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestViewerLiveStreamMissingStudent(t *testing.T) {
	f := newWSFixture(t)
	c, w := wsCtx("admin")
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.AdminUserID)
	c.Params = gin.Params{{Key: "id", Value: "999999999"}}

	f.ViewerHandler.ViewerLiveStream(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404", w.Code)
	}
}

// TestViewerLiveStreamCrossTenantForbidden is the tenant-isolation guard: an
// admin of tenant A must not watch a student belonging to tenant B.
func TestViewerLiveStreamCrossTenantForbidden(t *testing.T) {
	f := newWSFixture(t)
	c, w := wsCtx("admin")
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.AdminUserID)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(f.OtherStudentID)}}

	f.ViewerHandler.ViewerLiveStream(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}
