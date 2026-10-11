package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// resultViewRelease publishes a result via the real handler and returns the raw
// token from the link, so the tests exercise the same path production uses.
func resultViewRelease(t *testing.T, f *handlerFixture, studentID, assignmentID int) string {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Params = gin.Params{
		{Key: "id", Value: itoa(studentID)},
		{Key: "assignmentId", Value: itoa(assignmentID)},
	}
	f.Results.ReleaseAssignmentResult(c)
	if w.Code != http.StatusOK {
		t.Fatalf("release code=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Link string `json:"link"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal release response: %v", err)
	}
	i := strings.LastIndex(resp.Link, "/")
	if i < 0 || resp.Link == "" {
		t.Fatalf("link carries no token: %q", resp.Link)
	}
	return resp.Link[i+1:]
}

func itoa(v int) string { return strconv.Itoa(v) }

// getResultByToken drives the public token endpoint. The request must be set:
// the shared 404/401 helpers log c.Request.Method and c.Request.URL, so a nil
// request panics inside the logger rather than in the handler.
func getResultByToken(f *handlerFixture, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/student/results", nil)
	c.Params = gin.Params{{Key: "token", Value: token}}
	f.Results.GetResultByToken(c)
	return w
}

// A valid token returns the score.
func TestGetResultByTokenSucceeds(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)
	token := resultViewRelease(t, f, g.studentID, g.assignmentID)

	rec := getResultByToken(f, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Result struct {
			MarksEarned float64 `json:"marks_earned"`
			MaxMarks    float64 `json:"max_marks"`
			Percentage  float64 `json:"percentage"`
			StudentName string  `json:"student_name"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Result.MarksEarned == 0 && resp.Result.MaxMarks == 0 {
		t.Fatalf("no score in payload: %s", rec.Body.String())
	}

	// The token must not travel onward from this page.
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy=%q, want no-referrer so the token cannot leak via Referer", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control=%q, want no-store", got)
	}
}

// Every token failure is a 404 with the same body, so the endpoint cannot be
// used to probe for valid tokens.
func TestGetResultByTokenFailuresAre404(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)
	token := resultViewRelease(t, f, g.studentID, g.assignmentID)

	cases := map[string]string{
		"empty":     "",
		"garbage":   "not-a-real-token",
		"too_long":  strings.Repeat("a", 300),
	}
	for name, tok := range cases {
		rec := getResultByToken(f, tok)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: code=%d, want 404", name, rec.Code)
		}
	}

	// Revoke and confirm the same 404.
	if err := f.Results.TokenRepo.DeleteForAttempt(g.attemptID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	rec := getResultByToken(f, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("revoked token: code=%d, want 404 (body=%s)", rec.Code, rec.Body.String())
	}
}

// The token response must not carry coach-facing diagnostics.
func TestGetResultByTokenOmitsCoachingSignals(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)
	token := resultViewRelease(t, f, g.studentID, g.assignmentID)

	rec := getResultByToken(f, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, banned := range []string{"sqi", "SQI", "rank", "percentile", "grade"} {
		if strings.Contains(body, banned) {
			t.Errorf("token payload leaked %q: %s", banned, body)
		}
	}
}

// The authenticated list returns the caller's own released results and refuses
// a request without a verified student identity.
func TestListMyResults(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)
	resultViewRelease(t, f, g.studentID, g.assignmentID)

	// Signed in as the student who owns the result.
	c, w := f.ctxAsStudentWithID(t, g.studentID)
	f.Results.ListMyResults(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "marks_earned") {
		t.Errorf("expected the result payload: %s", w.Body.String())
	}

	// A context without a student identity must be refused.
	c2, w2 := f.ctx(t, "admin")
	f.Results.ListMyResults(c2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401 without a student identity", w2.Code)
	}
}

// An unreleased result must not appear on the authenticated list.
func TestListMyResultsExcludesUnreleased(t *testing.T) {
	f := newHandlerFixture(t)
	g := setupReleaseHandler(t, f)
	// setupReleaseHandler inserts a result but never releases it.

	c, w := f.ctxAsStudentWithID(t, g.studentID)
	f.Results.ListMyResults(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"results":[]`) {
		t.Fatalf("an unreleased result was listed: %s", w.Body.String())
	}
}