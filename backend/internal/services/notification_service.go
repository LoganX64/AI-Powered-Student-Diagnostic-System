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

// NotifyTenant fans out a broadcast notification to every admin/coach user in a
// tenant, skipping any user who has disabled that event_type in their
// preferences. Each recipient gets their own row (user_id set) so the list and
// unread-count queries filter correctly per viewer. Errors are logged, not
// fatal: a notification failure must never break the triggering action.
func (s *NotificationService) NotifyTenant(
	eventType string,
	tenantID int,
	title string,
	message string,
	priority string,
	metadata map[string]interface{},
) error {
	userIDs, err := s.UserRepo.ListIDsByTenantRoles(tenantID, []string{"admin", "coach"})
	if err != nil {
		log.Printf("[NOTIFICATION] failed to list tenant users for tenant %d: %v", tenantID, err)
		return err
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

func (s *NotificationService) NotifyExamSubmitted(tenantID, studentID, assignmentID int, studentName string) error {
	return s.NotifyTenant(
		EventExamSubmitted,
		tenantID,
		"Exam Submitted",
		fmt.Sprintf("Student %s (ID: %d) has submitted their exam for assignment %d.", studentName, studentID, assignmentID),
		"info",
		map[string]interface{}{"student_id": studentID, "assignment_id": assignmentID},
	)
}

func (s *NotificationService) NotifyCoachActivity(tenantID, coachID int, action, detail string) error {
	return s.NotifyTenant(
		EventCoachActivity,
		tenantID,
		"Coach Activity",
		fmt.Sprintf("Coach (ID: %d) %s: %s", coachID, action, detail),
		"info",
		map[string]interface{}{"coach_id": coachID, "action": action},
	)
}

func (s *NotificationService) NotifySystemAlert(tenantID int, message string) error {
	return s.NotifyTenant(
		EventSystemAlert,
		tenantID,
		"System Alert",
		message,
		"warning",
		nil,
	)
}

func (s *NotificationService) NotifySQIComplete(tenantID, jobID int, detail string) error {
	return s.NotifyTenant(
		EventSQIComplete,
		tenantID,
		"SQI Computation Complete",
		fmt.Sprintf("SQI batch job #%d has completed. %s", jobID, detail),
		"info",
		map[string]interface{}{"job_id": jobID},
	)
}

func (s *NotificationService) NotifyStorageWarning(tenantID int, usedBytes, limitBytes int64) error {
	priority := "warning"
	pct := float64(usedBytes) / float64(limitBytes) * 100
	if limitBytes > 0 && pct >= 95 {
		priority = "alert"
	}
	return s.NotifyTenant(
		EventStorageWarning,
		tenantID,
		"Storage Quota Warning",
		fmt.Sprintf("Storage usage is at %.1f%% (%d / %d bytes).", pct, usedBytes, limitBytes),
		priority,
		map[string]interface{}{"used_bytes": usedBytes, "limit_bytes": limitBytes, "percentage": pct},
	)
}

func (s *NotificationService) NotifyStudentExamLogout(tenantID, studentID, assignmentID int, studentName string) error {
	return s.NotifyTenant(
		EventStudentExamLogout,
		tenantID,
		"Student Exam Logout",
		fmt.Sprintf("Student %s (ID: %d) logged out during an active exam for assignment %d.", studentName, studentID, assignmentID),
		"warning",
		map[string]interface{}{"student_id": studentID, "assignment_id": assignmentID},
	)
}
