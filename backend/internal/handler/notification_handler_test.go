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
)

// notification_handler.go tests.
//
// The first version of this file existed to pin a missing authorization check:
// MarkRead/Delete filtered on id AND tenant_id and never on user_id, so any user
// in a tenant could act on another user's rows. Every mutation is now scoped to
// the caller (and admins no longer read tenant-wide), so the tests below assert
// refusal by name: TestNotificationMarkReadRefusesOtherUsers,
// TestNotificationDeleteRefusesOtherUsers and
// TestNotificationAdminActsOnlyOnOwnRows.

func TestNotificationListNotifications(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	unread := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "system_alert", "Primary Unread", false)
	read := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "coach_activity", "Primary Read", true)

	body, code := notifGet(t, f, "")
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	var resp struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	json.Unmarshal([]byte(body), &resp)
	if resp.Limit != 50 {
		t.Fatalf("limit=%d, want the ParsePagination default of 50", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Fatalf("offset=%d, want 0", resp.Offset)
	}
	if !strings.Contains(body, "Primary Unread") || !strings.Contains(body, "Primary Read") {
		t.Fatalf("own notifications missing: %s", body)
	}

	// event_type narrows.
	filtered, _ := notifGet(t, f, "event_type=system_alert")
	if !strings.Contains(filtered, "Primary Unread") {
		t.Fatalf("event_type=system_alert missed our row: %s", filtered)
	}
	if strings.Contains(filtered, "Primary Read") {
		t.Fatalf("event_type filter leaked the other event type: %s", filtered)
	}

	// unread=true drops the read row.
	unreadOnly, _ := notifGet(t, f, "unread=true")
	if !strings.Contains(unreadOnly, "Primary Unread") {
		t.Fatalf("unread=true missed the unread row: %s", unreadOnly)
	}
	if strings.Contains(unreadOnly, "Primary Read") {
		t.Fatalf("unread=true included an already-read row: %s", unreadOnly)
	}

	// Rows outside our tenant never appear.
	// The other tenant's only user is its coach, which is enough to own a
	// notification. Querying for an admin there returns nothing.
	otherUser := f.OtherCoachUserID
	foreign := testutil.CreateNotification(t, f.DB, f.OtherTenantID, &otherUser, "system_alert", "Foreign Notification", false)
	after, _ := notifGet(t, f, "")
	if strings.Contains(after, "Foreign Notification") {
		t.Fatalf("listing leaked another tenant's notification: %s", after)
	}
	// And the unread count must not move either.
	countAfter := notifUnreadCount(t, f)
	if countAfter != 1 {
		t.Fatalf("unread_count=%d, want 1 — the foreign unread row was counted", countAfter)
	}
	_ = read
	_ = unread
	_ = foreign
}

// An empty result must serialise as [], not null, or clients break on map.
func TestNotificationListEmptyIsArray(t *testing.T) {
	f := newHandlerFixture(t)
	// A user with no notifications at all.
	empty := testutil.CreateUser(t, f.DB, f.TenantID, testutil.UniqueEmail(t, "nonotify"), "admin")

	body, code := notifGetAs(t, f, empty, "")
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if !strings.Contains(body, `"data":[]`) {
		t.Fatalf("expected an empty array, got %s", body)
	}
}

func TestNotificationUnreadCount(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "system_alert", "A", false)
	testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "sqi_complete", "B", false)
	testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "storage_warning", "C", true)

	if got := notifUnreadCount(t, f); got != 2 {
		t.Fatalf("unread_count=%d, want 2", got)
	}
}

// The badge counts unseen rows, the Unread tab counts unread rows, and the two
// differ as soon as a row is seen without being read.
func TestNotificationUnreadAndUnseenCounts(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	testutil.CreateNotificationWithState(t, f.DB, &f.TenantID, "tenant", uid, "exam_submitted", "Read", true, true)
	testutil.CreateNotificationWithState(t, f.DB, &f.TenantID, "tenant", uid, "exam_submitted", "Seen Only", false, true)
	testutil.CreateNotificationWithState(t, f.DB, &f.TenantID, "tenant", uid, "exam_submitted", "Unseen", false, false)

	unread, unseen := notifUnreadCounts(t, f, "admin", uid)
	if unread != 2 {
		t.Fatalf("unread_count=%d, want 2 (seen-only and unseen rows)", unread)
	}
	if unseen != 1 {
		t.Fatalf("unseen_count=%d, want 1 (only the unseen row)", unseen)
	}
}

