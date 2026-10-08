package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestParseIDParam(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	withParam(c, "id", "42")

	id, err := parseIDParam(c, "id")
	if err != nil || id != 42 {
		t.Fatalf("parseIDParam=(%d,%v)", id, err)
	}
}

func TestParseIDParamInvalidResponds400(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	withParam(c, "id", "not-a-number")

	if _, err := parseIDParam(c, "id"); err == nil {
		t.Fatal("expected error")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestGetStudentIDFromContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("student_id", 7)

	id, err := getStudentIDFromContext(c)
	if err != nil || id != 7 {
		t.Fatalf("=(%d,%v)", id, err)
	}
}

func TestGetStudentIDFromContextMissing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	if _, err := getStudentIDFromContext(c); err == nil {
		t.Fatal("missing student_id must error")
	}
}

func TestGetStudentIDFromContextWrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("student_id", "7") // string, not int
	if _, err := getStudentIDFromContext(c); err == nil {
		t.Fatal("non-int student_id must error")
	}
}

func TestVerifyCoachExists(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")

	if !verifyCoachExists(c, f.CoachID, f.TenantID, f.CoachRepo) {
		t.Fatalf("own coach must verify; got %d %s", w.Code, w.Body.String())
	}
}

func TestVerifyCoachExistsCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")

	if verifyCoachExists(c, f.OtherCoachID, f.TenantID, f.CoachRepo) {
		t.Fatal("foreign coach must not verify in our tenant")
	}
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404", w.Code)
	}
}

func TestVerifyTestAccessCoachOwnsTest(t *testing.T) {
	f := newHandlerFixture(t)
	c, _ := f.ctxAsCoach(t)
	if err := verifyTestAccess(c, f.TestID, "coach", f.UserRepo, f.CoachRepo, f.TestPaperRepo, f.TenantID); err != nil {
		t.Fatalf("coach must access own test: %v", err)
	}
}

func TestVerifyTestAccessCoachForeignTest(t *testing.T) {
	f := newHandlerFixture(t)
	c, _ := f.ctxAsCoach(t)
	if err := verifyTestAccess(c, f.OtherTestID, "coach", f.UserRepo, f.CoachRepo, f.TestPaperRepo, f.TenantID); err == nil {
		t.Fatal("coach must not access another tenant's test")
	}
}

func TestVerifyTestAccessAdminInTenant(t *testing.T) {
	f := newHandlerFixture(t)
	c, _ := f.ctx(t, "admin")
	if err := verifyTestAccess(c, f.TestID, "admin", f.UserRepo, f.CoachRepo, f.TestPaperRepo, f.TenantID); err != nil {
		t.Fatalf("admin must access own tenant's test: %v", err)
	}
}

func TestVerifyTestAccessAdminCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	c, _ := f.ctx(t, "admin")
	if err := verifyTestAccess(c, f.OtherTestID, "admin", f.UserRepo, f.CoachRepo, f.TestPaperRepo, f.TenantID); err == nil {
		t.Fatal("admin must not access another tenant's test")
	}
}

func TestVerifyTestAccessUnknownRole(t *testing.T) {
	f := newHandlerFixture(t)
	c, _ := f.ctxAsStudent(t)
	if err := verifyTestAccess(c, f.TestID, "student", f.UserRepo, f.CoachRepo, f.TestPaperRepo, f.TenantID); err == nil {
		t.Fatal("student role must be rejected by verifyTestAccess")
	}
}

func TestBuildAssignmentResultsResponseNoAttempt(t *testing.T) {
	f := newHandlerFixture(t)

	body, err := buildAssignmentResultsResponse(f.AttemptRepo, f.StudentID, f.AssignmentID,
		"Primary Student", "S-1", f.TestID, "Test Title", "assigned", "2026-01-01")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if body["attempt"] != nil {
		t.Fatalf("attempt must be null with no submitted attempt: %v", body["attempt"])
	}
	if body["sqi_score"] != nil {
		t.Fatalf("sqi_score must be null: %v", body["sqi_score"])
	}
}

