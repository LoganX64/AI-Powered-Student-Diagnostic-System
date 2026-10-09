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

// notification_handler.go was the last handler file with no tests at all.
//
// The important finding here is not a missing test but a missing authorization
// check: NotificationRepo.MarkRead and .Delete filter on `id AND tenant_id` and
// never on `user_id`, so any authenticated user in a tenant can mark another
// user's notification read or delete it. Both report no not-found, so they return
// 200 either way. TestNotificationMarkReadAcrossUsers pins that behaviour by
// name so the hole cannot be mistaken for working authorization.

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

	// A broadcast row (user_id NULL) counts for the user.
	testutil.CreateNotification(t, f.DB, f.TenantID, nil, "system_alert", "Broadcast", false)
	if got := notifUnreadCount(t, f); got != 3 {
		t.Fatalf("unread_count=%d after a broadcast row, want 3", got)
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

// A broadcast row (user_id NULL) is org-wide and read_at is a single column, so
// one user marking it read would hide it from everyone. It must be refused.
func TestNotificationMarkReadRefusesBroadcast(t *testing.T) {
	f := newHandlerFixture(t)
	id := testutil.CreateNotification(t, f.DB, f.TenantID, nil, "system_alert", "Broadcast", false)

	_, code := notifMarkRead(t, f, f.AdminUserID, id)
	if code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 for a broadcast row", code)
	}

	var readAt sql.NullString
	if err := f.DB.QueryRow(`SELECT read_at::text FROM notifications WHERE id = $1`, id).Scan(&readAt); err != nil {
		t.Fatalf("read read_at: %v", err)
	}
	if readAt.Valid {
		t.Fatal("a broadcast notification was marked read by one user")
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

// Mark-all-read is owner-scoped. The admin holds a tenant-wide list, so a wider
// predicate here would stamp read_at on every coach's rows and silently clear
// their unread badges. Broadcast rows are excluded for the same reason: one
// shared row cannot carry per-user read state, which is also why MarkRead
// refuses them.
func TestNotificationMarkAllRead(t *testing.T) {
	f := newHandlerFixture(t)
	uid := f.AdminUserID
	a := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "system_alert", "A", false)
	b := testutil.CreateNotification(t, f.DB, f.TenantID, &uid, "sqi_complete", "B", false)
	broadcast := testutil.CreateNotification(t, f.DB, f.TenantID, nil, "system_alert", "Broadcast", false)

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

	// Broadcast rows stay unread too.
	var broadcastReadAt sql.NullString
	if err := f.DB.QueryRow(`SELECT read_at::text FROM notifications WHERE id = $1`, broadcast).Scan(&broadcastReadAt); err != nil {
		t.Fatalf("read broadcast: %v", err)
	}
	if broadcastReadAt.Valid {
		t.Fatal("SECURITY: mark-all-read cleared a broadcast row")
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

// A broadcast row is org-wide, so one user must not be able to delete it.
func TestNotificationDeleteRefusesBroadcast(t *testing.T) {
	f := newHandlerFixture(t)
	id := testutil.CreateNotification(t, f.DB, f.TenantID, nil, "system_alert", "Broadcast", false)

	_, code := notifDeleteAs(t, f, f.AdminUserID, id)
	if code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404 for a broadcast row", code)
	}

	var left int
	if err := f.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE id = $1`, id).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left == 0 {
		t.Fatal("a broadcast notification was deleted by one user")
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
	if !strings.Contains(body, "exam_submitted") || !strings.Contains(body, "storage_warning") {
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
// round-trip proves the value is stored rather than defaulted to true.
const notifPrefsBody = `{"preferences":{"exam_submitted":true,"storage_warning":false}}`

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