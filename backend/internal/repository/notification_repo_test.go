package repository

import (
	"database/sql"
	"encoding/json"
	"testing"
)

// notifSetup creates an isolated tenant and returns its id. Tests should defer
// deleting the tenant so the FK cascade cleans notifications + users + prefs.
func notifSetup(t *testing.T, db *sql.DB) int {
	var tid int
	if err := db.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, "notif-test-"+t.Name()).Scan(&tid); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tid
}

// Test 3.14: unread count accuracy (per-user + NULL rows + cross-tenant isolation)
func TestNotificationUnreadCount(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	tid := notifSetup(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	ur := NewUserRepo(db)
	uidA, err := ur.Create(tid, "notif_a@test.local", "x", "admin")
	if err != nil {
		t.Fatalf("create user A: %v", err)
	}
	if _, err := ur.Create(tid, "notif_b@test.local", "x", "coach"); err != nil {
		t.Fatalf("create user B: %v", err)
	}

	otherTid, err := ur.CreateTenant("notif-other")
	if err != nil {
		t.Fatalf("create other tenant: %v", err)
	}
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, otherTid)
	otherUID, err := ur.Create(otherTid, "notif_other@test.local", "x", "admin")
	if err != nil {
		t.Fatalf("create other user: %v", err)
	}

	nr := NewNotificationRepo(db)
	// 3 unread for uidA: 2 targeted + 1 NULL (NULL rows count for any user)
	if _, err := nr.Create(NotificationRow{TenantID: tid, UserID: &uidA, EventType: "exam_submitted", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	if _, err := nr.Create(NotificationRow{TenantID: tid, UserID: &uidA, EventType: "coach_activity", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	if _, err := nr.Create(NotificationRow{TenantID: tid, UserID: nil, EventType: "system_alert", Title: "t", Message: "m", Priority: "warning", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	// 1 unread for the other tenant (must NOT be counted for uidA)
	if _, err := nr.Create(NotificationRow{TenantID: otherTid, UserID: &otherUID, EventType: "exam_submitted", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}

	count, err := nr.UnreadCount(tid, &uidA)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected unread count 3 for uidA, got %d", count)
	}

	// mark one of uidA's notifications read
	rows, _, err := nr.List(tid, &uidA, "", false, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected rows for uidA")
	}
	if err := nr.MarkRead(rows[0].ID, tid); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	count, _ = nr.UnreadCount(tid, &uidA)
	if count != 2 {
		t.Fatalf("expected unread count 2 after mark read, got %d", count)
	}

	// tenant-scoped count (no userID): after marking one of uidA's read, tid has
	// 2 unread (uidA's remaining row + the NULL row).
	total, err := nr.UnreadCount(tid, nil)
	if err != nil {
		t.Fatalf("UnreadCount nil: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected tenant-scoped unread 2, got %d", total)
	}
}

// Test 3.13: mark read / mark all read / delete
func TestNotificationMarkReadAndDelete(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	tid := notifSetup(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	ur := NewUserRepo(db)
	uid, err := ur.Create(tid, "notif_c@test.local", "x", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	nr := NewNotificationRepo(db)
	id1, err := nr.Create(NotificationRow{TenantID: tid, UserID: &uid, EventType: "exam_submitted", Title: "t1", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id2, err := nr.Create(NotificationRow{TenantID: tid, UserID: &uid, EventType: "coach_activity", Title: "t2", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// MarkRead single
	if err := nr.MarkRead(id1, tid); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	row, err := nr.GetByID(id1, tid)
	if err != nil || row == nil {
		t.Fatalf("GetByID: %v row=%v", err, row)
	}
	if row.ReadAt == nil {
		t.Fatalf("expected ReadAt to be set after MarkRead")
	}

	// MarkAllRead for the user (only their targeted + NULL rows)
	if err := nr.MarkAllRead(tid, &uid); err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	count, _ := nr.UnreadCount(tid, &uid)
	if count != 0 {
		t.Fatalf("expected 0 unread after MarkAllRead, got %d", count)
	}

	// Delete
	if err := nr.Delete(id1, tid); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	row, _ = nr.GetByID(id1, tid)
	if row != nil {
		t.Fatalf("expected nil after delete")
	}
	_ = id2
}

// Test 3.12: preferences respected
func TestNotificationPreferences(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	tid := notifSetup(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	ur := NewUserRepo(db)
	uid, err := ur.Create(tid, "notif_d@test.local", "x", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	nr := NewNotificationRepo(db)

	// Default (no row) => enabled true
	enabled, err := nr.IsEventEnabled(uid, "exam_submitted")
	if err != nil {
		t.Fatalf("IsEventEnabled default: %v", err)
	}
	if !enabled {
		t.Fatalf("expected default-on preference, got disabled")
	}

	// Disable exam_submitted
	if err := nr.UpdatePreferences(uid, map[string]bool{"exam_submitted": false, "coach_activity": true}); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}
	enabled, err = nr.IsEventEnabled(uid, "exam_submitted")
	if err != nil {
		t.Fatalf("IsEventEnabled after disable: %v", err)
	}
	if enabled {
		t.Fatalf("expected exam_submitted disabled, got enabled")
	}
	enabled, _ = nr.IsEventEnabled(uid, "coach_activity")
	if !enabled {
		t.Fatalf("expected coach_activity enabled")
	}

	prefs, err := nr.GetPreferences(uid)
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	if len(prefs) < 2 {
		t.Fatalf("expected >=2 preference rows, got %d", len(prefs))
	}
}
