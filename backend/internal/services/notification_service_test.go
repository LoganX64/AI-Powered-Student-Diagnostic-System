package services

import (
	"database/sql"
	"encoding/json"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
)

// setupNotifTenant creates an isolated tenant with an admin (enabled), an admin
// that has exam_submitted disabled, and a coach (enabled). Returns tenant id and
// the three user ids (A=enabled admin, B=disabled admin, C=coach).
func setupNotifTenant(t *testing.T, db *sql.DB) (tid int, a, b, c int) {
	if err := db.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, "svc-notif-"+t.Name()).Scan(&tid); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	ur := repository.NewUserRepo(db)
	var err error
	if a, err = ur.Create(tid, "svc_a@test.local", "x", "admin"); err != nil {
		t.Fatalf("create A: %v", err)
	}
	if b, err = ur.Create(tid, "svc_b@test.local", "x", "admin"); err != nil {
		t.Fatalf("create B: %v", err)
	}
	if c, err = ur.Create(tid, "svc_c@test.local", "x", "coach"); err != nil {
		t.Fatalf("create C: %v", err)
	}
	nr := repository.NewNotificationRepo(db)
	if err := nr.UpdatePreferences(b, map[string]bool{"exam_submitted": false}); err != nil {
		t.Fatalf("disable pref for B: %v", err)
	}
	return tid, a, b, c
}

// Test 3.11: notification created on exam submission (fan-out), respects disabled pref
func TestNotifyExamSubmittedFanout(t *testing.T) {
	db := svcTestDB(t)
	defer db.Close()
	tid, a, b, c := setupNotifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	if err := svc.NotifyExamSubmitted(tid, 42, 7, "Bob"); err != nil {
		t.Fatalf("NotifyExamSubmitted: %v", err)
	}

	rows, total, err := nr.List(tid, nil, "exam_submitted", false, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 fanned-out notifications, got %d", total)
	}
	got := map[int]bool{}
	for _, r := range rows {
		if r.UserID == nil {
			t.Fatalf("fan-out must create per-user rows, got user_id=NULL")
		}
		got[*r.UserID] = true
		// metadata must round-trip student_id / assignment_id
		var meta map[string]float64
		if err := json.Unmarshal(r.Metadata, &meta); err != nil {
			t.Fatalf("unmarshal metadata: %v", err)
		}
		if int(meta["student_id"]) != 42 || int(meta["assignment_id"]) != 7 {
			t.Fatalf("metadata mismatch: %v", meta)
		}
	}
	if !got[a] || !got[c] {
		t.Fatalf("expected notifications for enabled admin A and coach C, got %v", got)
	}
	if got[b] {
		t.Fatalf("disabled admin B should NOT receive exam_submitted notification")
	}
}

// Test 3.11 (supplementary): per-event priority for student-exam-logout is warning
func TestNotifyStudentExamLogoutPriority(t *testing.T) {
	db := svcTestDB(t)
	defer db.Close()
	tid, a, _, _ := setupNotifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	if err := svc.NotifyStudentExamLogout(tid, 42, 7, "Bob"); err != nil {
		t.Fatalf("NotifyStudentExamLogout: %v", err)
	}
	rows, _, err := nr.List(tid, &a, "student_exam_logout", false, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 notification for admin A, got %d", len(rows))
	}
	if rows[0].Priority != "warning" {
		t.Fatalf("expected priority warning, got %q", rows[0].Priority)
	}
}
