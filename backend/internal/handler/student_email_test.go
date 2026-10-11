package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
	"github.com/gin-gonic/gin"
)

// Phase 6: the coach-facing email field and the release-state flags that drive
// the Release button.

func TestNormalizeStudentEmail(t *testing.T) {
	cases := []struct {
		in   string
		want *string
	}{
		{"", nil},
		{"   ", nil},
		{"\t\n", nil},
		{"Riya@Example.COM", ptr("riya@example.com")},
		{" riya@example.com  ", ptr("riya@example.com")},
		{"a@b.co", ptr("a@b.co")},
	}
	for _, tc := range cases {
		got := normalizeStudentEmail(tc.in)
		switch {
		case tc.want == nil && got != nil:
			t.Errorf("normalizeStudentEmail(%q) = %q, want nil", tc.in, *got)
		case tc.want != nil && got == nil:
			t.Errorf("normalizeStudentEmail(%q) = nil, want %q", tc.in, *tc.want)
		case tc.want != nil && *got != *tc.want:
			t.Errorf("normalizeStudentEmail(%q) = %q, want %q", tc.in, *got, *tc.want)
		}
	}
}

func TestValidStudentEmail(t *testing.T) {
	valid := []string{"a@b.co", "first.last@sub.domain.org", "x+y@z.io", "UPPER@EXAMPLE.COM"}
	for _, e := range valid {
		if !validStudentEmail(e) {
			t.Errorf("validStudentEmail(%q) = false, want true", e)
		}
	}
	invalid := []string{
		"", "plain", "no-at-sign", "@nolocal.com", "notrailing@",
		"a@b", "a@@b.com", "two@at@signs.com", "has space@b.com",
		"tab\t@b.com", "newline\n@b.com", strings.Repeat("a", 250) + "@b.com",
	}
	for _, e := range invalid {
		if validStudentEmail(e) {
			t.Errorf("validStudentEmail(%q) = true, want false", e)
		}
	}
}

func ptr(s string) *string { return &s }

// studentBody builds a student payload with json.Marshal rather than a format
// string, so an address containing punctuation can never corrupt the JSON.
func studentBody(t *testing.T, coachID int, name, email string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"name": name, "coach_id": coachID, "email": email,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return string(b)
}

// withAssignmentResultsParams sets both route params at once: withParam
// replaces the whole slice, so setting them one after another drops the first.
func withAssignmentResultsParams(c *gin.Context, studentID, assignmentID int) {
	c.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(studentID)},
		{Key: "assignmentId", Value: strconv.Itoa(assignmentID)},
	}
}

func studentEmail(t *testing.T, f *handlerFixture, studentID int) sql.NullString {
	t.Helper()
	var e sql.NullString
	if err := f.DB.QueryRow(`SELECT email FROM students WHERE id=$1`, studentID).Scan(&e); err != nil {
		t.Fatalf("read email: %v", err)
	}
	return e
}

// TestEnsureStudentCodeDoesNotRetryOnEmailCollision is the regression test for a
// bug introduced by migration 000021: ensureStudentCode retried on ANY 23505,
// so a duplicate email burned all ten attempts regenerating student codes and
// then reported "failed to generate a unique student code" - a 500 for what is
// a user input error, with a message naming the wrong field.
func TestEnsureStudentCodeDoesNotRetryOnEmailCollision(t *testing.T) {
	f := newHandlerFixture(t)

	taken := "collide@example.com"
	other := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "em"), "Email Holder")
	if _, err := f.DB.Exec(`UPDATE students SET email=$1 WHERE id=$2`, taken, other); err != nil {
		t.Fatalf("seed email: %v", err)
	}

	calls := 0
	orig := generateStudentCode
	t.Cleanup(func() { generateStudentCode = orig })
	generateStudentCode = func(tenantID int) string {
		calls++
		return testutil.UniqueCode(t, "wasted")
	}

	_, _, err := ensureStudentCode(f.StudentRepo, f.TenantID, "Second", "", f.CoachID, &taken)
	if err == nil {
		t.Fatal("expected an error for a duplicate email")
	}
	// One code is generated for the single insert attempt. The regression is
	// looping: the old code retried up to 10 times, so anything above one here
	// means a duplicate email is still being treated as a code collision.
	if calls != 1 {
		t.Fatalf("generator called %d times; a duplicate email cannot be fixed by regenerating a student code", calls)
	}
	if err != ErrStudentEmailTaken {
		t.Fatalf("err = %v, want ErrStudentEmailTaken", err)
	}
}

