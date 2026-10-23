package services

import (
	"ai-student-diagnostic/backend/internal/repository"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type NotificationService struct {
	NotificationRepo *repository.NotificationRepo
	UserRepo         *repository.UserRepo
}

func NewNotificationService(notifRepo *repository.NotificationRepo, userRepo *repository.UserRepo) *NotificationService {
	return &NotificationService{NotificationRepo: notifRepo, UserRepo: userRepo}
}

// The event types this system actually emits.
//
// system_alert and storage_warning were dropped: neither had a producer, so
// nothing could create them, and the preference rows seeded for them by
// migration 000016 could never be exercised. The SettingsPage no longer
// exposes toggles for them.
const (
	EventExamSubmitted     = "exam_submitted"
	EventCoachActivity     = "coach_activity"
	EventSQIComplete       = "sqi_complete"
	EventStudentExamLogout = "student_exam_logout"

	EventTenantCreated     = "tenant_created"
	EventTenantSuspended   = "tenant_suspended"
	EventTenantReactivated = "tenant_reactivated"
	EventPlanChanged       = "plan_changed"
)

const (
	notificationScopeTenant   = "tenant"
	notificationScopePlatform = "platform"
)

// IsValidEventType reports whether eventType is one this system emits. Handlers
// use it to reject unknown preference keys instead of silently storing them.
func IsValidEventType(eventType string) bool {
	switch eventType {
	case EventExamSubmitted,
		EventCoachActivity,
		EventSQIComplete,
		EventStudentExamLogout,
		EventTenantCreated,
		EventTenantSuspended,
		EventTenantReactivated,
		EventPlanChanged:
		return true
	default:
		return false
	}
}

// resolveRecipients picks the users who should receive an event.
//
// It replaces a blanket fan-out to every admin and coach in the tenant, which
// told coaches about colleagues' work they had no part in and gave the admin one
// copy per recipient of the same event. Recipients are now derived from who the
// event concerns:
//
//   - the admin always receives it: they own the organization and hold the
//     tenant-wide view that depends on these rows.
//   - a coach receives it only when the event is about their own students or
//     their own actions.
//
// An empty recipient set is a bug, not a reason to skip: log it and still notify
// the admin rather than dropping the event on the floor.
func (s *NotificationService) resolveRecipients(tenantID int, coachUserID *int) ([]int, error) {
	admins, err := s.UserRepo.ListIDsByTenantRoles(tenantID, []string{"admin"})
	if err != nil {
		return nil, err
	}
	if coachUserID == nil {
		return admins, nil
	}
	for _, id := range admins {
		if id == *coachUserID {
			// Already covered by the admin list; one row per person, never two.
			return admins, nil
		}
	}
	return append(admins, *coachUserID), nil
}

// notifyUsers writes one row per recipient, skipping any who disabled the event.
// Errors are logged, not fatal: a notification failure must never break the
// triggering action.
//
// scope is "tenant" (with a non-nil tenantID) or "platform" (tenantID nil).
// eventKey makes retries idempotent: CreateIdempotent treats a duplicate
// (user_id, event_key) pair as a no-op, so a replayed job or request cannot
// create a second row for the same recipient.
func (s *NotificationService) notifyUsers(
	scope string,
	tenantID *int,
	userIDs []int,
	eventType string,
	eventKey string,
	title string,
	message string,
	priority string,
	metadata map[string]interface{},
) error {
	if len(userIDs) == 0 {
		if tenantID != nil {
			log.Printf("[NOTIFICATION] no recipients resolved for event %s in tenant %d", eventType, *tenantID)
		} else {
			log.Printf("[NOTIFICATION] no platform recipients resolved for event %s", eventType)
		}
	}

	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		// Non-fatal: store an empty object rather than dropping the notification.
		log.Printf("[NOTIFICATION] failed to marshal metadata for event %s: %v", eventType, err)
		metaBytes = []byte("{}")
	}

	var keyPtr *string
	if eventKey != "" {
		keyPtr = &eventKey
	}

	for _, uid := range userIDs {
		enabled, perr := s.NotificationRepo.IsEventEnabled(uid, eventType)
		if perr != nil {
			// Fail open: if preference lookup errors, still deliver.
			log.Printf("[NOTIFICATION] preference check failed for user %d event %s: %v", uid, eventType, perr)
		}
		if !enabled {
			continue
		}
		if _, _, cerr := s.NotificationRepo.CreateIdempotent(repository.NotificationRow{
			TenantID:  tenantID,
			UserID:    &uid,
			EventType: eventType,
			Title:     title,
			Message:   message,
			Priority:  priority,
			Scope:     scope,
			EventKey:  keyPtr,
			Metadata:  metaBytes,
		}); cerr != nil {
			log.Printf("[NOTIFICATION] failed to create notification for user %d event %s: %v", uid, eventType, cerr)
		}
	}
	return nil
}

