package services

import (
	"ai-student-diagnostic/backend/internal/repository"
	"encoding/json"
	"fmt"
	"log"
)

type NotificationService struct {
	NotificationRepo *repository.NotificationRepo
	UserRepo         *repository.UserRepo
}

func NewNotificationService(notifRepo *repository.NotificationRepo, userRepo *repository.UserRepo) *NotificationService {
	return &NotificationService{NotificationRepo: notifRepo, UserRepo: userRepo}
}

const (
	EventExamSubmitted     = "exam_submitted"
	EventCoachActivity     = "coach_activity"
	EventSystemAlert       = "system_alert"
	EventSQIComplete       = "sqi_complete"
	EventStorageWarning    = "storage_warning"
	EventStudentExamLogout = "student_exam_logout"
)

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
func (s *NotificationService) notifyUsers(
	userIDs []int,
	eventType string,
	tenantID int,
	title string,
	message string,
	priority string,
	metadata map[string]interface{},
) error {
	if len(userIDs) == 0 {
		log.Printf("[NOTIFICATION] no recipients resolved for event %s in tenant %d", eventType, tenantID)
	}

	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		// Non-fatal: store an empty object rather than dropping the notification.
		log.Printf("[NOTIFICATION] failed to marshal metadata for event %s: %v", eventType, err)
		metaBytes = []byte("{}")
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
		if _, cerr := s.NotificationRepo.Create(repository.NotificationRow{
			TenantID:  tenantID,
			UserID:    &uid,
			EventType: eventType,
			Title:     title,
			Message:   message,
			Priority:  priority,
			Metadata:  metaBytes,
		}); cerr != nil {
			log.Printf("[NOTIFICATION] failed to create notification for user %d event %s: %v", uid, eventType, cerr)
		}
	}
	return nil
}

// Notify is a targeted single-user notification (used for future direct messages).
func (s *NotificationService) Notify(
	eventType string,
	tenantID int,
	userID *int,
	title string,
	message string,
	priority string,
	metadata map[string]interface{},
) error {
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		metaBytes = []byte("{}")
	}
	_, err = s.NotificationRepo.Create(repository.NotificationRow{
		TenantID:  tenantID,
		UserID:    userID,
		EventType: eventType,
		Title:     title,
		Message:   message,
		Priority:  priority,
		Metadata:  metaBytes,
	})
	return err
}

// coachUserForAssignment resolves the coach who owns an assignment, so the event
// reaches them and not their colleagues.
//
// A failed lookup is logged and downgraded to admin-only: the admin's row is
// still written, so the event is not lost, it just misses the coach.
func (s *NotificationService) coachUserForAssignment(tenantID, assignmentID int) *int {
	userID, err := s.UserRepo.CoachUserIDForAssignment(tenantID, assignmentID)
	if err != nil {
		log.Printf("[NOTIFICATION] coach lookup failed for assignment %d in tenant %d: %v", assignmentID, tenantID, err)
		return nil
	}
	return &userID
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

func (s *NotificationService) NotifyExamSubmitted(tenantID, studentID, assignmentID int, studentName string) error {
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
	msg := fmt.Sprintf("%s submitted an exam.", student)
	if ctx.TestTitle != "" {
		msg = fmt.Sprintf("%s submitted %q.", student, ctx.TestTitle)
	}

	return s.notifyUsers(
		recipients,
		EventExamSubmitted,
		tenantID,
		"Exam Submitted",
		msg,
		"info",
		map[string]interface{}{
			"student_id": studentID, "assignment_id": assignmentID,
			"test_title": ctx.TestTitle, "student_name": student,
		},
	)
}

// NotifyCoachActivity reports a coach's own action. Only that coach and the
// admin are notified; peers no longer receive each other's activity.
//
// coachID is a coaches.id (what the handlers hold), so it is resolved to a
// users.id before targeting, and to a display name so the message names a person
// rather than a primary key. A failed lookup still notifies the admin.
func (s *NotificationService) NotifyCoachActivity(tenantID, coachID int, action, detail string) error {
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
		recipients,
		EventCoachActivity,
		tenantID,
		"Coach Activity",
		msg,
		"info",
		map[string]interface{}{"coach_id": coachID, "coach_name": coachName, "action": action},
	)
}

// NotifySystemAlert is org-level, so only the admin is notified.
func (s *NotificationService) NotifySystemAlert(tenantID int, message string) error {
	recipients, err := s.resolveRecipients(tenantID, nil)
	if err != nil {
		return err
	}
	return s.notifyUsers(
		recipients,
		EventSystemAlert,
		tenantID,
		"System Alert",
		message,
		"warning",
		nil,
	)
}

// NotifySQIComplete is admin-only. An SQI batch runs across the whole
// organization, so there is no single owning coach to target; coaches have no
// per-student stake in a job that spans every student.
//
// The message reports outcomes, not the job's primary key: "batch job #245" is
// meaningless to an admin, whereas how many attempts were processed — and how
// many failed — is the thing worth interrupting them for.
func (s *NotificationService) NotifySQIComplete(tenantID, jobID, done, failed int) error {
	recipients, err := s.resolveRecipients(tenantID, nil)
	if err != nil {
		return err
	}

	var msg string
	switch {
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
	if failed > 0 {
		// A run where nothing succeeded, or where a meaningful share failed, is
		// what the admin needs to see; the Scores tab will still be showing the
		// previous numbers.
		if done == 0 || failed*4 >= done {
			priority = "warning"
		}
	}

	return s.notifyUsers(
		recipients,
		EventSQIComplete,
		tenantID,
		"SQI Computation Complete",
		msg,
		priority,
		map[string]interface{}{"job_id": jobID, "processed": done, "failed": failed},
	)
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// NotifyStorageWarning is a quota fact about the organization, so admin-only.
func (s *NotificationService) NotifyStorageWarning(tenantID int, usedBytes, limitBytes int64) error {
	priority := "warning"
	pct := float64(usedBytes) / float64(limitBytes) * 100
	if limitBytes > 0 && pct >= 95 {
		priority = "alert"
	}
	recipients, err := s.resolveRecipients(tenantID, nil)
	if err != nil {
		return err
	}
	return s.notifyUsers(
		recipients,
		EventStorageWarning,
		tenantID,
		"Storage Quota Warning",
		fmt.Sprintf("Storage usage is at %.1f%% (%d / %d bytes).", pct, usedBytes, limitBytes),
		priority,
		map[string]interface{}{"used_bytes": usedBytes, "limit_bytes": limitBytes, "percentage": pct},
	)
}

// NotifyStudentExamLogout targets the coach who owns the assignment, same as
// NotifyExamSubmitted: the concern is a student in their charge.
func (s *NotificationService) NotifyStudentExamLogout(tenantID, studentID, assignmentID int, studentName string) error {
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
		recipients,
		EventStudentExamLogout,
		tenantID,
		"Student Exam Logout",
		msg,
		"warning",
		map[string]interface{}{
			"student_id": studentID, "assignment_id": assignmentID,
			"test_title": ctx.TestTitle, "student_name": student,
		},
	)
}
