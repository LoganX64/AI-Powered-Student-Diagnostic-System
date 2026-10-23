package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/testutil"
)

// notifTenant creates an isolated tenant mirroring the real topology: exactly one
// admin (A, who owns the org), plus coaches B (no students), C (no students) and
// D (owns the student). Returns the tenant id, the four user ids, and D's
// assignment.
//
// One admin matters for the assertions: sqi_complete resolves to every admin, so
// a second admin would make the expected recipient count wrong and hide a bug
// where the admin list is not actually read.
//
// The coaches table is what resolves targeting — notifications are addressed by
// users.id while assignments reference coaches.id — so a test that omits a coach
// row would silently exercise the admin-only fallback instead of the coach path.
func notifTenant(t *testing.T, db *sql.DB) (tid int, a, b, c, d int, assignmentID int) {
	t.Helper()
	if err := db.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, "svc-notif-"+t.Name()).Scan(&tid); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	// One admin and three coaches. Admin count matters: sqi_complete resolves to
	// every admin, so a second admin would change the expected recipient set.
	a = testutil.CreateAdmin(t, db, tid)
	// CreateCoach inserts the users row and the coaches row together, which is
	// what targeting resolves through.
	b, _ = testutil.CreateCoach(t, db, tid)
	c, _ = testutil.CreateCoach(t, db, tid)
	var coachCID int
	d, coachCID = testutil.CreateCoach(t, db, tid)
	subject := testutil.CreateSubject(t, db, tid, "Subject")
	student := testutil.CreateStudent(t, db, tid, coachCID, testutil.UniqueCode(t, "stu"), "Bob")
	testID := testutil.CreateTest(t, db, tid, subject, coachCID, 60)
	assignmentID = testutil.CreateAssignment(t, db, student, testID, coachCID)

	return tid, a, b, c, d, assignmentID
}

// disableEvent turns one user's preference for an event off, proving the
// per-recipient gate still applies now that fan-out is targeted rather than
// blanket.
func disableEvent(t *testing.T, db *sql.DB, userID int, eventType string) {
	t.Helper()
	nr := repository.NewNotificationRepo(db)
	if err := nr.UpdatePreferences(userID, map[string]bool{eventType: false}); err != nil {
		t.Fatalf("disable %s for %d: %v", eventType, userID, err)
	}
}

// notifyRecipients returns the set of user_ids that received an event type,
// tenant-wide, so a test asserts on delivery rather than on a specific list order.
func notifyRecipients(t *testing.T, db *sql.DB, tid int, eventType string) map[int]bool {
	t.Helper()
	rows, err := db.Query(`SELECT user_id FROM notifications WHERE tenant_id = $1 AND event_type = $2`, tid, eventType)
	if err != nil {
		t.Fatalf("recipients query: %v", err)
	}
	defer rows.Close()
	got := map[int]bool{}
	for rows.Next() {
		var uid sql.NullInt64
		if err := rows.Scan(&uid); err != nil {
			t.Fatalf("scan recipient: %v", err)
		}
		if uid.Valid {
			got[int(uid.Int64)] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate recipients: %v", err)
	}
	return got
}

// Exam submission reaches the admin and the coach who owns the student, and
// nobody else — coach C has no students and must not be told.
func platformRecipients(t *testing.T, db *sql.DB, eventType string) map[int]bool {
	t.Helper()
	rows, err := db.Query(`SELECT user_id FROM notifications WHERE scope = 'platform' AND event_type = $1`, eventType)
	if err != nil {
		t.Fatalf("platform recipients query: %v", err)
	}
	defer rows.Close()
	got := map[int]bool{}
	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err != nil {
			t.Fatalf("scan platform recipient: %v", err)
		}
		got[uid] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate platform recipients: %v", err)
	}
	return got
}