// read= and seen= filter independently; unread=true is sugar for read=false.
func TestNotificationListReadSeenFilters(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	testutil.CreateNotificationWithState(t, f.DB, &f.TenantID, "tenant", uid, "exam_submitted", "Unread Unseen", false, false)
	testutil.CreateNotificationWithState(t, f.DB, &f.TenantID, "tenant", uid, "exam_submitted", "Seen Only", false, true)
	testutil.CreateNotificationWithState(t, f.DB, &f.TenantID, "tenant", uid, "exam_submitted", "Read Row", true, true)

	cases := []struct {
		query   string
		want    []string
		notWant []string
	}{
		{"read=false", []string{"Unread Unseen", "Seen Only"}, []string{"Read Row"}},
		{"read=true", []string{"Read Row"}, []string{"Unread Unseen", "Seen Only"}},
		{"unread=true", []string{"Unread Unseen", "Seen Only"}, []string{"Read Row"}},
		{"seen=false", []string{"Unread Unseen"}, []string{"Seen Only", "Read Row"}},
		{"seen=true", []string{"Seen Only", "Read Row"}, []string{"Unread Unseen"}},
	}
	for _, tc := range cases {
		body, code := notifGet(t, f, "event_type=exam_submitted&"+tc.query)
		if code != http.StatusOK {
			t.Fatalf("%s code=%d body=%s", tc.query, code, body)
		}
		for _, want := range tc.want {
			if !strings.Contains(body, want) {
				t.Fatalf("%s missed %q: %s", tc.query, want, body)
			}
		}
		for _, notWant := range tc.notWant {
			if strings.Contains(body, notWant) {
				t.Fatalf("%s leaked %q: %s", tc.query, notWant, body)
			}
		}
	}
}

func TestNotificationMarkRead(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	id := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "system_alert", "To Read", false)

	_, code := notifMarkRead(t, f, f.AdminUserID, id)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	var readAt sql.NullString
	if err := f.DB.QueryRow(`SELECT read_at::text FROM notifications WHERE id = $1`, id).Scan(&readAt); err != nil {
		t.Fatalf("read read_at: %v", err)
	}
	if !readAt.Valid {
		t.Fatal("read_at still NULL after MarkRead")
	}
	if got := notifUnreadCount(t, f); got != 0 {
		t.Fatalf("unread_count=%d after marking the only row read, want 0", got)
	}
}

func TestNotificationMarkReadBadID(t *testing.T) {
	f := newHandlerFixture(t)

	bad, wBad := f.ctx(t, "admin")
	withParam(bad, "id", "abc")
	f.Notifications.MarkRead(bad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("bad id code=%d, want 400", wBad.Code)
	}

	// A valid-but-unknown id is not a client error: the repo filters on
	// id AND tenant_id, matches nothing, and reports no error.
	unknown, wUnknown := f.ctx(t, "admin")
	withParam(unknown, "id", "99999999")
	f.Notifications.MarkRead(unknown)
	t.Logf("unknown notification id returns %d (repo reports no not-found)", wUnknown.Code)
}

// A different user in the SAME tenant must be refused. Before the fix MarkRead
// filtered on id AND tenant_id only, so this user could mark the owner's row
// read and still received 200.
func TestNotificationMarkReadRefusesOtherUsers(t *testing.T) {
	f := newHandlerFixture(t)
	owner := f.AdminUserID
	id := testutil.CreateNotification(t, f.DB, f.TenantID, &owner, "system_alert", "Owner's", false)
	other := testutil.CreateUser(t, f.DB, f.TenantID, testutil.UniqueEmail(t, "other"), "admin")

	_, code := notifMarkRead(t, f, other, id)
	if code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 when another user targets the notification", code)
	}

	var readAt sql.NullString
	if err := f.DB.QueryRow(`SELECT read_at::text FROM notifications WHERE id = $1`, id).Scan(&readAt); err != nil {
		t.Fatalf("read read_at: %v", err)
	}
	if readAt.Valid {
		t.Fatal("SECURITY: another user's notification was marked read")
	}
}