func TestCreateStudentStoresNormalizedEmail(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	withJSONBody(c, http.MethodPost, studentBody(t, f.CoachID, "Emailed", "  Riya@Example.COM  "))

	f.Admin.CreateStudent(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var id int
	if err := f.DB.QueryRow(`SELECT id FROM students WHERE name='Emailed'`).Scan(&id); err != nil {
		t.Fatalf("find student: %v", err)
	}
	if got := studentEmail(t, f, id); !got.Valid || got.String != "riya@example.com" {
		t.Fatalf("stored email = %v, want riya@example.com", got)
	}
}

// Omitting the field must not clobber, and an explicit empty string must clear.
func TestCreateStudentWithoutEmailStoresNull(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	withJSONBody(c, http.MethodPost, studentBody(t, f.CoachID, "No Email", ""))

	f.Admin.CreateStudent(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var id int
	_ = f.DB.QueryRow(`SELECT id FROM students WHERE name='No Email'`).Scan(&id)
	if got := studentEmail(t, f, id); got.Valid {
		t.Fatalf("email = %q, want NULL", got.String)
	}
}

func TestCreateStudentEmailRejections(t *testing.T) {
	f := newHandlerFixture(t)

	existing := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "dup"), "First")
	if _, err := f.DB.Exec(`UPDATE students SET email='dup@example.com' WHERE id=$1`, existing); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cases := []struct {
		name, email string
	}{
		{"duplicate in tenant", "dup@example.com"},
		// Normalization lowercases before the uniqueness check, so a
		// differently-cased duplicate must still be refused.
		{"duplicate differing case", "DUP@Example.com"},
		{"duplicate with surrounding space", "  dup@example.com  "},
		{"malformed", "not-an-email"},
		{"no at sign", "plainaddress"},
		{"no domain dot", "a@b"},
		{"embedded whitespace", "has space@example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := f.ctx(t, "admin")
			body, err := json.Marshal(map[string]any{
				"name": "Second", "coach_id": f.CoachID, "email": tc.email,
			})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			withJSONBody(c, http.MethodPost, string(body))

			f.Admin.CreateStudent(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("code=%d want 400 (body=%s)", w.Code, w.Body.String())
			}
		})
	}
}

// Two tenants may hold the same address; only same-tenant duplicates collide.
func TestCreateStudentSameEmailDifferentTenantAllowed(t *testing.T) {
	f := newHandlerFixture(t)
	otherTenant := testutil.CreateTenant(t, f.DB)
	_, otherCoach := testutil.CreateCoach(t, f.DB, otherTenant)

	stu := testutil.CreateStudent(t, f.DB, otherTenant, otherCoach, testutil.UniqueCode(t, "cross"), "Cross Tenant")
	if _, err := f.DB.Exec(`UPDATE students SET email='shared@example.com' WHERE id=$1`, stu); err != nil {
		t.Fatalf("seed: %v", err)
	}

	c, w := f.ctx(t, "admin")
	withJSONBody(c, http.MethodPost, studentBody(t, f.CoachID, "Mine", "shared@example.com"))
	f.Admin.CreateStudent(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d want 201 (body=%s)", w.Code, w.Body.String())
	}
}