func TestNotifyExamSubmittedTargetsOwningCoach(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, a, b, c, d, assignmentID := notifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	if err := svc.NotifyExamSubmitted(tid, 42, assignmentID, 1001, "Bob", false); err != nil {
		t.Fatalf("NotifyExamSubmitted: %v", err)
	}

	got := notifyRecipients(t, db, tid, "exam_submitted")
	if !got[a] {
		t.Fatalf("admin %d must receive exam_submitted, got %v", a, got)
	}
	if !got[d] {
		t.Fatalf("owning coach %d must receive exam_submitted, got %v", d, got)
	}
	if got[c] {
		t.Fatalf("coach %d owns no students and must not be notified, got %v", c, got)
	}
	if got[b] {
		t.Fatalf("coach %d was never a recipient and must not be notified, got %v", b, got)
	}
	if len(got) != 2 {
		t.Fatalf("expected exactly 2 recipients, got %v", got)
	}

	// Metadata must round-trip so the UI can deep-link to the submission.
	rows, _, err := nr.List(tid, &a, repository.ScopeOwn, "exam_submitted", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("no rows for the admin")
	}
	// Metadata must round-trip so the UI can deep-link to the submission. It is
	// mixed-type now: numeric ids for routing, plus resolved display names.
	var meta map[string]interface{}
	if err := json.Unmarshal(rows[0].Metadata, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if int(meta["student_id"].(float64)) != 42 || int(meta["assignment_id"].(float64)) != assignmentID {
		t.Fatalf("metadata mismatch: %v", meta)
	}
	// The message must read as prose, not as a primary key: "Coach (ID: 3)" and
	// "assignment 7" tell an admin nothing they can act on.
	if !strings.Contains(rows[0].Message, "Bob") {
		t.Fatalf("message must name the student, got %q", rows[0].Message)
	}
	for _, banned := range []string{"(ID:", "assignment"} {
		if strings.Contains(rows[0].Message, banned) {
			t.Fatalf("message leaks raw identifiers (%q): %q", banned, rows[0].Message)
		}
	}
}

// sqi_complete is admin-only: the batch spans the organization, so no single
// coach owns it.
func TestNotifySQICompleteIsAdminOnly(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, a, _, c, d, _ := notifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	if err := svc.NotifySQIComplete(tid, 7, "completed", 120, 0); err != nil {
		t.Fatalf("NotifySQIComplete: %v", err)
	}

	got := notifyRecipients(t, db, tid, "sqi_complete")
	if !got[a] {
		t.Fatalf("admin must receive sqi_complete, got %v", got)
	}
	if got[c] || got[d] {
		t.Fatalf("no coach may receive sqi_complete, got %v", got)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 recipient, got %v", got)
	}
}

// coach_activity reaches only the coach who acted and the admin. This was the
// noisiest event: it used to tell every coach about every other coach's work.
func TestNotifyCoachActivityTargetsActingCoach(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, a, _, c, d, _ := notifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	actingCoachID, err := repository.NewCoachRepo(db).GetIDFromUser(d)
	if err != nil {
		t.Fatalf("coach id: %v", err)
	}
	if err := svc.NotifyCoachActivity(tid, actingCoachID, "created test", "Patent exam", "coach_activity:created_test:1"); err != nil {
		t.Fatalf("NotifyCoachActivity: %v", err)
	}

	got := notifyRecipients(t, db, tid, "coach_activity")
	if !got[a] {
		t.Fatalf("admin must receive coach_activity, got %v", got)
	}
	if !got[d] {
		t.Fatalf("acting coach %d must receive coach_activity, got %v", d, got)
	}
	if got[c] {
		t.Fatalf("coach %d did not act and must not be notified, got %v", c, got)
	}
	if len(got) != 2 {
		t.Fatalf("expected exactly 2 recipients, got %v", got)
	}
}

// student_exam_logout targets the owning coach, same as exam_submitted.
func TestNotifyStudentExamLogoutTargetsOwningCoach(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, a, _, c, d, assignmentID := notifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	if err := svc.NotifyStudentExamLogout(tid, 42, assignmentID, 1002, "Bob"); err != nil {
		t.Fatalf("NotifyStudentExamLogout: %v", err)
	}

	got := notifyRecipients(t, db, tid, "student_exam_logout")
	if !got[a] || !got[d] {
		t.Fatalf("expected admin %d and owning coach %d, got %v", a, d, got)
	}
	if got[c] {
		t.Fatalf("coach %d owns no students and must not be notified, got %v", c, got)
	}

	rows, _, err := nr.List(tid, &a, repository.ScopeOwn, "student_exam_logout", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 notification for the admin, got %d", len(rows))
	}
	if rows[0].Priority != "warning" {
		t.Fatalf("expected priority warning, got %q", rows[0].Priority)
	}
}

// A timer-expiry submission is labelled "Auto Submit" and states no reason. The
// server cannot distinguish a student who ran out of time from one still on the
// last question, so any cause in the wording would be a guess the coach could
// act on wrongly.
func TestNotifyExamSubmittedAutoSubmitWording(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, a, _, _, _, assignmentID := notifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	if err := svc.NotifyExamSubmitted(tid, 42, assignmentID, 2001, "Bob", false); err != nil {
		t.Fatalf("manual: %v", err)
	}
	if err := svc.NotifyExamSubmitted(tid, 42, assignmentID, 2002, "Bob", true); err != nil {
		t.Fatalf("auto: %v", err)
	}

	rows, _, err := nr.List(tid, &a, repository.ScopeOwn, "exam_submitted", "", false, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	byTitle := map[string]repository.NotificationRow{}
	for _, r := range rows {
		byTitle[r.Title] = r
	}
	manualRow, hasManual := byTitle["Exam Submitted"]
	autoRow, hasAuto := byTitle["Auto Submit"]
	if !hasManual || !hasAuto {
		t.Fatalf("titles = %v, want one 'Exam Submitted' and one 'Auto Submit'", byTitle)
	}

	if !strings.Contains(autoRow.Message, "auto-submitted") {
		t.Fatalf("auto message must say it was auto-submitted, got %q", autoRow.Message)
	}
	if !strings.Contains(autoRow.Message, "Bob") || !strings.Contains(autoRow.Message, "Test") {
		t.Fatalf("auto message must name the student and the paper, got %q", autoRow.Message)
	}
	// The reason is the whole point: none of these may appear.
	for _, banned := range []string{"ran out of time", "time expired", "logged out", "unanswered"} {
		if strings.Contains(strings.ToLower(autoRow.Message), banned) {
			t.Fatalf("auto message attributes a reason (%q): %q", banned, autoRow.Message)
		}
	}
	// Manual wording must be untouched by this feature.
	if !strings.Contains(manualRow.Message, "submitted") {
		t.Fatalf("manual message changed unexpectedly: %q", manualRow.Message)
	}

	// Both stay the same event type and severity, so the tabs and unread counts
	// behave identically for a manual and an auto submission.
	if manualRow.EventType != autoRow.EventType {
		t.Fatalf("event_type diverged: %q vs %q", manualRow.EventType, autoRow.EventType)
	}
	if autoRow.Priority != "info" {
		t.Fatalf("auto priority = %q, want info", autoRow.Priority)
	}

	// The flag is recorded so a consumer can tell the two apart later.
	var meta map[string]interface{}
	if err := json.Unmarshal(autoRow.Metadata, &meta); err != nil {
		t.Fatalf("unmarshal auto metadata: %v", err)
	}
	if meta["auto_submit"] != true {
		t.Fatalf("auto_submit missing from metadata: %v", meta)
	}
	var manualMeta map[string]interface{}
	if err := json.Unmarshal(manualRow.Metadata, &manualMeta); err != nil {
		t.Fatalf("unmarshal manual metadata: %v", err)
	}
	if _, present := manualMeta["auto_submit"]; present {
		t.Fatalf("manual submission must not carry auto_submit: %v", manualMeta)
	}
}

// A replayed submission must not produce a second row for either recipient.
func TestNotifyExamSubmittedIsIdempotent(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, a, _, _, d, assignmentID := notifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	for i := 0; i < 2; i++ {
		if err := svc.NotifyExamSubmitted(tid, 42, assignmentID, 4001, "Bob", false); err != nil {
			t.Fatalf("NotifyExamSubmitted attempt %d: %v", i+1, err)
		}
	}

	got := notifyRecipients(t, db, tid, "exam_submitted")
	if len(got) != 2 || !got[a] || !got[d] {
		t.Fatalf("expected exactly admin %d and coach %d, got %v", a, d, got)
	}

	var rows int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE tenant_id = $1 AND event_type = 'exam_submitted' AND event_key = $2
	`, tid, "exam_submitted:attempt:4001").Scan(&rows); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rows != 2 {
		t.Fatalf("expected one row per recipient after retry, got %d", rows)
	}
}

func TestIsValidEventType(t *testing.T) {
	valid := []string{
		EventExamSubmitted, EventCoachActivity, EventSQIComplete, EventStudentExamLogout,
		EventTenantCreated, EventTenantSuspended, EventTenantReactivated, EventPlanChanged,
	}
	for _, eventType := range valid {
		if !IsValidEventType(eventType) {
			t.Errorf("IsValidEventType(%q) = false, want true", eventType)
		}
	}
	for _, eventType := range []string{"", "system_alert", "storage_warning", "not_an_event"} {
		if IsValidEventType(eventType) {
			t.Errorf("IsValidEventType(%q) = true, want false", eventType)
		}
	}
}

// Platform events go to super admins only, with scope=platform and no tenant.
func TestNotifyPlatformEventsTargetSuperAdmins(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid := testutil.CreateTenant(t, db)
	admin := testutil.CreateAdmin(t, db, tid)
	superA := testutil.CreateSuperAdmin(t, db)
	superB := testutil.CreateSuperAdmin(t, db)
	defer func() {
		db.Exec(`DELETE FROM notifications WHERE scope = 'platform' AND metadata->>'tenant_id' = $1`, fmt.Sprintf("%d", tid))
		db.Exec(`DELETE FROM users WHERE id IN ($1, $2)`, superA, superB)
		db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)
	}()

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	if err := svc.NotifyTenantCreated(tid, "Acme"); err != nil {
		t.Fatalf("NotifyTenantCreated: %v", err)
	}
	// tenant_created has a fixed event key; a retry is suppressed per recipient.
	if err := svc.NotifyTenantCreated(tid, "Acme"); err != nil {
		t.Fatalf("NotifyTenantCreated retry: %v", err)
	}

	created := platformRecipients(t, db, EventTenantCreated)
	if !created[superA] || !created[superB] {
		t.Fatalf("both super admins must receive tenant_created, got %v", created)
	}
	if created[admin] {
		t.Fatalf("tenant admin %d must not receive a platform event, got %v", admin, created)
	}

	var createdRows int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE event_type = $1 AND event_key = $2 AND scope = 'platform' AND tenant_id IS NULL
		  AND user_id IN ($3, $4)
	`, EventTenantCreated, fmt.Sprintf("tenant_created:%d", tid), superA, superB).Scan(&createdRows); err != nil {
		t.Fatalf("count tenant_created: %v", err)
	}
	if createdRows != 2 {
		t.Fatalf("expected one tenant_created row per super admin, got %d", createdRows)
	}

	events := []struct {
		eventType string
		notify    func() error
	}{
		{EventTenantSuspended, func() error { return svc.NotifyTenantSuspended(tid, "Acme") }},
		{EventTenantReactivated, func() error { return svc.NotifyTenantReactivated(tid, "Acme") }},
		{EventPlanChanged, func() error { return svc.NotifyPlanChanged(tid, "Acme", 2, "Starter") }},
	}
	for _, e := range events {
		if err := e.notify(); err != nil {
			t.Fatalf("%s: %v", e.eventType, err)
		}
		got := platformRecipients(t, db, e.eventType)
		if !got[superA] || !got[superB] {
			t.Fatalf("%s must reach both super admins, got %v", e.eventType, got)
		}
		if got[admin] {
			t.Fatalf("%s must not reach tenant admin %d, got %v", e.eventType, admin, got)
		}
	}
}

// The per-recipient preference gate still applies after targeting: an admin who
// disabled the event is skipped while the owning coach still receives it.
func TestNotifyTargetedRespectsPreferences(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, a, _, c, d, assignmentID := notifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	disableEvent(t, db, a, "exam_submitted")

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))
	if err := svc.NotifyExamSubmitted(tid, 42, assignmentID, 1001, "Bob", false); err != nil {
		t.Fatalf("NotifyExamSubmitted: %v", err)
	}

	got := notifyRecipients(t, db, tid, "exam_submitted")
	if got[a] {
		t.Fatalf("admin %d disabled exam_submitted and must be skipped, got %v", a, got)
	}
	if !got[d] {
		t.Fatalf("owning coach %d must still receive it, got %v", d, got)
	}
	if got[c] {
		t.Fatalf("coach %d is not a recipient, got %v", c, got)
	}
}

// A coach lookup that fails must not lose the event: the admin still receives it.
// Assignment 9999999 does not exist, so the coach cannot be resolved.
func TestNotifyExamSubmittedFallsBackToAdminWhenCoachUnknown(t *testing.T) {
	db := testutil.OpenTestDB(t)
	tid, a, _, _, _, _ := notifTenant(t, db)
	defer db.Exec(`DELETE FROM tenants WHERE id = $1`, tid)

	nr := repository.NewNotificationRepo(db)
	svc := NewNotificationService(nr, repository.NewUserRepo(db))

	if err := svc.NotifyExamSubmitted(tid, 42, 9999999, 3001, "Bob", false); err != nil {
		t.Fatalf("NotifyExamSubmitted must not fail on an unresolvable coach: %v", err)
	}

	got := notifyRecipients(t, db, tid, "exam_submitted")
	if !got[a] {
		t.Fatalf("admin must still be notified when the coach lookup fails, got %v", got)
	}
}