// An unknown id is a 404, not a silent 200: the repo now reports whether a row
// was actually touched.
func TestNotificationMarkReadUnknownIsNotFound(t *testing.T) {
	f := newHandlerFixture(t)

	_, code := notifMarkRead(t, f, f.AdminUserID, 99999999)
	if code != http.StatusNotFound {
		t.Fatalf("unknown id code=%d, want 404", code)
	}
}

// A cross-tenant id must not be touchable at all.
func TestNotificationMarkReadCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	otherUser := f.OtherCoachUserID
	id := testutil.CreateNotification(t, f.DB, f.OtherTenantID, &otherUser, "system_alert", "Foreign", false)

	_, code := notifMarkRead(t, f, f.AdminUserID, id)
	t.Logf("cross-tenant MarkRead returns %d", code)

	var readAt sql.NullString
	if err := f.DB.QueryRow(`SELECT read_at::text FROM notifications WHERE id = $1`, id).Scan(&readAt); err != nil {
		t.Fatalf("read read_at: %v", err)
	}
	if readAt.Valid {
		t.Fatal("SECURITY: a notification in another tenant was marked read")
	}
}

// MarkSeen clears the badge without touching read_at, so a row can leave the
// bell badge while staying in the Unread tab.
func TestNotificationMarkSeen(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	id := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "exam_submitted", "Seen Only", false)

	_, code := notifMarkSeen(t, f, uid, id)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	var readAt, seenAt sql.NullString
	if err := f.DB.QueryRow(`SELECT read_at::text, seen_at::text FROM notifications WHERE id = $1`, id).Scan(&readAt, &seenAt); err != nil {
		t.Fatalf("read timestamps: %v", err)
	}
	if !seenAt.Valid {
		t.Fatal("seen_at still NULL after MarkSeen")
	}
	if readAt.Valid {
		t.Fatal("MarkSeen must not set read_at")
	}
	unread, unseen := notifUnreadCounts(t, f, "admin", uid)
	if unread != 1 || unseen != 0 {
		t.Fatalf("unread=%d unseen=%d, want 1 and 0", unread, unseen)
	}
}

// MarkSeen must be owner-scoped exactly like MarkRead: another user's row is a
// 404 and stays unseen.
func TestNotificationMarkSeenRefusesOtherUsers(t *testing.T) {
	f := newHandlerFixture(t)
	owner := f.AdminUserID
	id := testutil.CreateNotification(t, f.DB, f.TenantID, &owner, "exam_submitted", "Owner's", false)
	other := testutil.CreateUser(t, f.DB, f.TenantID, testutil.UniqueEmail(t, "other"), "admin")

	_, code := notifMarkSeen(t, f, other, id)
	if code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 when another user targets the notification", code)
	}

	var seenAt sql.NullString
	if err := f.DB.QueryRow(`SELECT seen_at::text FROM notifications WHERE id = $1`, id).Scan(&seenAt); err != nil {
		t.Fatalf("read seen_at: %v", err)
	}
	if seenAt.Valid {
		t.Fatal("SECURITY: another user's notification was marked seen")
	}
}

// Mark-all-seen clears the caller's badge only; another user's unseen row
// survives and their badge is unaffected.
func TestNotificationMarkAllSeen(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	a := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "exam_submitted", "A", false)
	b := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "sqi_complete", "B", false)
	other := testutil.CreateUser(t, f.DB, f.TenantID, testutil.UniqueEmail(t, "other"), "coach")
	otherRow := testutil.CreateNotification(t, f.DB, f.TenantID, &other, "exam_submitted", "Other's", false)

	if _, code := notifMarkAllSeen(t, f); code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}

	for _, id := range []int{a, b} {
		var seenAt sql.NullString
		if err := f.DB.QueryRow(`SELECT seen_at::text FROM notifications WHERE id = $1`, id).Scan(&seenAt); err != nil {
			t.Fatalf("read %d: %v", id, err)
		}
		if !seenAt.Valid {
			t.Fatalf("notification %d still unseen after mark-all-seen", id)
		}
	}
	var otherSeenAt sql.NullString
	if err := f.DB.QueryRow(`SELECT seen_at::text FROM notifications WHERE id = $1`, otherRow).Scan(&otherSeenAt); err != nil {
		t.Fatalf("read other row: %v", err)
	}
	if otherSeenAt.Valid {
		t.Fatalf("SECURITY: mark-all-seen cleared user %d's notification %d", other, otherRow)
	}
}