func TestUpdateStudentSetsAndClearsEmail(t *testing.T) {
	f := newHandlerFixture(t)
	studentID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "upd"), "Updatable")

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(studentID))
	withJSONBody(c, http.MethodPut, studentBody(t, f.CoachID, "Updatable", "New@Example.com"))
	f.Admin.UpdateStudent(c)
	if w.Code != http.StatusOK {
		t.Fatalf("set: code=%d body=%s", w.Code, w.Body.String())
	}
	if got := studentEmail(t, f, studentID); !got.Valid || got.String != "new@example.com" {
		t.Fatalf("after set email = %v, want new@example.com", got)
	}

	// An empty string is an explicit clear, distinct from omitting the key.
	c2, w2 := f.ctx(t, "admin")
	withParam(c2, "id", strconv.Itoa(studentID))
	withJSONBody(c2, http.MethodPut, studentBody(t, f.CoachID, "Updatable", ""))
	f.Admin.UpdateStudent(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("clear: code=%d body=%s", w2.Code, w2.Body.String())
	}
	if got := studentEmail(t, f, studentID); got.Valid {
		t.Fatalf("after clear email = %q, want NULL", got.String)
	}
}

func TestUpdateStudentEmailRejections(t *testing.T) {
	f := newHandlerFixture(t)

	holder := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "h"), "Holder")
	if _, err := f.DB.Exec(`UPDATE students SET email='holder@example.com' WHERE id=$1`, holder); err != nil {
		t.Fatalf("seed: %v", err)
	}
	target := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "t"), "Target")

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(target))
	withJSONBody(c, http.MethodPut, studentBody(t, f.CoachID, "Target", "holder@example.com"))
	f.Admin.UpdateStudent(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want 400 (body=%s)", w.Code, w.Body.String())
	}
	// The rejected write must not have landed.
	if got := studentEmail(t, f, target); got.Valid {
		t.Fatalf("email = %q, want NULL after a rejected update", got.String)
	}
}

// Re-saving a student with their OWN email must succeed; the uniqueness check
// has to exclude the student being edited.
func TestUpdateStudentKeepsOwnEmail(t *testing.T) {
	f := newHandlerFixture(t)
	studentID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "own"), "Owner")
	if _, err := f.DB.Exec(`UPDATE students SET email='mine@example.com' WHERE id=$1`, studentID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	c, w := f.ctx(t, "admin")
	withParam(c, "id", strconv.Itoa(studentID))
	withJSONBody(c, http.MethodPut, studentBody(t, f.CoachID, "Owner Renamed", "mine@example.com"))
	f.Admin.UpdateStudent(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d want 200 (body=%s)", w.Code, w.Body.String())
	}
	if got := studentEmail(t, f, studentID); !got.Valid || got.String != "mine@example.com" {
		t.Fatalf("email = %v, want mine@example.com", got)
	}
}