// assignmentContext resolves the readable facts about an assignment: which coach
// owns it, what the test was called, and who the student is. A failed lookup
// still returns usable text so a notification is never lost over formatting.
func (s *NotificationService) assignmentContext(tenantID, assignmentID int) repository.AssignmentNotifyContext {
	ctx, err := s.UserRepo.AssignmentContextForNotification(tenantID, assignmentID)
	if err != nil {
		log.Printf("[NOTIFICATION] assignment context lookup failed for %d in tenant %d: %v", assignmentID, tenantID, err)
		return repository.AssignmentNotifyContext{CoachUserID: -1}
	}
	return ctx
}

// describeStudent renders a student the way staff refer to them: name, with the
// login code alongside since that is what an admin looks up in the portal.
func describeStudent(name, code string) string {
	switch {
	case name != "" && code != "":
		return fmt.Sprintf("%s (%s)", name, code)
	case name != "":
		return name
	case code != "":
		return code
	default:
		return "A student"
	}
}

// NotifyExamSubmitted reports a completed exam.
//
// autoSubmit distinguishes the client's timer-expiry submission from a manual
// one, and changes only the wording. Neither variant states a reason: the server
// cannot tell a student who ran out of time from one who was still on the last
// question when the clock expired, so the coach reads the paper to find out.
func (s *NotificationService) NotifyExamSubmitted(tenantID, studentID, assignmentID, attemptID int, studentName string, autoSubmit bool) error {
	ctx := s.assignmentContext(tenantID, assignmentID)
	if ctx.StudentName != "" {
		studentName = ctx.StudentName
	}

	var coachUserID *int
	if ctx.CoachUserID >= 0 {
		coachUserID = &ctx.CoachUserID
	}
	recipients, err := s.resolveRecipients(tenantID, coachUserID)
	if err != nil {
		return err
	}

	student := describeStudent(studentName, ctx.StudentCode)

	title := "Exam Submitted"
	msg := fmt.Sprintf("%s submitted an exam.", student)
	if ctx.TestTitle != "" {
		msg = fmt.Sprintf("%s submitted %q.", student, ctx.TestTitle)
	}
	if autoSubmit {
		// Neutral by design: "auto-submitted" is a fact the server can state.
		// "ran out of time" is not, and would read as unanswered questions when
		// the student may simply have been on the last question.
		title = "Auto Submit"
		msg = fmt.Sprintf("%s's test paper was auto-submitted.", student)
		if ctx.TestTitle != "" {
			msg = fmt.Sprintf("%s's test paper %q was auto-submitted.", student, ctx.TestTitle)
		}
	}

	meta := map[string]interface{}{
		"student_id": studentID, "assignment_id": assignmentID, "attempt_id": attemptID,
		"test_title": ctx.TestTitle, "student_name": student,
	}
	if autoSubmit {
		meta["auto_submit"] = true
	}

	return s.notifyUsers(
		notificationScopeTenant, &tenantID, recipients,
		EventExamSubmitted,
		fmt.Sprintf("exam_submitted:attempt:%d", attemptID),
		title, msg, "info", meta,
	)
}