// Mark-all-read is owner-scoped. A wider predicate would stamp read_at on every
// colleague's rows and silently empty their Unread tab.
func TestNotificationMarkAllRead(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	a := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "system_alert", "A", false)
	b := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "sqi_complete", "B", false)

	// Another user's row in the same tenant must be left alone.
	other := testutil.CreateUser(t, f.DB, f.TenantID, testutil.UniqueEmail(t, "other"), "coach")
	otherRow := testutil.CreateNotification(t, f.DB, f.TenantID, &other, "exam_submitted", "Other's", false)

	_, code := notifMarkAllRead(t, f)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}

	// The admin's own rows are cleared.
	for _, id := range []int{a, b} {
		var readAt sql.NullString
		if err := f.DB.QueryRow(`SELECT read_at::text FROM notifications WHERE id = $1`, id).Scan(&readAt); err != nil {
			t.Fatalf("read %d: %v", id, err)
		}
		if !readAt.Valid {
			t.Fatalf("notification %d still unread after mark-all-read", id)
		}
	}

	// The coach's row must survive — the isolation guarantee.
	var otherReadAt sql.NullString
	if err := f.DB.QueryRow(`SELECT read_at::text FROM notifications WHERE id = $1`, otherRow).Scan(&otherReadAt); err != nil {
		t.Fatalf("read other row: %v", err)
	}
	if otherReadAt.Valid {
		t.Fatalf("SECURITY: mark-all-read cleared user %d's notification %d", other, otherRow)
	}
}

// The admin sees every row in the tenant but must not be able to act on a
// coach's: MarkRead and Delete are owner-scoped, so both 404.
func TestNotificationAdminActsOnlyOnOwnRows(t *testing.T) {
	f := newHandlerFixture(t)
	coachRow := testutil.CreateNotification(t, f.DB, f.TenantID, &f.CoachUserID, "exam_submitted", "Coach's", false)

	if _, code := notifMarkRead(t, f, f.AdminUserID, coachRow); code != http.StatusNotFound {
		t.Fatalf("admin MarkRead on a coach's row = %d, want 404", code)
	}
	if _, code := notifDelete(t, f, coachRow); code != http.StatusNotFound {
		t.Fatalf("admin Delete on a coach's row = %d, want 404", code)
	}

	var readAt sql.NullString
	if err := f.DB.QueryRow(`SELECT read_at::text FROM notifications WHERE id = $1`, coachRow).Scan(&readAt); err != nil {
		t.Fatalf("read coach row: %v", err)
	}
	if readAt.Valid {
		t.Fatal("SECURITY: the admin marked a coach's notification read")
	}
	var left int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE id = $1`, coachRow).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left == 0 {
		t.Fatal("SECURITY: the admin deleted a coach's notification")
	}
}

// A coach's list is scoped to their own rows: the admin's notifications and
// another coach's must both be absent.
func TestNotificationCoachSeesOnlyOwnRows(t *testing.T) {
	f := newHandlerFixture(t)
	adminRow := testutil.CreateNotification(t, f.DB, f.TenantID, &f.AdminUserID, "system_alert", "Admin's row", false)
	otherCoach := testutil.CreateUser(t, f.DB, f.TenantID, testutil.UniqueEmail(t, "peer"), "coach")
	peerRow := testutil.CreateNotification(t, f.DB, f.TenantID, &otherCoach, "exam_submitted", "Peer's row", false)
	ownRow := testutil.CreateNotification(t, f.DB, f.TenantID, &f.CoachUserID, "exam_submitted", "Own row", false)

	body, code := notifGetAsCoach(t, f, "")
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if !strings.Contains(body, "Own row") {
		t.Fatalf("coach's own notification missing: %s", body)
	}
	if strings.Contains(body, "Admin's row") {
		t.Fatalf("coach list leaked the admin's notification: %s", body)
	}
	if strings.Contains(body, "Peer's row") {
		t.Fatalf("coach list leaked another coach's notification: %s", body)
	}
	// Unread count must be scoped the same way.
	if got := notifUnreadCountAsCoach(t, f); got != 1 {
		t.Fatalf("coach unread_count=%d, want 1 (own unread row only)", got)
	}
	_ = adminRow
	_ = peerRow
	_ = ownRow
}

// A super admin reads platform rows only. The same row must be invisible and
// untouchable from a tenant context, and tenant rows must not leak upward.
func TestNotificationSuperAdminPlatformScope(t *testing.T) {
	f := newHandlerFixture(t)
	superID := testutil.CreateSuperAdmin(t, f.DB)
	defer f.DB.Exec(`DELETE FROM users WHERE id = $1`, superID)

	platformRow := testutil.CreatePlatformNotification(t, f.DB, superID, "tenant_created", "Platform Only", false)
	testutil.CreateNotification(t, f.DB, f.TenantID, &f.AdminUserID, "exam_submitted", "Tenant Row", false)

	body, code := notifGetAsSuperAdmin(t, f, superID, "")
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if !strings.Contains(body, "Platform Only") {
		t.Fatalf("platform list missed the platform row: %s", body)
	}
	if strings.Contains(body, "Tenant Row") {
		t.Fatalf("platform list leaked a tenant row: %s", body)
	}
	unread, unseen := notifUnreadCounts(t, f, "super_admin", superID)
	if unread != 1 || unseen != 1 {
		t.Fatalf("platform counts unread=%d unseen=%d, want 1 and 1", unread, unseen)
	}

	// A tenant admin never sees the platform row and cannot act on it.
	tenantBody, _ := notifGet(t, f, "")
	if strings.Contains(tenantBody, "Platform Only") {
		t.Fatalf("tenant list leaked a platform row: %s", tenantBody)
	}
	if _, code := notifMarkRead(t, f, f.AdminUserID, platformRow); code != http.StatusNotFound {
		t.Fatalf("tenant MarkRead on a platform row = %d, want 404", code)
	}

	// Its owner can, under platform scope.
	if _, code := notifMarkReadAsSuperAdmin(t, f, superID, platformRow); code != http.StatusOK {
		t.Fatalf("platform MarkRead = %d, want 200", code)
	}
	unread, unseen = notifUnreadCounts(t, f, "super_admin", superID)
	if unread != 0 || unseen != 0 {
		t.Fatalf("platform counts after read unread=%d unseen=%d, want 0 and 0", unread, unseen)
	}
}

func TestNotificationDelete(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	id := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "system_alert", "To Delete", false)

	_, code := notifDelete(t, f, id)
	if code != http.StatusOK {
		t.Fatalf("code=%d", code)
	}
	var left int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE id = $1`, id).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left != 0 {
		t.Fatal("notification survived Delete")
	}

	bad, wBad := f.ctx(t, "admin")
	withParam(bad, "id", "abc")
	f.Notifications.DeleteNotification(bad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("bad id code=%d, want 400", wBad.Code)
	}
}

