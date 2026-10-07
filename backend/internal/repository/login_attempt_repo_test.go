package repository

import (
	"testing"
	"time"

	"ai-student-diagnostic/backend/internal/testutil"
)

func TestIsLockedUnknownIdentifier(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewLoginAttemptRepo(db)
	locked, err := r.IsLocked("nobody-" + t.Name())
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("unknown identifier must not be locked")
	}
}

func TestRecordFailureLocksAfterMax(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewLoginAttemptRepo(db)
	id := "lock-" + t.Name() + "@fixtures.local"
	t.Cleanup(func() { db.Exec(`DELETE FROM login_attempts WHERE account_identifier = $1`, id) })

	for i := 0; i < maxLoginFailures-1; i++ {
		if err := r.RecordFailure(id); err != nil {
			t.Fatalf("RecordFailure #%d: %v", i+1, err)
		}
		locked, err := r.IsLocked(id)
		if err != nil {
			t.Fatalf("IsLocked: %v", err)
		}
		if locked {
			t.Fatalf("locked after %d failures, want lock at %d", i+1, maxLoginFailures)
		}
	}
	if err := r.RecordFailure(id); err != nil {
		t.Fatalf("RecordFailure final: %v", err)
	}
	locked, err := r.IsLocked(id)
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if !locked {
		t.Fatalf("expected locked after %d failures", maxLoginFailures)
	}
}

func TestResetClearsLock(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewLoginAttemptRepo(db)
	id := "reset-" + t.Name() + "@fixtures.local"
	t.Cleanup(func() { db.Exec(`DELETE FROM login_attempts WHERE account_identifier = $1`, id) })

	for i := 0; i < maxLoginFailures; i++ {
		r.RecordFailure(id)
	}
	if locked, _ := r.IsLocked(id); !locked {
		t.Fatal("precondition: should be locked")
	}
	if err := r.Reset(id); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	locked, err := r.IsLocked(id)
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("Reset must clear lock")
	}
}

func TestResetOnUnknownIdentifierStillWorks(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewLoginAttemptRepo(db)
	if err := r.Reset("never-seen-" + t.Name()); err != nil {
		t.Fatalf("Reset unknown: %v", err)
	}
}

func TestLockedUntilIsInFuture(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewLoginAttemptRepo(db)
	id := "future-" + t.Name() + "@fixtures.local"
	t.Cleanup(func() { db.Exec(`DELETE FROM login_attempts WHERE account_identifier = $1`, id) })

	for i := 0; i < maxLoginFailures; i++ {
		r.RecordFailure(id)
	}
	var lockedUntil time.Time
	err := db.QueryRow(`SELECT locked_until FROM login_attempts WHERE account_identifier = $1`, id).Scan(&lockedUntil)
	if err != nil {
		t.Fatalf("query locked_until: %v", err)
	}
	if !lockedUntil.After(time.Now()) {
		t.Fatalf("locked_until %v should be in the future", lockedUntil)
	}
}