func TestCoachCreateStudentWithEmail(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctxAsCoach(t)
	withJSONBody(c, http.MethodPost, `{"name":"Coach Email Stu","email":"coach.stu@example.com"}`)

	f.Coach.CreateStudent(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var id int
	_ = f.DB.QueryRow(`SELECT id FROM students WHERE name='Coach Email Stu'`).Scan(&id)
	if got := studentEmail(t, f, id); !got.Valid || got.String != "coach.stu@example.com" {
		t.Fatalf("email = %v, want coach.stu@example.com", got)
	}
}

// TestAssignmentResultsExposeReleaseState covers the two flags the Release
// button is driven by.
func TestAssignmentResultsExposeReleaseState(t *testing.T) {
	f := newHandlerFixture(t)

	t.Run("computed but not released", func(t *testing.T) {
		g := setupReleaseHandler(t, f)
		c, w := f.ctx(t, "admin")
		withAssignmentResultsParams(c, g.studentID, g.assignmentID)

		f.Admin.GetAssignmentResults(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, `"result_computed":true`) {
			t.Errorf("expected result_computed=true, body=%s", body)
		}
		if !strings.Contains(body, `"released_at":null`) {
			t.Errorf("expected released_at=null, body=%s", body)
		}
	})

	t.Run("released", func(t *testing.T) {
		g := setupReleaseHandler(t, f)
		rc, rw := f.ctx(t, "admin")
		releaseAs(rc, rw, f, "admin", f.AdminUserID, g.studentID, g.assignmentID)
		if rw.Code != http.StatusOK {
			t.Fatalf("release: code=%d body=%s", rw.Code, rw.Body.String())
		}

		c, w := f.ctx(t, "admin")
		withAssignmentResultsParams(c, g.studentID, g.assignmentID)
		f.Admin.GetAssignmentResults(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, `"result_computed":true`) {
			t.Errorf("expected result_computed=true, body=%s", body)
		}
		if strings.Contains(body, `"released_at":null`) {
			t.Errorf("released_at should be set, body=%s", body)
		}
	})

	// No attempt at all: both flags false so the button renders disabled rather
	// than throwing on a missing key.
	t.Run("no attempt", func(t *testing.T) {
		studentID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "na"), "No Attempt")
		assignmentID := testutil.CreateAssignment(t, f.DB, studentID, f.TestID, f.CoachID)

		c, w := f.ctx(t, "admin")
		withAssignmentResultsParams(c, studentID, assignmentID)
		f.Admin.GetAssignmentResults(c)
		if w.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, `"result_computed":false`) {
			t.Errorf("expected result_computed=false, body=%s", body)
		}
	})

	// A computed result scoring zero must still count as computed, otherwise the
	// students who scored nothing could never have their results released.
	t.Run("zero score is still computed", func(t *testing.T) {
		studentID := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "zs"), "Zero Scorer")
		assignmentID := testutil.CreateAssignment(t, f.DB, studentID, f.TestID, f.CoachID)
		attemptID := testutil.CreateAttempt(t, f.DB, assignmentID)
		testutil.CreateAttemptResult(t, f.DB, attemptID, 0)

		computed, releasedAt, err := f.AttemptRepo.GetResultReleaseState(attemptID)
		if err != nil {
			t.Fatalf("GetResultReleaseState: %v", err)
		}
		if !computed {
			t.Error("a zero-score result row must report computed=true")
		}
		if releasedAt.Valid {
			t.Error("released_at must be unset before release")
		}
	})
}

// GetResultReleaseState must distinguish "no result row" from "released".
// The list endpoint must return email, otherwise the edit form renders a blank
// field and saving posts an empty string - silently erasing the address the
// coach just entered. This is the data-loss path, not a cosmetic gap.
func TestListStudentsReturnsEmail(t *testing.T) {
	f := newHandlerFixture(t)
	want := "listed@example.com"
	stu := testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "ls"), "Listed")
	if _, err := f.DB.Exec(`UPDATE students SET email=$1 WHERE id=$2`, want, stu); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// A second student with no address proves NULL serializes rather than
	// breaking the scan.
	testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "nl"), "No Email Listed")

	c, w := f.ctx(t, "admin")
	c.Request = httptest.NewRequest(http.MethodGet, "/students", nil)
	f.Admin.ListStudents(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []struct {
			Name  string  `json:"name"`
			Email *string `json:"email"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var sawWith, sawWithout bool
	for _, s := range resp.Data {
		switch s.Name {
		case "Listed":
			if s.Email == nil || *s.Email != want {
				t.Errorf("Listed email = %v, want %q", s.Email, want)
			}
			sawWith = true
		case "No Email Listed":
			if s.Email != nil {
				t.Errorf("No Email Listed email = %q, want null", *s.Email)
			}
			sawWithout = true
		}
	}
	if !sawWith || !sawWithout {
		t.Fatalf("expected both students in the list (with=%v without=%v)", sawWith, sawWithout)
	}
}

func TestGetResultReleaseStateNoResultRow(t *testing.T) {
	f := newHandlerFixture(t)
	assignmentID := testutil.CreateAssignment(t, f.DB,
		testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, testutil.UniqueCode(t, "nr"), "No Result"),
		f.TestID, f.CoachID)
	attemptID := testutil.CreateAttempt(t, f.DB, assignmentID)

	computed, releasedAt, err := f.AttemptRepo.GetResultReleaseState(attemptID)
	if err != nil {
		t.Fatalf("GetResultReleaseState: %v", err)
	}
	if computed {
		t.Error("computed must be false when no attempt_results row exists")
	}
	if releasedAt.Valid {
		t.Error("releasedAt must be invalid when no result row exists")
	}
}