// NotifyCoachActivity reports a coach's own action. Only that coach and the
// admin are notified; peers no longer receive each other's activity.
//
// coachID is a coaches.id (what the handlers hold), so it is resolved to a
// users.id before targeting, and to a display name so the message names a person
// rather than a primary key. A failed lookup still notifies the admin.
func (s *NotificationService) NotifyCoachActivity(tenantID, coachID int, action, detail, eventKey string) error {
	var coachUserID *int
	coachName := "A coach"
	if uid, name, err := s.UserRepo.CoachIdentity(tenantID, coachID); err != nil {
		log.Printf("[NOTIFICATION] coach lookup failed for coach %d in tenant %d: %v", coachID, tenantID, err)
	} else {
		coachUserID = &uid
		if name != "" {
			coachName = name
		}
	}
	recipients, err := s.resolveRecipients(tenantID, coachUserID)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("%s %s", coachName, action)
	if detail != "" {
		msg = fmt.Sprintf("%s %s: %s", coachName, action, detail)
	}
	return s.notifyUsers(
		notificationScopeTenant, &tenantID, recipients,
		EventCoachActivity, eventKey,
		"Coach Activity", msg, "info",
		map[string]interface{}{"coach_id": coachID, "coach_name": coachName, "action": action},
	)
}