// Another user's notification must be refused rather than deleted.
func TestNotificationDeleteRefusesOtherUsers(t *testing.T) {
	f := newHandlerFixture(t)
	owner := f.AdminUserID
	id := testutil.CreateNotification(t, f.DB, f.TenantID, &owner, "system_alert", "Owner's", false)
	other := testutil.CreateUser(t, f.DB, f.TenantID, testutil.UniqueEmail(t, "other"), "admin")

	_, code := notifDeleteAs(t, f, other, id)
	if code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 when another user targets the notification", code)
	}

	var left int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE id = $1`, id).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left == 0 {
		t.Fatal("SECURITY: another user's notification was deleted")
	}
}

func TestNotificationDeleteUnknownIsNotFound(t *testing.T) {
	f := newHandlerFixture(t)

	_, code := notifDeleteAs(t, f, f.AdminUserID, 99999999)
	if code != http.StatusNotFound {
		t.Fatalf("unknown id code=%d, want 404", code)
	}
}

// Cross-tenant delete must be refused even though the handler cannot tell.
func TestNotificationDeleteCrossTenant(t *testing.T) {
	f := newHandlerFixture(t)
	otherUser := f.OtherCoachUserID
	id := testutil.CreateNotification(t, f.DB, f.OtherTenantID, &otherUser, "system_alert", "Foreign", false)

	_, code := notifDelete(t, f, id)
	t.Logf("cross-tenant Delete returns %d", code)

	var left int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE id = $1`, id).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left == 0 {
		t.Fatal("SECURITY: a notification in another tenant was deleted")
	}
}

