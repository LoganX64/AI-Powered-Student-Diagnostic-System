package repository

import (
	"database/sql"
	"encoding/json"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
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
	db := testutil.OpenTestDB(t)
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
	// 2 unread owned by uidA.
	if _, err := nr.Create(NotificationRow{TenantID: tid, UserID: &uidA, EventType: "exam_submitted", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	if _, err := nr.Create(NotificationRow{TenantID: tid, UserID: &uidA, EventType: "coach_activity", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	// A broadcast row, which ScopeOwn must NOT surface: read_at is a single
	// column shared by the whole org, so it cannot carry per-user read state.
	if _, err := nr.Create(NotificationRow{TenantID: tid, UserID: nil, EventType: "system_alert", Title: "t", Message: "m", Priority: "warning", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	// A second user in the SAME tenant, to prove MarkRead/Delete are scoped to
	// the owner rather than just the tenant.
	uidB, err := ur.Create(tid, "notif_a2@test.local", "x", "admin")
	if err != nil {
		t.Fatalf("create second user: %v", err)
	}
	// 1 unread for the other tenant (must NOT be counted for uidA)
	if _, err := nr.Create(NotificationRow{TenantID: otherTid, UserID: &otherUID, EventType: "exam_submitted", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	// A row owned by uidB in the same tenant: the isolation case. uidA must never
	// see it under ScopeOwn.
	if _, err := nr.Create(NotificationRow{TenantID: tid, UserID: &uidB, EventType: "exam_submitted", Title: "uidB only", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create uidB notif: %v", err)
	}

	count, err := nr.UnreadCount(tid, &uidA, ScopeOwn)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected unread count 2 for uidA, got %d", count)
	}

	// ScopeOwn must exclude both the broadcast row and uidB's row.
	ownRows, _, err := nr.List(tid, &uidA, ScopeOwn, "", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List ScopeOwn: %v", err)
	}
	for _, r := range ownRows {
		if r.UserID == nil {
			t.Fatal("ScopeOwn surfaced a broadcast row")
		}
		if *r.UserID != uidA {
			t.Fatalf("ScopeOwn surfaced another user's row (user_id=%d)", *r.UserID)
		}
	}

	// ScopeTenant (the admin's view) does include uidB's row, so oversight works.
	allRows, allTotal, err := nr.List(tid, &uidA, ScopeTenant, "", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List ScopeTenant: %v", err)
	}
	sawUIDB := false
	for _, r := range allRows {
		if r.UserID != nil && *r.UserID == uidB && r.Title == "uidB only" {
			sawUIDB = true
		}
	}
	if !sawUIDB {
		t.Fatal("ScopeTenant hid another user's row from the admin")
	}
	// Tenant-wide unread covers uidA's 2 + uidB's 1 + the broadcast row.
	allUnread, err := nr.UnreadCount(tid, &uidA, ScopeTenant)
	if err != nil {
		t.Fatalf("UnreadCount ScopeTenant: %v", err)
	}
	if allUnread != 4 {
		t.Fatalf("tenant-scoped unread=%d, want 4 (2 uidA + 1 uidB + 1 broadcast)", allUnread)
	}
	_ = allTotal

	// mark one of uidA's notifications read
	rows, _, err := nr.List(tid, &uidA, ScopeOwn, "", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected rows for uidA")
	}
	// Every row ScopeOwn returned is uidA's by construction, so rows[0] is safe.
	uidARow := rows[0].ID
	if uidARow == 0 {
		t.Fatalf("no row owned by uidA in the listing")
	}
	marked, err := nr.MarkRead(uidARow, tid, uidA)
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if !marked {
		t.Fatalf("MarkRead reported no row updated for uidA's own notification")
	}
	// Another user in the same tenant must not be able to touch it.
	if marked, err := nr.MarkRead(uidARow, tid, uidB); err != nil {
		t.Fatalf("MarkRead as uidB: %v", err)
	} else if marked {
		t.Fatalf("MarkRead let uidB mark uidA's notification read")
	}
	// A broadcast row must be refused outright: read_at is one column, so one
	// user marking an org-wide notification read would hide it for everyone.
	broadcastRow, err := nr.Create(NotificationRow{TenantID: tid, UserID: nil, EventType: "system_alert", Title: "broadcast2", Message: "m", Priority: "warning", Metadata: json.RawMessage("{}")})
	if err != nil {
		t.Fatalf("create broadcast: %v", err)
	}
	if marked, err := nr.MarkRead(broadcastRow, tid, uidA); err != nil {
		t.Fatalf("MarkRead on broadcast: %v", err)
	} else if marked {
		t.Fatalf("MarkRead let a single user mark a broadcast notification read")
	}
	count, _ = nr.UnreadCount(tid, &uidA, ScopeOwn)
	if count != 1 {
		t.Fatalf("expected unread count 1 after mark read, got %d", count)
	}
}

// The priority filter backs the UI's Info/Warning/Alert tabs. Both the returned
// rows and the total must be scoped to it, because the pager renders
// "Showing X–Y of <total>" — a total that ignored the filter would make the page
// claim more rows than the tab can ever produce.
func TestNotificationPriorityFilter(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := notifSetup(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	ur := NewUserRepo(db)
	uid, err := ur.Create(tid, "notif_prio@test.local", "x", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	nr := NewNotificationRepo(db)
	for _, p := range []string{"info", "info", "info", "warning", "alert"} {
		if _, err := nr.Create(NotificationRow{TenantID: tid, UserID: &uid, EventType: "exam_submitted", Title: p, Message: "m", Priority: p, Metadata: json.RawMessage("{}")}); err != nil {
			t.Fatalf("create %s row: %v", p, err)
		}
	}

	// An unfiltered page sees everything.
	_, allTotal, err := nr.List(tid, &uid, ScopeOwn, "", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List unfiltered: %v", err)
	}
	if allTotal != 5 {
		t.Fatalf("unfiltered total=%d, want 5", allTotal)
	}

	for _, tc := range []struct {
		priority string
		want     int
	}{
		{"info", 3},
		{"warning", 1},
		{"alert", 1},
		// No such severity: an empty priority is how the handler says "no filter",
		// so it must not be mistaken for a match-nothing filter.
		{"", 5},
	} {
		rows, total, err := nr.List(tid, &uid, ScopeOwn, "", tc.priority, false, 50, 0)
		if err != nil {
			t.Fatalf("List priority=%q: %v", tc.priority, err)
		}
		if total != tc.want {
			t.Fatalf("priority=%q total=%d, want %d", tc.priority, total, tc.want)
		}
		if len(rows) != tc.want {
			t.Fatalf("priority=%q returned %d rows, want %d", tc.priority, len(rows), tc.want)
		}
		for _, r := range rows {
			if tc.priority != "" && r.Priority != tc.priority {
				t.Fatalf("priority=%q leaked a %q row", tc.priority, r.Priority)
			}
		}
	}

	// A page within a filtered set must page the filtered rows, and its total
	// must still describe the whole filtered set.
	page1, total, err := nr.List(tid, &uid, ScopeOwn, "", "info", false, 2, 0)
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if len(page1) != 2 || total != 3 {
		t.Fatalf("page 1 of the info set: %d rows of total %d, want 2 of 3", len(page1), total)
	}
	page2, _, err := nr.List(tid, &uid, ScopeOwn, "", "info", false, 2, 2)
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page 2 of the info set: %d rows, want 1", len(page2))
	}
	seen := map[int]bool{}
	for _, r := range append(page1, page2...) {
		if seen[r.ID] {
			t.Fatalf("row %d appeared on both pages", r.ID)
		}
		seen[r.ID] = true
	}
}

// Test 3.13: mark read / mark all read / delete
func TestNotificationMarkReadAndDelete(t *testing.T) {
	db := testutil.OpenTestDB(t)
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
	marked, err := nr.MarkRead(id1, tid, uid)
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if !marked {
		t.Fatalf("MarkRead reported no row updated for the owner")
	}
	row, err := nr.GetByID(id1, tid)
	if err != nil || row == nil {
		t.Fatalf("GetByID: %v row=%v", err, row)
	}
	if row.ReadAt == nil {
		t.Fatalf("expected ReadAt to be set after MarkRead")
	}

	// MarkAllRead for the user (only their own rows)
	if err := nr.MarkAllRead(tid, &uid); err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	count, _ := nr.UnreadCount(tid, &uid, ScopeOwn)
	if count != 0 {
		t.Fatalf("expected 0 unread after MarkAllRead, got %d", count)
	}

	// Delete
	deleted, err := nr.Delete(id1, tid, uid)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Fatalf("Delete reported no row removed for the owner")
	}
	row, _ = nr.GetByID(id1, tid)
	if row != nil {
		t.Fatalf("expected nil after delete")
	}
	// A second delete must report nothing removed rather than claiming success.
	if again, err := nr.Delete(id1, tid, uid); err != nil {
		t.Fatalf("second Delete: %v", err)
	} else if again {
		t.Fatalf("second Delete reported a row removed")
	}
	_ = id2
}

// Test 3.12: preferences respected
func TestNotificationPreferences(t *testing.T) {
	db := testutil.OpenTestDB(t)
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