// NotifySQIComplete is admin-only. An SQI batch runs across the whole
// organization, so there is no single owning coach to target; coaches have no
// per-student stake in a job that spans every student.
//
// The message reports outcomes, not the job's primary key: "batch job #245" is
// meaningless to an admin, whereas how many attempts were processed — and how
// many failed — is the thing worth interrupting them for.
func (s *NotificationService) NotifySQIComplete(tenantID, jobID int, status string, done, failed int) error {
	recipients, err := s.resolveRecipients(tenantID, nil)
	if err != nil {
		return err
	}

	var msg string
	switch {
	case status == "failed" && done == 0 && failed == 0:
		msg = "SQI recomputation failed before any attempts were processed."
	case done == 0 && failed == 0:
		msg = "SQI recomputation finished, but there were no attempts to process."
	case failed == 0:
		msg = fmt.Sprintf("SQI recomputation finished. %s processed, none failed.", pluralize(done, "attempt"))
	case done == 0:
		msg = fmt.Sprintf("SQI recomputation finished, but all %s failed.", pluralize(failed, "attempt"))
	default:
		msg = fmt.Sprintf("SQI recomputation finished. %s processed, %s failed.",
			pluralize(done, "attempt"), pluralize(failed, "attempt"))
	}

	priority := "info"
	title := "SQI Computation Complete"
	switch status {
	case "failed":
		title = "SQI Computation Failed"
		priority = "warning"
	case "partial":
		title = "SQI Computation Partial"
		// A run where a meaningful share failed is what the admin needs to see;
		// the Scores tab will still be showing the previous numbers.
		if failed*4 >= done {
			priority = "warning"
		}
	}

	return s.notifyUsers(
		notificationScopeTenant, &tenantID, recipients,
		EventSQIComplete,
		fmt.Sprintf("sqi_complete:job:%d", jobID),
		title, msg, priority,
		map[string]interface{}{"job_id": jobID, "processed": done, "failed": failed},
	)
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// NotifyStudentExamLogout targets the coach who owns the assignment, same as
// NotifyExamSubmitted: the concern is a student in their charge.
func (s *NotificationService) NotifyStudentExamLogout(tenantID, studentID, assignmentID, attemptID int, studentName string) error {
	ctx := s.assignmentContext(tenantID, assignmentID)
	if ctx.StudentName != "" {
		studentName = ctx.StudentName
	}

	var coachUserID *int
	if ctx.CoachUserID >= 0 {
		coachUserID = &ctx.CoachUserID
	}
	recipients, err := s.resolveRecipients(tenantID, coachUserID)
	if err != nil {
		return err
	}

	student := describeStudent(studentName, ctx.StudentCode)
	msg := fmt.Sprintf("%s was logged out during an active exam.", student)
	if ctx.TestTitle != "" {
		msg = fmt.Sprintf("%s was logged out during %q.", student, ctx.TestTitle)
	}

	return s.notifyUsers(
		notificationScopeTenant, &tenantID, recipients,
		EventStudentExamLogout,
		fmt.Sprintf("student_exam_logout:attempt:%d", attemptID),
		"Student Exam Logout", msg, "warning",
		map[string]interface{}{
			"student_id": studentID, "assignment_id": assignmentID, "attempt_id": attemptID,
			"test_title": ctx.TestTitle, "student_name": student,
		},
	)
}

// describeTenant renders a tenant for a platform notification without exposing
// a bare primary key when the name is available.
func describeTenant(tenantID int, tenantName string) string {
	if tenantName != "" {
		return fmt.Sprintf("Tenant %q", tenantName)
	}
	return fmt.Sprintf("Tenant #%d", tenantID)
}

// transitionKey builds the timestamped part of a lifecycle event key. These
// transitions are legitimately repeatable (a tenant can be suspended twice), so
// unlike tenant_created the key must not be fixed.
func transitionKey(prefix string, parts ...interface{}) string {
	key := prefix
	for _, part := range parts {
		key += fmt.Sprintf(":%v", part)
	}
	return key + ":" + time.Now().UTC().Format(time.RFC3339Nano)
}

// NotifyTenantCreated tells every super admin that a new tenant exists.
func (s *NotificationService) NotifyTenantCreated(tenantID int, tenantName string) error {
	tenant := describeTenant(tenantID, tenantName)
	return s.notifyPlatform(
		EventTenantCreated,
		fmt.Sprintf("tenant_created:%d", tenantID),
		"Tenant Created",
		fmt.Sprintf("%s was created.", tenant),
		"info",
		map[string]interface{}{"tenant_id": tenantID, "tenant_name": tenantName},
	)
}

// NotifyTenantSuspended tells every super admin that a tenant was suspended.
func (s *NotificationService) NotifyTenantSuspended(tenantID int, tenantName string) error {
	tenant := describeTenant(tenantID, tenantName)
	return s.notifyPlatform(
		EventTenantSuspended,
		transitionKey("tenant_suspended", tenantID),
		"Tenant Suspended",
		fmt.Sprintf("%s was suspended.", tenant),
		"warning",
		map[string]interface{}{"tenant_id": tenantID, "tenant_name": tenantName},
	)
}

// NotifyTenantReactivated tells every super admin that a tenant was reactivated.
func (s *NotificationService) NotifyTenantReactivated(tenantID int, tenantName string) error {
	tenant := describeTenant(tenantID, tenantName)
	return s.notifyPlatform(
		EventTenantReactivated,
		transitionKey("tenant_reactivated", tenantID),
		"Tenant Reactivated",
		fmt.Sprintf("%s was reactivated.", tenant),
		"info",
		map[string]interface{}{"tenant_id": tenantID, "tenant_name": tenantName},
	)
}

// NotifyPlanChanged tells every super admin that a tenant moved to a new plan.
func (s *NotificationService) NotifyPlanChanged(tenantID int, tenantName string, planID int, planName string) error {
	tenant := describeTenant(tenantID, tenantName)
	plan := fmt.Sprintf("plan #%d", planID)
	if planName != "" {
		plan = fmt.Sprintf("the %q plan", planName)
	}
	return s.notifyPlatform(
		EventPlanChanged,
		transitionKey("plan_changed", tenantID, planID),
		"Plan Changed",
		fmt.Sprintf("%s was moved to %s.", tenant, plan),
		"info",
		map[string]interface{}{
			"tenant_id": tenantID, "tenant_name": tenantName,
			"plan_id": planID, "plan_name": planName,
		},
	)
}

// notifyPlatform fans an event out to every super admin. Platform rows have no
// tenant_id, so they cannot leak tenant data into the platform inbox.
func (s *NotificationService) notifyPlatform(
	eventType string,
	eventKey string,
	title string,
	message string,
	priority string,
	metadata map[string]interface{},
) error {
	recipients, err := s.UserRepo.ListIDsByRole("super_admin")
	if err != nil {
		return err
	}
	return s.notifyUsers(
		notificationScopePlatform, nil, recipients,
		eventType, eventKey, title, message, priority, metadata,
	)
}
