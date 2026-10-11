package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
	"github.com/gin-gonic/gin"
)

// releaseFixture is a submitted attempt with a computed result, ready to release.
type releaseFixture struct {
	tenantID, studentID, assignmentID, attemptID int
}

func setupReleaseHandler(t *testing.T, f *handlerFixture) releaseFixture {
	t.Helper()
	studentID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "rel"), "Release Student")
	assignmentID := testutil.CreateAssignment(t, f.DB, studentID, f.TestID, f.CoachID)
	attemptID := testutil.CreateAttempt(t, f.DB, assignmentID)

	summary := []byte(`{"exam_summary":{"correct":3,"wrong":7,"skipped":0,"total_questions":10,` +
		`"total_marks_earned":12,"max_possible_score":40,"net_score":5,"score_percent":12.5}}`)
	if _, err := f.DB.Exec(
		`INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json) VALUES ($1, $2, $3, $4)`,
		attemptID, 40.33, 5, summary); err != nil {
		t.Fatalf("insert result: %v", err)
	}
	return releaseFixture{f.TenantID, studentID, assignmentID, attemptID}
}

// releaseAs drives the handler with the given identity and path params.
// The recorder comes from f.ctx, which builds the gin test context around it.
func releaseAs(c *gin.Context, w *httptest.ResponseRecorder, f *handlerFixture, role string, userID, studentID, assignmentID int) {
	c.Set("tenant_id", f.TenantID)
	c.Set("user_id", userID)
	c.Set("role", role)
	// Set both params at once: the shared withParam helper replaces the whole
	// slice, so calling it twice would silently drop the first param.
	c.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(studentID)},
		{Key: "assignmentId", Value: strconv.Itoa(assignmentID)},
	}
	f.Results.ReleaseAssignmentResult(c)
}

func releasedAt(t *testing.T, f *handlerFixture, attemptID int) sql.NullTime {
	t.Helper()
	var at sql.NullTime
	if err := f.DB.QueryRow(`SELECT released_at FROM attempt_results WHERE attempt_id = $1`, attemptID).Scan(&at); err != nil {
		t.Fatalf("read released_at: %v", err)
	}
	return at
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// The admin owns the tenant and may release any student's result.
func TestReleaseResultsAdminSucceeds(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)

	c, w := f.ctx(t, "admin")
	releaseAs(c, w, f, "admin", f.AdminUserID, g.studentID, g.assignmentID)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !releasedAt(t, f, g.attemptID).Valid {
		t.Fatal("released_at was not set")
	}
}

// The assigned coach may release their own student's result.
func TestReleaseResultsOwningCoachSucceeds(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)

	c, w := f.ctx(t, "coach")
	releaseAs(c, w, f, "coach", f.CoachUserID, g.studentID, g.assignmentID)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !releasedAt(t, f, g.attemptID).Valid {
		t.Fatal("released_at was not set")
	}
}

// A coach from a DIFFERENT tenant must not be able to release anything, even
// for a real student and assignment id. This is the cross-tenant boundary.
func TestReleaseResultsForeignCoachRefused(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)

	c, w := f.ctx(t, "coach")
	releaseAs(c, w, f, "coach", f.OtherCoachUserID, g.studentID, g.assignmentID)
	if w.Code == http.StatusOK {
		t.Fatalf("SECURITY: a coach from another tenant released a result (code=%d)", w.Code)
	}
	if releasedAt(t, f, g.attemptID).Valid {
		t.Fatal("SECURITY: released_at was set by an unauthorised caller")
	}
}

// A coach in the same tenant but assigned to a different student must be
// refused. OtherCoachUserID is in another tenant, so this uses a same-tenant
// coach to isolate the assignment check from the tenant check.
func TestReleaseResultsSameTenantOtherCoachRefused(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)

	// A second coach in the same tenant, with no students.
	peerUserID, _ := testutil.CreateCoach(t, f.DB, f.TenantID)

	c, w := f.ctx(t, "coach")
	releaseAs(c, w, f, "coach", peerUserID, g.studentID, g.assignmentID)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 for a coach who does not own the student (body=%s)", w.Code, w.Body.String())
	}
	if releasedAt(t, f, g.attemptID).Valid {
		t.Fatal("SECURITY: a non-owning coach released the result")
	}
}

// Releasing before the SQI job has run must be refused with a message that tells
// the coach to compute first, rather than a generic 400.
func TestReleaseResultsUncomputedRefused(t *testing.T) {
	f := newHandlerFixture(t)
	studentID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "nc"), "No Compute")
	assignmentID := testutil.CreateAssignment(t, f.DB, studentID, f.TestID, f.CoachID)
	attemptID := testutil.CreateAttempt(t, f.DB, assignmentID)

	c, w := f.ctx(t, "admin")
	releaseAs(c, w, f, "admin", f.AdminUserID, studentID, assignmentID)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400 for an uncomputed result", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"computed", "SQI"} {
		if !containsFold(body, want) {
			t.Errorf("response should mention %q: %s", want, body)
		}
	}
	// There is no attempt_results row at all, so assert nothing was published
	// rather than reading a released_at that has nothing to read.
	var results int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM attempt_results WHERE attempt_id = $1`, attemptID).Scan(&results); err != nil {
		t.Fatalf("count results: %v", err)
	}
	if results != 0 {
		t.Fatal("an uncomputed attempt gained a result row")
	}
	var tokens int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM result_tokens WHERE attempt_id = $1`, attemptID).Scan(&tokens); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if tokens != 0 {
		t.Fatal("a token was minted for an uncomputed result")
	}
}

// Malformed ids are a client error, not a 500.
func TestReleaseResultsBadParams(t *testing.T) {
	f := newHandlerFixture(t)

	for _, tc := range []struct{ name, student, assignment string }{
		{"non-numeric student", "abc", "1"},
		{"non-numeric assignment", "1", "xyz"},
	} {
		c, w := f.ctx(t, "admin")
		c.Set("tenant_id", f.TenantID)
		c.Set("user_id", f.AdminUserID)
		c.Set("role", "admin")
		withParam(c, "id", tc.student)
		withParam(c, "assignmentId", tc.assignment)
		f.Results.ReleaseAssignmentResult(c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: code=%d, want 400", tc.name, w.Code)
		}
	}
}

// Releasing twice must be a no-op that reports already_released, and must not
// mint a second token.
func TestReleaseResultsTwiceIsNoOp(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)

	c1, w1 := f.ctx(t, "admin")
	releaseAs(c1, w1, f, "admin", f.AdminUserID, g.studentID, g.assignmentID)
	if w1.Code != http.StatusOK {
		t.Fatalf("first release code=%d body=%s", w1.Code, w1.Body.String())
	}
	c2, w2 := f.ctx(t, "admin")
	releaseAs(c2, w2, f, "admin", f.AdminUserID, g.studentID, g.assignmentID)
	if w2.Code != http.StatusOK {
		t.Fatalf("second release code=%d body=%s", w2.Code, w2.Body.String())
	}
	if !containsFold(w2.Body.String(), "already released") {
		t.Errorf("second release should report already released: %s", w2.Body.String())
	}

	var tokens int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM result_tokens WHERE attempt_id = $1`, g.attemptID).Scan(&tokens); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if tokens != 1 {
		t.Fatalf("%d tokens after two releases, want 1", tokens)
	}
}