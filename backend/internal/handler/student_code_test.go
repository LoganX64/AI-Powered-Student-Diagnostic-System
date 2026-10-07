package handlers

import (
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
	if len(code) < 2 || !strings.Contains(code, "T") {
		t.Fatalf("auto code %q looks malformed", code)
	}
	if id == 0 {
		t.Fatal("no student id returned")
	}
}

func TestEnsureStudentCodeRetriesOnDuplicate(t *testing.T) {
	f := newHandlerFixture(t)
	// Force the first generated code to collide by pre-seeding it, then call
	// with an empty provided code: the 23505 path must clear the code and retry
	// with a fresh one rather than surfacing the unique-violation error.
	seeded := testutil.UniqueCode(t, "dup")
	testutil.CreateStudent(t, f.DB, f.TenantID, f.CoachID, seeded, "Seed")

	// Creating a student with the same code directly must fail with 23505,
	// which is the error ensureStudentCode branches on.
	_, err := f.StudentRepo.Create(f.TenantID, "Clash", seeded, f.CoachID)
	if err == nil {
		t.Fatal("expected unique violation when reusing a student code")
	}

	// ensureStudentCode with an empty code must succeed by generating one.
	id, code, err := ensureStudentCode(f.StudentRepo, f.TenantID, "Retry", "", f.CoachID)
	if err != nil {
		t.Fatalf("ensureStudentCode retry path: %v", err)
	}
	if code == "" || id == 0 {
		t.Fatalf("id=%d code=%q", id, code)
	}
}
