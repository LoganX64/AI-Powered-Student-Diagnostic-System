package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// Cross-tenant and role guards across the admin/coach CRUD surfaces. The rule
// under test everywhere: an id belonging to another tenant must never be
// reachable, and a context without the expected identity must be refused.

func adminCtxWithID(t *testing.T, f *handlerFixture, id int) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.AdminUserID)
	c.Set("role", "admin")
	withParam(c, "id", strconv.Itoa(id))
	return w, c
}

func TestAdminGetStudentCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := adminCtxWithID(t, f, f.OtherStudentID)
	f.Admin.GetStudent(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetStudent cross-tenant code=%d, want 404 (body=%s)", w.Code, w.Body.String())
	}
}

func TestAdminDeleteStudentCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := adminCtxWithID(t, f, f.OtherStudentID)
	f.Admin.DeleteStudent(c)
	if w.Code == http.StatusOK {
		t.Fatalf("must not delete another tenant's student: %s", w.Body.String())
	}
	// student must still exist
	if _, err := f.StudentRepo.GetName(f.OtherStudentID, f.OtherTenantID); err != nil {
		t.Fatalf("foreign student was mutated: %v", err)
	}
}

func TestAdminUpdateStudentCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := adminCtxWithID(t, f, f.OtherStudentID)
	c.Request = httptest.NewRequest(http.MethodPut, "/x", postBody(`{"name":"Hacked","student_code":"x","coach_id":1}`))
	c.Request.Header.Set("Content-Type", "application/json")

	f.Admin.UpdateStudent(c)
	if w.Code == http.StatusOK {
		t.Fatalf("must not update another tenant's student: %s", w.Body.String())
	}
	name, _ := f.StudentRepo.GetName(f.OtherStudentID, f.OtherTenantID)
	if name != "Foreign Student" {
		t.Fatalf("foreign student name changed to %q", name)
	}
}

func TestAdminGetTestCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := adminCtxWithID(t, f, f.OtherTestID)
	f.Admin.GetTest(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetTest cross-tenant code=%d, want 404", w.Code)
	}
}

func TestAdminDeleteTestCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := adminCtxWithID(t, f, f.OtherTestID)
	f.Admin.DeleteTest(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
	// The foreign test must be untouched — this is what a 500 crash would hide.
	if ok, _ := f.TestPaperRepo.Exists(f.OtherTestID, f.OtherTenantID); !ok {
		t.Fatal("foreign test was soft-deleted")
	}
}

func TestAdminDeleteAssignmentCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := adminCtxWithID(t, f, f.OtherAssignmentID)
	f.Admin.DeleteAssignment(c)
	if w.Code == http.StatusOK {
		t.Fatalf("must not delete another tenant's assignment: %s", w.Body.String())
	}
	// assignment must survive
	if _, _, _, _, _, err := f.AssignmentRepo.GetByID(f.OtherAssignmentID, f.OtherStudentID); err != nil {
		t.Fatalf("foreign assignment was deleted: %v", err)
	}
}

func TestAdminListStudentsScopedToTenant(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	f.Admin.ListStudents(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	// The other tenant's student must not appear in our listing.
	if strings.Contains(w.Body.String(), "Foreign Student") {
		t.Fatalf("list leaked the other tenant's student: %s", w.Body.String())
	}
}

func TestAdminListTestsScopedToTenant(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	f.Admin.ListTests(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	if strings.Contains(w.Body.String(), "Foreign Subject") {
		t.Fatalf("list leaked the other tenant's test: %s", w.Body.String())
	}
}

func TestAdminListSubjectsScopedToTenant(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	f.Admin.ListSubjects(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	if strings.Contains(w.Body.String(), "Foreign Subject") {
		t.Fatalf("list leaked the other tenant's subject: %s", w.Body.String())
	}
}

func TestAdminListAssignmentsScopedToTenant(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	f.Admin.ListAssignments(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d", w.Code)
	}
	if strings.Contains(w.Body.String(), "Foreign Student") {
		t.Fatalf("list leaked the other tenant's assignment: %s", w.Body.String())
	}
}

func TestCoachGetStudentSQIInvalidID(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.CoachUserID)
	c.Set("role", "coach")
	withParam(c, "id", "abc")

	f.Coach.GetStudentSQI(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestCoachGetStudentSQIForeignStudent(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.CoachUserID)
	c.Set("role", "coach")
	withParam(c, "id", strconv.Itoa(f.OtherStudentID))

	f.Coach.GetStudentSQI(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

func TestCoachListCoachesScoped(t *testing.T) {
	f := newHandlerFixture(t)
	// A coach listing returns coach rows (id/user_id/name/email), so assert on
	// the foreign coach's email rather than a subject name, which can never
	// appear in this payload.
	var foreignEmail string
	if err := f.DB.QueryRow(`SELECT email FROM users WHERE id = $1`, f.OtherCoachUserID).Scan(&foreignEmail); err != nil {
		t.Fatalf("read foreign coach email: %v", err)
	}

	c, w := f.ctxAsCoach(t)
	f.Coach.ListCoaches(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if foreignEmail != "" && strings.Contains(w.Body.String(), foreignEmail) {
		t.Fatalf("coach listing leaked another tenant's coach (%s): %s", foreignEmail, w.Body.String())
	}
}

func TestCoachGetAssignmentResultsForeignAssignment(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.CoachUserID)
	c.Set("role", "coach")
	// This route takes both a student id and an assignment id; the student is
	// the foreign one, so ExistsActive rejects it before the assignment matters.
	// Both params must be set in one slice — withParam replaces c.Params.
	c.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(f.OtherStudentID)},
		{Key: "assignmentId", Value: strconv.Itoa(f.OtherAssignmentID)},
	}

	f.Coach.GetAssignmentResults(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 (body=%s)", w.Code, w.Body.String())
	}
}

func TestCoachUpdateSubjectCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/x", postBody(`{"name":"Hijacked"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", f.CoachUserID)
	c.Set("role", "coach")
	withParam(c, "id", strconv.Itoa(f.OtherSubjectID))

	f.Coach.UpdateSubject(c)
	if w.Code == http.StatusOK {
		t.Fatalf("must not rename another tenant's subject: %s", w.Body.String())
	}
	name, err := f.TestPaperRepo.GetSubjectName(f.OtherTestID)
	if err != nil || name != "Foreign Subject" {
		t.Fatalf("foreign subject name=%q err=%v", name, err)
	}
}

func TestAdminDeleteSubjectCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := adminCtxWithID(t, f, f.OtherSubjectID)
	f.Admin.DeleteSubject(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 (body=%s)", w.Code, w.Body.String())
	}
	// The foreign subject must still be active in its own tenant.
	subs, _, err := f.TestPaperRepo.ListSubjects(f.OtherTenantID, "", false, 10, 0)
	if err != nil {
		t.Fatalf("ListSubjects: %v", err)
	}
	for _, s := range subs {
		if s.SubjectID == f.OtherSubjectID {
			return // survived
		}
	}
	t.Fatalf("foreign subject was soft-deleted; subjects=%+v", subs)
}
