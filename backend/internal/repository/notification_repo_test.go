package repository

import (
	"database/sql"
	"encoding/json"
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

func notifIntPtr(v int) *int { return &v }

func notifTestUser(t *testing.T, db *sql.DB, tenantID int, email, role string) int {
	t.Helper()
	uid, err := NewUserRepo(db).Create(tenantID, email, "x", role)
	if err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return uid
}

func notifTestSuperAdmin(t *testing.T, db *sql.DB, email string) int {
	t.Helper()
	var uid int
	if err := db.QueryRow(`INSERT INTO users (tenant_id, email, password, role)
		VALUES (NULL, $1, 'x', 'super_admin') RETURNING id`, email).Scan(&uid); err != nil {
		t.Fatalf("create super admin: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM users WHERE id = $1`, uid)
	})
	return uid
}

// notifSetup creates an isolated tenant and returns its id. Tests should defer
// deleting the tenant so the FK cascade cleans notifications + users + prefs.
func notifSetup(t *testing.T, db *sql.DB) int {
	var tid int
	if err := db.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, "notif-test-"+t.Name()).Scan(&tid); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tid
}

// Test 3.14: unread count accuracy (per-user + tenant scope + cross-tenant isolation)
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
	if _, err := nr.Create(NotificationRow{TenantID: notifIntPtr(tid), UserID: &uidA, EventType: "exam_submitted", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	if _, err := nr.Create(NotificationRow{TenantID: notifIntPtr(tid), UserID: &uidA, EventType: "coach_activity", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	// A second user in the SAME tenant, to prove MarkRead/Delete are scoped to
	// the owner rather than just the tenant.
	uidB, err := ur.Create(tid, "notif_a2@test.local", "x", "admin")
	if err != nil {
		t.Fatalf("create second user: %v", err)
	}
	// 1 unread for the other tenant (must NOT be counted for uidA)
	if _, err := nr.Create(NotificationRow{TenantID: notifIntPtr(otherTid), UserID: &otherUID, EventType: "exam_submitted", Title: "t", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")}); err != nil {
		t.Fatalf("create notif: %v", err)
	}
	// A row owned by uidB in the same tenant: the isolation case. uidA must never
	// see it under ScopeOwn.
	uidBRow, err := nr.Create(NotificationRow{TenantID: notifIntPtr(tid), UserID: &uidB, EventType: "exam_submitted", Title: "uidB only", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")})
	if err != nil {
		t.Fatalf("create uidB notif: %v", err)
	}

	count, err := nr.UnreadCount(tid, &uidA, ScopeOwn)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected unread count 2 for uidA, got %d", count)
	}

	// ScopeOwn must exclude uidB's row.
	ownRows, _, err := nr.List(tid, &uidA, ScopeOwn, "", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List ScopeOwn: %v", err)
	}
	if len(ownRows) != 2 {
		t.Fatalf("ScopeOwn returned %d rows, want 2", len(ownRows))
	}
	for _, r := range ownRows {
		if r.UserID == nil || *r.UserID != uidA {
			t.Fatalf("ScopeOwn surfaced another user's row: %+v", r.UserID)
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
	// Tenant-wide unread covers uidA's 2 + uidB's 1.
	allUnread, err := nr.UnreadCount(tid, &uidA, ScopeTenant)
	if err != nil {
		t.Fatalf("UnreadCount ScopeTenant: %v", err)
	}
	if allUnread != 3 {
		t.Fatalf("tenant-scoped unread=%d, want 3 (2 uidA + 1 uidB)", allUnread)
	}
	if allTotal != 3 {
		t.Fatalf("tenant-scoped total=%d, want 3", allTotal)
	}

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
	// Even the wide ScopeTenant listing must not widen write access.
	if marked, err := nr.MarkReadScoped(uidBRow, &tid, uidA, ScopeTenant); err != nil {
		t.Fatalf("MarkReadScoped ScopeTenant: %v", err)
	} else if marked {
		t.Fatalf("ScopeTenant let uidA mark uidB's row read")
	}
	stored, err := nr.GetByID(uidARow, tid)
	if err != nil || stored == nil {
		t.Fatalf("GetByID: %v row=%v", err, stored)
	}
	if stored.ReadAt == nil || stored.SeenAt == nil {
		t.Fatalf("MarkRead must set both read_at and seen_at: read=%v seen=%v", stored.ReadAt, stored.SeenAt)
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
		if _, err := nr.Create(NotificationRow{TenantID: notifIntPtr(tid), UserID: &uid, EventType: "exam_submitted", Title: p, Message: "m", Priority: p, Metadata: json.RawMessage("{}")}); err != nil {
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
	id1, err := nr.Create(NotificationRow{TenantID: notifIntPtr(tid), UserID: &uid, EventType: "exam_submitted", Title: "t1", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id2, err := nr.Create(NotificationRow{TenantID: notifIntPtr(tid), UserID: &uid, EventType: "coach_activity", Title: "t2", Message: "m", Priority: "info", Metadata: json.RawMessage("{}")})
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
	if row.ReadAt == nil || row.SeenAt == nil {
		t.Fatalf("expected ReadAt and SeenAt to be set after MarkRead: read=%v seen=%v", row.ReadAt, row.SeenAt)
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

func notifBoolPtr(v bool) *bool { return &v }

// Seen clears the bell badge without removing the row from the Unread tab.
func TestNotificationSeenAndReadFilters(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := notifSetup(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)
	uid := notifTestUser(t, db, tid, "notif_states@test.local", "admin")

	nr := NewNotificationRepo(db)
	create := func(title string) int {
		id, err := nr.Create(NotificationRow{
			TenantID: notifIntPtr(tid), UserID: &uid,
			EventType: "exam_submitted", Title: title, Message: "m",
			Priority: "info", Metadata: json.RawMessage("{}"),
		})
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		return id
	}
	unseenID := create("unseen")
	seenID := create("seen")
	readID := create("read")

	if ok, err := nr.MarkSeenScoped(seenID, &tid, uid, ScopeOwn); err != nil || !ok {
		t.Fatalf("MarkSeen: ok=%v err=%v", ok, err)
	}
	if ok, err := nr.MarkReadScoped(readID, &tid, uid, ScopeOwn); err != nil || !ok {
		t.Fatalf("MarkRead: ok=%v err=%v", ok, err)
	}

	counts, err := nr.Counts(&tid, uid, ScopeOwn)
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if counts.Unread != 2 || counts.Unseen != 1 {
		t.Fatalf("counts = unread:%d unseen:%d, want unread:2 unseen:1", counts.Unread, counts.Unseen)
	}

	list := func(read, seen *bool) []NotificationRow {
		t.Helper()
		rows, _, err := nr.ListFiltered(NotificationFilter{
			TenantID: &tid, UserID: uid, Scope: ScopeOwn,
			Read: read, Seen: seen, Limit: 50,
		})
		if err != nil {
			t.Fatalf("ListFiltered: %v", err)
		}
		return rows
	}
	if got := len(list(notifBoolPtr(true), nil)); got != 1 {
		t.Fatalf("read filter returned %d rows, want 1", got)
	}
	if got := len(list(notifBoolPtr(false), nil)); got != 2 {
		t.Fatalf("unread filter returned %d rows, want 2", got)
	}
	if got := len(list(nil, notifBoolPtr(true))); got != 2 {
		t.Fatalf("seen filter returned %d rows, want 2", got)
	}
	if got := len(list(nil, notifBoolPtr(false))); got != 1 {
		t.Fatalf("unseen filter returned %d rows, want 1", got)
	}

	if err := nr.MarkAllSeenScoped(&tid, uid, ScopeOwn); err != nil {
		t.Fatalf("MarkAllSeen: %v", err)
	}
	counts, err = nr.Counts(&tid, uid, ScopeOwn)
	if err != nil {
		t.Fatalf("Counts after mark all seen: %v", err)
	}
	if counts.Unseen != 0 || counts.Unread != 2 {
		t.Fatalf("counts after mark all seen = unread:%d unseen:%d, want unread:2 unseen:0", counts.Unread, counts.Unseen)
	}
	_ = unseenID
}

func TestNotificationCreateIdempotent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := notifSetup(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)
	uidA := notifTestUser(t, db, tid, "notif_idem_a@test.local", "admin")
	uidB := notifTestUser(t, db, tid, "notif_idem_b@test.local", "coach")

	nr := NewNotificationRepo(db)
	key := "exam_submitted:attempt:123"
	row := NotificationRow{
		TenantID: notifIntPtr(tid), UserID: &uidA,
		EventType: "exam_submitted", Title: "Submitted", Message: "m",
		Priority: "info", Metadata: json.RawMessage("{}"), EventKey: &key,
	}

	id, created, err := nr.CreateIdempotent(row)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if !created || id == 0 {
		t.Fatalf("first create: created=%v id=%d, want created=true and a row id", created, id)
	}

	againID, created, err := nr.CreateIdempotent(row)
	if err != nil {
		t.Fatalf("duplicate create: %v", err)
	}
	if created || againID != 0 {
		t.Fatalf("duplicate create: created=%v id=%d, want created=false id=0", created, againID)
	}

	row.UserID = &uidB
	otherID, created, err := nr.CreateIdempotent(row)
	if err != nil {
		t.Fatalf("create for second user: %v", err)
	}
	if !created || otherID == 0 {
		t.Fatalf("second user must get their own row: created=%v id=%d", created, otherID)
	}

	rows, _, err := nr.List(tid, &uidA, ScopeOwn, "", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("uidA has %d rows, want 1", len(rows))
	}
	if rows[0].EventKey == nil || *rows[0].EventKey != key {
		t.Fatalf("event_key did not round-trip: %v", rows[0].EventKey)
	}
}

func TestNotificationPlatformScopeIsolation(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := notifSetup(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)
	uid := notifTestUser(t, db, tid, "notif_platform_admin@test.local", "admin")
	superID := notifTestSuperAdmin(t, db, "notif_platform_super@test.local")

	nr := NewNotificationRepo(db)
	tenantID, _, err := nr.CreateIdempotent(NotificationRow{
		TenantID: notifIntPtr(tid), UserID: &uid,
		EventType: "exam_submitted", Title: "Tenant row", Message: "m",
		Priority: "info", Metadata: json.RawMessage("{}"),
	})
	if err != nil {
		t.Fatalf("create tenant row: %v", err)
	}
	platformKey := "tenant_created:1"
	platformID, _, err := nr.CreateIdempotent(NotificationRow{
		TenantID: nil, UserID: &superID, Scope: "platform",
		EventType: "tenant_created", Title: "Platform row", Message: "m",
		Priority: "info", Metadata: json.RawMessage("{}"), EventKey: &platformKey,
	})
	if err != nil {
		t.Fatalf("create platform row: %v", err)
	}

	platformRows, total, err := nr.ListFiltered(NotificationFilter{
		UserID: superID, Scope: ScopePlatform, Limit: 50,
	})
	if err != nil {
		t.Fatalf("platform list: %v", err)
	}
	if total != 1 || len(platformRows) != 1 || platformRows[0].TenantID != nil {
		t.Fatalf("platform list = %d rows of %d, want exactly 1 with tenant_id NULL", len(platformRows), total)
	}

	tenantRows, _, err := nr.ListFiltered(NotificationFilter{
		TenantID: &tid, UserID: uid, Scope: ScopeOwn, Limit: 50,
	})
	if err != nil {
		t.Fatalf("tenant list: %v", err)
	}
	if len(tenantRows) != 1 {
		t.Fatalf("tenant admin sees %d rows, want 1", len(tenantRows))
	}

	if ok, err := nr.MarkReadScoped(platformID, &tid, uid, ScopeOwn); err != nil {
		t.Fatalf("tenant action on platform row: %v", err)
	} else if ok {
		t.Fatal("tenant user marked a platform notification read")
	}
	if ok, err := nr.MarkReadScoped(platformID, nil, superID, ScopePlatform); err != nil || !ok {
		t.Fatalf("platform MarkRead: ok=%v err=%v", ok, err)
	}
	if ok, err := nr.MarkSeenScoped(tenantID, &tid, uid, ScopeOwn); err != nil || !ok {
		t.Fatalf("tenant MarkSeen: ok=%v err=%v", ok, err)
	}

	if _, _, err := nr.CreateIdempotent(NotificationRow{
		UserID: &uid, EventType: "exam_submitted", Title: "bad", Message: "m",
		Priority: "info", Metadata: json.RawMessage("{}"),
	}); err == nil {
		t.Fatal("tenant notification without tenant_id must fail")
	}
	if _, _, err := nr.CreateIdempotent(NotificationRow{
		TenantID: notifIntPtr(tid), UserID: &uid, Scope: "platform",
		EventType: "tenant_created", Title: "bad", Message: "m",
		Priority: "info", Metadata: json.RawMessage("{}"),
	}); err == nil {
		t.Fatal("platform notification with tenant_id must fail")
	}
}