func TestNotificationPreferences(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID

	// Migration 000016 seeds preference rows only for users that existed at
	// migration time, so a fixture user starts with none.
	body, code := notifGetPrefs(t, f)
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if !strings.Contains(body, `"preferences":[]`) {
		t.Fatalf("expected an empty array for a fresh user, got %s", body)
	}

	_, code = notifUpdatePrefs(t, f, f.AdminUserID, notifPrefsBody)
	if code != http.StatusOK {
		t.Fatalf("update code=%d", code)
	}

	body, code = notifGetPrefs(t, f)
	if code != http.StatusOK {
		t.Fatalf("code=%d body=%s", code, body)
	}
	if !strings.Contains(body, "exam_submitted") || !strings.Contains(body, "coach_activity") {
		t.Fatalf("preferences not persisted: %s", body)
	}

	// Upsert: the same event type must update, not duplicate.
	_, code = notifUpdatePrefs(t, f, f.AdminUserID, notifPrefsBody)
	if code != http.StatusOK {
		t.Fatalf("second update code=%d", code)
	}
	var rows int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM notification_preferences WHERE user_id = $1 AND event_type = 'exam_submitted'`, uid).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Fatalf("exam_submitted has %d rows, want 1 (the ON CONFLICT upsert must not duplicate)", rows)
	}
}

func TestNotificationUpdatePreferencesBadPayload(t *testing.T) {
	f := newHandlerFixture(t)

	bad, wBad := f.ctx(t, "admin")
	withJSONBody(bad, http.MethodPut, `{}`)
	f.Notifications.UpdatePreferences(bad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("missing preferences code=%d, want 400", wBad.Code)
	}

	malformed, wMalformed := f.ctx(t, "admin")
	withJSONBody(malformed, http.MethodPut, `{"preferences":`)
	f.Notifications.UpdatePreferences(malformed)
	if wMalformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed code=%d, want 400", wMalformed.Code)
	}
}

// Retired and unknown event types are rejected rather than stored as dead
// preferences that can never fire.
func TestNotificationUpdatePreferencesRejectsUnknownEventType(t *testing.T) {
	f := newHandlerFixture(t)

	if _, code := notifUpdatePrefs(t, f, f.AdminUserID, `{"preferences":{"system_alert":true}}`); code != http.StatusBadRequest {
		t.Fatalf("retired event type code=%d, want 400", code)
	}
	if _, code := notifUpdatePrefs(t, f, f.AdminUserID, `{"preferences":{"not_a_real_event":false}}`); code != http.StatusBadRequest {
		t.Fatalf("unknown event type code=%d, want 400", code)
	}

	var rows int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM notification_preferences WHERE user_id = $1 AND event_type IN ('system_alert', 'not_a_real_event')`, f.AdminUserID).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 0 {
		t.Fatalf("rejected event types were persisted: %d rows", rows)
	}
}

// notifGetPrefs reads the preferences endpoint, which is a different handler
// method from ListNotifications.
func notifGetPrefs(t *testing.T, f *handlerFixture) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notifications/preferences", nil)
	f.Notifications.GetPreferences(c)
	return w.Body.String(), w.Code
}

// notifGet takes a query string, not a path. "" means no filter.
func notifGet(t *testing.T, f *handlerFixture, query string) (string, int) {
	return notifGetAs(t, f, f.AdminUserID, query)
}

func notifGetAs(t *testing.T, f *handlerFixture, userID int, query string) (string, int) {
	t.Helper()
	target := "/admin/notifications"
	if query != "" {
		target += "?" + query
	}
	c, w := f.ctx(t, "admin")
	c.Set("user_id", userID)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	f.Notifications.ListNotifications(c)
	return w.Body.String(), w.Code
}

func notifGetAsCoach(t *testing.T, f *handlerFixture, query string) (string, int) {
	t.Helper()
	target := "/coach/notifications"
	if query != "" {
		target += "?" + query
	}
	c, w := f.ctxAsCoach(t)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	f.Notifications.ListNotifications(c)
	return w.Body.String(), w.Code
}

