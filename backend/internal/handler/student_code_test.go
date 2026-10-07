package handlers

import (
	"database/sql"
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

func TestGenerateStudentCodeFormat(t *testing.T) {
	tenantID := 42
	code := generateStudentCode(tenantID)
	if !strings.HasPrefix(code, "T42") {
		t.Fatalf("code=%q must start with T{tenantID}", code)
	}
	suffix := strings.TrimPrefix(code, "T42")
	if len(suffix) != 6 {
		t.Fatalf("suffix=%q must be 6 chars", suffix)
	}
	for _, r := range suffix {
		if !strings.ContainsRune(studentCodeChars, r) {
			t.Fatalf("char %q not in %q", r, studentCodeChars)
		}
	}
}

func TestGenerateStudentCodeVaries(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		c := generateStudentCode(7)
		if seen[c] {
			t.Fatalf("duplicate code %q within 50 draws", c)
		}
		seen[c] = true
	}
}

func TestEnsureStudentCodeUsesProvidedCode(t *testing.T) {
	f := newHandlerFixture(t)
	provided := testutil.UniqueCode(t, "given")

	id, code, err := ensureStudentCode(f.StudentRepo, f.TenantID, "Provided", provided, f.CoachID)
	if err != nil {
		t.Fatalf("ensureStudentCode: %v", err)
	}
	if code != provided {
		t.Fatalf("code=%q, want provided %q", code, provided)
	}
	got, err := f.StudentRepo.GetName(id, f.TenantID)
	if err != nil || got != "Provided" {
		t.Fatalf("GetName=(%q,%v)", got, err)
	}
}

func TestEnsureStudentCodeAutoGenerates(t *testing.T) {
	f := newHandlerFixture(t)

	id, code, err := ensureStudentCode(f.StudentRepo, f.TenantID, "Auto", "", f.CoachID)
	if err != nil {
		t.Fatalf("ensureStudentCode: %v", err)
	}
	if !strings.HasPrefix(code, "T") {
		t.Fatalf("auto code %q must be prefixed with the tenant id", code)
	}
	if id == 0 {
		t.Fatal("no student id returned")
	}
}

// TestEnsureStudentCodeRetriesOnDuplicate forces the generator to return an
// already-taken code first, so ensureStudentCode hits the 23505 unique
// violation, clears the code, and retries with a fresh one instead of surfacing
// the database error.
func TestEnsureStudentCodeRetriesOnDuplicate(t *testing.T) {
	f := newHandlerFixture(t)

	taken := testutil.UniqueCode(t, "taken")
	testutil.CreateStudent(t, db(t, f), f.TenantID, f.CoachID, taken, "Seed")

	calls := 0
	orig := generateStudentCode
	t.Cleanup(func() { generateStudentCode = orig })
	generateStudentCode = func(tenantID int) string {
		calls++
		if calls == 1 {
			return taken // collide on the first attempt
		}
		return testutil.UniqueCode(t, "fresh")
	}

	id, code, err := ensureStudentCode(f.StudentRepo, f.TenantID, "Retry", "", f.CoachID)
	if err != nil {
		t.Fatalf("ensureStudentCode must retry past the collision, got %v", err)
	}
	if id == 0 {
		t.Fatal("no student id returned")
	}
	if calls < 2 {
		t.Fatalf("generator called %d time(s); the 23505 retry path was not exercised", calls)
	}
	if code == taken {
		t.Fatalf("returned the colliding code %q", code)
	}
	// The retry must have persisted a student under the fresh code.
	if _, _, err := f.StudentRepo.GetNameCode(id, f.TenantID); err != nil {
		t.Fatalf("retry did not persist the student: %v", err)
	}
}

// db exposes the fixture's pool for helpers that need raw SQL.
func db(t *testing.T, f *handlerFixture) *sql.DB {
	t.Helper()
	return f.DB
}
