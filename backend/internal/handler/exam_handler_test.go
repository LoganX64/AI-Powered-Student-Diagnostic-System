package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// The exam surface is the highest-traffic authorization path in the app: every
// method loads the assignment, then refuses unless the assignment's owner is the
// student in the context. Each test here builds the fixture and deliberately
// points the student at somebody else's assignment.

func examCtx(t *testing.T, f *handlerFixture, studentID int, assignmentID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("tenant_id", f.TenantID)
	c.Set("student_id", studentID)
	c.Set("role", "student")
	withParam(c, "id", assignmentID)
	return c, w
}

func TestStartExamInvalidAssignmentID(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := examCtx(t, f, f.StudentID, "not-a-number")
	f.Student.StartExam(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestStartExamUnauthorizedNoStudentContext(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	withParam(c, "id", strconv.Itoa(f.AssignmentID))

	f.Student.StartExam(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestStartExamForeignAssignmentForbidden(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := examCtx(t, f, f.StudentID, strconv.Itoa(f.OtherAssignmentID))
	f.Student.StartExam(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

func TestStartExamMissingAssignmentNotFound(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := examCtx(t, f, f.StudentID, "999999999")
	f.Student.StartExam(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404", w.Code)
	}
}

func TestAutosaveForeignAssignmentForbidden(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := examCtx(t, f, f.StudentID, strconv.Itoa(f.OtherAssignmentID))
	f.Student.Autosave(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

func TestAutosaveInvalidAssignmentID(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := examCtx(t, f, f.StudentID, "abc")
	f.Student.Autosave(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestGetStateForeignAssignmentForbidden(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := examCtx(t, f, f.StudentID, strconv.Itoa(f.OtherAssignmentID))
	f.Student.GetState(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

func TestSubmitExamForeignAssignmentForbidden(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := examCtx(t, f, f.StudentID, strconv.Itoa(f.OtherAssignmentID))
	f.Student.SubmitExam(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

func TestSubmitExamUnauthorizedNoStudentContext(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	withParam(c, "id", strconv.Itoa(f.AssignmentID))

	f.Student.SubmitExam(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestListStudentAssignmentsUnauthorized(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	f.Student.ListStudentAssignments(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestGetAssignmentQuestionsForeignAssignment(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := examCtx(t, f, f.StudentID, strconv.Itoa(f.OtherAssignmentID))
	f.Student.GetAssignmentQuestions(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403 (body=%s)", w.Code, w.Body.String())
	}
}

func TestGraceSecondsDefault(t *testing.T) {
	f := newHandlerFixture(t)
	// Fixture sets SubmitGraceSeconds=30 explicitly.
	if got := f.Student.graceSeconds(); got != 30 {
		t.Fatalf("graceSeconds=%d, want 30", got)
	}
	// With a nil Cfg the helper must still return the 30s default.
	g := &StudentHandler{}
	if got := g.graceSeconds(); got != 30 {
		t.Fatalf("nil-Cfg graceSeconds=%d, want 30", got)
	}
}