func TestBuildAssignmentResultsResponseWithAttempt(t *testing.T) {
	f := newHandlerFixture(t)
	q1 := testutil.CreateQuestionWithAnswer(t, f.DB, f.TestID, "A")

	seen := true
	res, err := f.AttemptRepo.SubmitAnswersTx(f.AssignmentID, map[int]string{q1: "A"},
		[]repository.AnswerInput{{QuestionID: q1, SelectedAnswer: "A", TimeSpent: 12, Seen: &seen}},
		func(tx *sql.Tx) error { return nil })
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := f.AttemptRepo.StoreResult(res.AttemptID, 77, 4, []byte(`{"v":1}`), "v2"); err != nil {
		t.Fatalf("store result: %v", err)
	}

	body, err := buildAssignmentResultsResponse(f.AttemptRepo, f.StudentID, f.AssignmentID,
		"Primary Student", "S-1", f.TestID, "Test Title", "submitted", "2026-01-01")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if body["attempt"] == nil {
		t.Fatal("attempt must be populated")
	}
	if body["sqi_score"] != float64(77) {
		t.Fatalf("sqi_score=%v", body["sqi_score"])
	}
	answers, _ := body["answers"].([]repository.AnswerDetail)
	if len(answers) != 1 || answers[0].QuestionID != q1 {
		t.Fatalf("answers=%v", body["answers"])
	}
}

func TestResolveTenantIDAndCoachHelpers(t *testing.T) {
	f := newHandlerFixture(t)
	c, _ := f.ctxAsCoach(t)

	tid, err := resolveTenantID(c)
	if err != nil || tid != f.TenantID {
		t.Fatalf("resolveTenantID=(%d,%v)", tid, err)
	}
	cid, err := resolveCoachID(c, f.CoachRepo)
	if err != nil || cid != f.CoachID {
		t.Fatalf("resolveCoachID=(%d,%v)", cid, err)
	}
	gotCoach, gotTenant, err := resolveCoachAndTenant(c, f.CoachRepo)
	if err != nil || gotCoach != f.CoachID || gotTenant != f.TenantID {
		t.Fatalf("resolveCoachAndTenant=(%d,%d,%v)", gotCoach, gotTenant, err)
	}
}

// resolveTenantID must fail when the key is absent. It used to return
// (0, nil), which made the `if err != nil` branch dead at all 43 call sites and
// turned a missing tenant into an unscoped query against tenant 0.
func TestResolveTenantIDFailsClosed(t *testing.T) {
	newHandlerFixture(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	tid, err := resolveTenantID(c)
	if err == nil {
		t.Fatalf("resolveTenantID on a bare context returned (%d, nil); it must refuse", tid)
	}
	if tid != 0 {
		t.Fatalf("on failure resolveTenantID returned %d, want 0", tid)
	}
}

// A present-but-zero tenant_id is legitimate, not a failure: users.tenant_id is
// nullable and every super_admin has NULL, so AuthMiddleware sets tenant_id to 0
// for them. An implementation that treated 0 as "missing" would refuse every
// super-admin route.
func TestResolveTenantIDAllowsPresentZero(t *testing.T) {
	newHandlerFixture(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", 0)

	tid, err := resolveTenantID(c)
	if err != nil {
		t.Fatalf("resolveTenantID rejected a super-admin's tenant_id of 0: %v", err)
	}
	if tid != 0 {
		t.Fatalf("tid=%d want 0", tid)
	}
}

// The 42 call sites all map the error to Unauthorized, so a context with no
// tenant produces a 401 instead of a 500 or silently querying tenant 0.
func TestHandlerWithMissingTenantIsRefused(t *testing.T) {
	f := newHandlerFixture(t)

	// ListCoaches calls resolveTenantID first and returns Unauthorized on error.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/coaches", nil)
	c.Set("user_id", f.AdminUserID)
	c.Set("role", "admin")
	// no tenant_id

	f.Admin.ListCoaches(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("ListCoaches code=%d want 401 Unauthorized; body=%s", w.Code, w.Body.String())
	}
}

