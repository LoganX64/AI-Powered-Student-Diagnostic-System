package repository

import (
	"database/sql"
	"testing"
	"time"

	"ai-student-diagnostic/backend/internal/testutil"
)

func TestPasswordResetCreateFindValid(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewPasswordResetRepo(db)
	email := uniqueEmail(t, "reset")
	t.Cleanup(func() { db.Exec(`DELETE FROM password_resets WHERE email = $1`, email) })

	if err := r.Create(email, "hash-create-"+t.Name(), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := r.FindValid("hash-create-" + t.Name())
	if err != nil {
		t.Fatalf("FindValid: %v", err)
	}
	if got != email {
		t.Fatalf("FindValid = %q, want %q", got, email)
	}
}

func TestPasswordResetFindValidExpired(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewPasswordResetRepo(db)
	email := uniqueEmail(t, "reset-expired")
	t.Cleanup(func() { db.Exec(`DELETE FROM password_resets WHERE email = $1`, email) })

	if err := r.Create(email, "hash-expired-"+t.Name(), time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := r.FindValid("hash-expired-" + t.Name()); err != sql.ErrNoRows {
		t.Fatalf("FindValid expired: got %v, want sql.ErrNoRows", err)
	}
}

func TestPasswordResetMarkUsedInvalidates(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewPasswordResetRepo(db)
	email := uniqueEmail(t, "reset-used")
	hash := "hash-used-" + t.Name()
	t.Cleanup(func() { db.Exec(`DELETE FROM password_resets WHERE email = $1`, email) })

	if err := r.Create(email, hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := r.FindValid(hash); err != nil {
		t.Fatalf("FindValid pre-MarkUsed: %v", err)
	}
	if err := r.MarkUsed(hash); err != nil {
		t.Fatalf("MarkUsed: %v", err)
	}
	if _, err := r.FindValid(hash); err != sql.ErrNoRows {
		t.Fatalf("FindValid after MarkUsed: got %v, want sql.ErrNoRows", err)
	}
}

func TestPasswordResetFindValidUnknownHash(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewPasswordResetRepo(db)
	if _, err := r.FindValid("hash-that-does-not-exist-xyz"); err != sql.ErrNoRows {
		t.Fatalf("got %v, want sql.ErrNoRows", err)
	}
}