func notifUnreadCountAsCoach(t *testing.T, f *handlerFixture) int {
	t.Helper()
	c, w := f.ctxAsCoach(t)
	c.Request = httptest.NewRequest(http.MethodGet, "/coach/notifications/unread-count", nil)
	f.Notifications.UnreadCount(c)
	if w.Code != http.StatusOK {
		t.Fatalf("coach unread count code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Count int `json:"unread_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp.Count
}

func notifUnreadCount(t *testing.T, f *handlerFixture) int {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/notifications/unread-count", nil)
	f.Notifications.UnreadCount(c)
	if w.Code != http.StatusOK {
		t.Fatalf("unread count code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Count int `json:"unread_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp.Count
}

// notifPrefsBody sets two event types, one enabled and one disabled, so the
// round-trip proves the value is stored rather than defaulted to true. Both are
// live event types: UpdatePreferences rejects retired or unknown keys.
const notifPrefsBody = `{"preferences":{"exam_submitted":true,"coach_activity":false}}`

// notifMarkRead calls MarkRead with the id path param set, the way the router
// does. The param has to be seeded explicitly: the handler reads c.Param("id"),
// not the URL, so a request built with a path alone yields an empty param.
func notifMarkRead(t *testing.T, f *handlerFixture, userID, id int) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Set("user_id", userID)
	c.Request = httptest.NewRequest(http.MethodPut, notifReadPath(id), nil)
	withParam(c, "id", strconv.Itoa(id))
	f.Notifications.MarkRead(c)
	return w.Body.String(), w.Code
}

func notifMarkAllRead(t *testing.T, f *handlerFixture) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/notifications/read-all", nil)
	f.Notifications.MarkAllRead(c)
	return w.Body.String(), w.Code
}

func notifUpdatePrefs(t *testing.T, f *handlerFixture, userID int, body string) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Set("user_id", userID)
	withJSONBody(c, http.MethodPut, body)
	f.Notifications.UpdatePreferences(c)
	return w.Body.String(), w.Code
}

func notifDelete(t *testing.T, f *handlerFixture, id int) (string, int) {
	return notifDeleteAs(t, f, f.AdminUserID, id)
}

func notifDeleteAs(t *testing.T, f *handlerFixture, userID, id int) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Set("user_id", userID)
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin/notifications/"+strconv.Itoa(id), nil)
	withParam(c, "id", strconv.Itoa(id))
	f.Notifications.DeleteNotification(c)
	return w.Body.String(), w.Code
}

func notifReadPath(id int) string {
	return "/admin/notifications/" + strconv.Itoa(id) + "/read"
}

// notifGetAsSuperAdmin lists under /super-admin, where scopeFor maps the role to
// ScopePlatform. userID must be a real super admin: the platform predicate
// filters on the recipient's user_id.
func notifGetAsSuperAdmin(t *testing.T, f *handlerFixture, superID int, query string) (string, int) {
	t.Helper()
	target := "/super-admin/notifications"
	if query != "" {
		target += "?" + query
	}
	c, w := f.ctx(t, "super_admin")
	c.Set("user_id", superID)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	f.Notifications.ListNotifications(c)
	return w.Body.String(), w.Code
}

// notifUnreadCounts returns both counts the bell needs, so tests can prove the
// badge (unseen) moves independently of the Unread tab (unread).
func notifUnreadCounts(t *testing.T, f *handlerFixture, role string, userID int) (unread, unseen int) {
	t.Helper()
	c, w := f.ctx(t, role)
	c.Set("user_id", userID)
	c.Request = httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	f.Notifications.UnreadCount(c)
	if w.Code != http.StatusOK {
		t.Fatalf("unread count code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Unread int `json:"unread_count"`
		Unseen int `json:"unseen_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp.Unread, resp.Unseen
}

func notifMarkSeen(t *testing.T, f *handlerFixture, userID, id int) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Set("user_id", userID)
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/notifications/"+strconv.Itoa(id)+"/seen", nil)
	withParam(c, "id", strconv.Itoa(id))
	f.Notifications.MarkSeen(c)
	return w.Body.String(), w.Code
}

func notifMarkAllSeen(t *testing.T, f *handlerFixture) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "admin")
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/notifications/seen-all", nil)
	f.Notifications.MarkAllSeen(c)
	return w.Body.String(), w.Code
}

func notifMarkReadAsSuperAdmin(t *testing.T, f *handlerFixture, superID, id int) (string, int) {
	t.Helper()
	c, w := f.ctx(t, "super_admin")
	c.Set("user_id", superID)
	c.Request = httptest.NewRequest(http.MethodPut, "/super-admin/notifications/"+strconv.Itoa(id)+"/read", nil)
	withParam(c, "id", strconv.Itoa(id))
	f.Notifications.MarkRead(c)
	return w.Body.String(), w.Code
}
