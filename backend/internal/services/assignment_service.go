package services

import (
	"ai-student-diagnostic/backend/internal/repository"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/lib/pq"
)

// proctoringBytesPerMinute is the fixed recording rate (500 kbps / 480x360),
// ~3.75 MB/min. Mirrors frontend StudentQuizPage.tsx:391.
const proctoringBytesPerMinute = 3_750_000

type AssignmentService struct {
	AssignmentRepo    *repository.AssignmentRepo
	StudentRepo       *repository.StudentRepo
	TestPaperRepo     *repository.TestPaperRepo
	CoachRepo         *repository.CoachRepo
	UserRepo          *repository.UserRepo
	SubscriptionRepo  *repository.SubscriptionRepo
	RedisEnabled      bool
}

func NewAssignmentService(assignmentRepo *repository.AssignmentRepo, studentRepo *repository.StudentRepo, testPaperRepo *repository.TestPaperRepo, coachRepo *repository.CoachRepo, userRepo *repository.UserRepo, subscriptionRepo *repository.SubscriptionRepo, redisEnabled bool) *AssignmentService {
	return &AssignmentService{
		AssignmentRepo:   assignmentRepo,
		StudentRepo:      studentRepo,
		TestPaperRepo:    testPaperRepo,
		CoachRepo:        coachRepo,
		UserRepo:         userRepo,
		SubscriptionRepo: subscriptionRepo,
		RedisEnabled:     redisEnabled,
	}
}

// ─────────────────────────────────────────────
// CreateAssignment
// ─────────────────────────────────────────────

type CreateAssignmentInput struct {
	CallerRole       string // "admin" or "coach"
	CallerID         int    // user_id from JWT
	TenantID         int    // tenant_id from JWT
	StudentID        int
	TestID           int
	CoachID          int // used when CallerRole == "admin"
	IntegrityPolicy  []byte
	EstimatedCost    float64
	DeliveryMode     string
}

type CreateBatchAssignmentInput struct {
	CallerRole      string
	CallerID        int
	TenantID        int
	TestID          int
	CoachID         int
	StudentIDs      []int
	IntegrityPolicy []byte
	EstimatedCost   float64
	DeliveryMode    string
}

type CreateAssignmentError struct {
	Status  int
	Message string
}

func (e *CreateAssignmentError) Error() string {
	return e.Message
}

func (s *AssignmentService) CreateAssignment(input CreateAssignmentInput) (int, error) {
	var coachID int
	var tenantID int
	var err error

	if input.CallerRole == "coach" {
		coachID, tenantID, err = s.CoachRepo.GetIDAndTenantFromUser(input.CallerID)
		if err != nil {
			return 0, &CreateAssignmentError{Status: 401, Message: "coach not found"}
		}
	} else if input.CallerRole == "admin" {
		coachID = input.CoachID
		tenantID = input.TenantID
	} else {
		return 0, &CreateAssignmentError{Status: 403, Message: "only admin or coach can assign tests"}
	}

	if err := s.guardScaleMode(input.DeliveryMode); err != nil {
		return 0, err
	}

	studentCoachID, studentTenantID, err := s.StudentRepo.GetCoachIDAndTenantID(input.StudentID)
	if err != nil {
		log.Printf("[ASSIGNMENT] GetCoachIDAndTenantID failed for student %d: %v", input.StudentID, err)
		return 0, &CreateAssignmentError{Status: 500, Message: "failed to verify student"}
	}
	if studentCoachID != coachID || studentTenantID != tenantID {
		return 0, &CreateAssignmentError{Status: 403, Message: "student not found, deactivated, or not in your organization"}
	}

	testCoachID, testTenantID, err := s.TestPaperRepo.GetCoachAndTenant(input.TestID)
	if err != nil {
		log.Printf("[ASSIGNMENT] GetCoachAndTenant failed for test %d: %v", input.TestID, err)
		return 0, &CreateAssignmentError{Status: 500, Message: "failed to verify test"}
	}
	if testCoachID != coachID || testTenantID != tenantID {
		return 0, &CreateAssignmentError{Status: 403, Message: "test not found or not in your organization"}
	}

	// Project proctoring storage at creation; block only if it would exceed the cap.
	if err := s.guardStorageForProctoring(tenantID, input.TestID, 1); err != nil {
		return 0, err
	}

	id, err := s.AssignmentRepo.Create(input.StudentID, input.TestID, coachID, input.IntegrityPolicy, input.EstimatedCost, input.DeliveryMode)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, &CreateAssignmentError{Status: 409, Message: "student is already assigned to this test"}
		}
		return 0, &CreateAssignmentError{Status: 500, Message: "failed to create assignment"}
	}

	return id, nil
}

// CreateBatchAssignment expands a set of student IDs (and optionally cohort/batch
// IDs) into assignments for one test, deduped and dup-guarded at the DB level.
func (s *AssignmentService) CreateBatchAssignment(input CreateBatchAssignmentInput) (int, error) {
	var coachID, tenantID int
	var err error

	if input.CallerRole == "coach" {
		coachID, tenantID, err = s.CoachRepo.GetIDAndTenantFromUser(input.CallerID)
		if err != nil {
			return 0, &CreateAssignmentError{Status: 401, Message: "coach not found"}
		}
	} else if input.CallerRole == "admin" {
		coachID = input.CoachID
		tenantID = input.TenantID
	} else {
		return 0, &CreateAssignmentError{Status: 403, Message: "only admin or coach can assign tests"}
	}

	if err := s.guardScaleMode(input.DeliveryMode); err != nil {
		return 0, err
	}

	testCoachID, testTenantID, err := s.TestPaperRepo.GetCoachAndTenant(input.TestID)
	if err != nil || testCoachID != coachID || testTenantID != tenantID {
		return 0, &CreateAssignmentError{Status: 403, Message: "test not found or not in your organization"}
	}

	// Validate every student belongs to the caller's organization.
	valid := make([]int, 0, len(input.StudentIDs))
	seen := make(map[int]bool, len(input.StudentIDs))
	for _, sid := range input.StudentIDs {
		if seen[sid] {
			continue
		}
		seen[sid] = true
		studentCoachID, studentTenantID, err := s.StudentRepo.GetCoachIDAndTenantID(sid)
		if err != nil || studentCoachID != coachID || studentTenantID != tenantID {
			return 0, &CreateAssignmentError{Status: 403, Message: "one or more students are not in your organization"}
		}
		valid = append(valid, sid)
	}

	if len(valid) == 0 {
		return 0, &CreateAssignmentError{Status: 400, Message: "no valid students to assign"}
	}

	// Project proctoring storage across the whole batch; block only if it would
	// exceed the plan cap. No-op when the plan does not include proctoring.
	if err := s.guardStorageForProctoring(tenantID, input.TestID, len(valid)); err != nil {
		return 0, err
	}

	created, err := s.AssignmentRepo.CreateBatch(valid, input.TestID, coachID, input.IntegrityPolicy, input.EstimatedCost, input.DeliveryMode)
	if err != nil {
		return 0, &CreateAssignmentError{Status: 500, Message: "failed to create assignments"}
	}
	return created, nil
}

// DeliveryModeForN derives the delivery_mode from the student count N.
func DeliveryModeForN(n, scaleBandC int) string {
	if scaleBandC > 0 && n >= scaleBandC {
		return "scale"
	}
	return "standard"
}

// guardScaleMode enforces the plan rule: Band C (scale) delivery requires Redis.
// Without Redis, block and tell the caller to use staggered sub-batches.
func (s *AssignmentService) guardScaleMode(deliveryMode string) error {
	if deliveryMode == "scale" && !s.RedisEnabled {
		return &CreateAssignmentError{
			Status:  400,
			Message: "scale (Band C) delivery requires Redis; either set delivery_mode='standard' or split the cohort into staggered sub-batches",
		}
	}
	return nil
}

// guardStorageForProctoring projects the storage a video-proctored assignment
// will consume and blocks (402) only when it would exceed the plan cap. It is a
// projection only — no storage is written here. The path is a no-op when the
// plan does not include video proctoring, or when there is no subscription
// (Free default, which never accrues proctoring storage).
func (s *AssignmentService) guardStorageForProctoring(tenantID, testID, studentCount int) error {
	if s.SubscriptionRepo == nil {
		return nil
	}
	sub, err := s.SubscriptionRepo.GetByTenantID(tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // no subscription → Free default → no proctoring
		}
		return &CreateAssignmentError{Status: 500, Message: "failed to check storage quota"}
	}
	if !sub.VideoProctoringIncluded {
		return nil
	}

	duration, err := s.TestPaperRepo.GetDuration(testID)
	if err != nil {
		return &CreateAssignmentError{Status: 500, Message: "failed to read test duration"}
	}

	est := int64(proctoringBytesPerMinute) * int64(duration) * int64(studentCount)
	if sub.StorageUsedBytes+est > sub.StorageLimitBytes {
		return &CreateAssignmentError{
			Status:  http.StatusPaymentRequired,
			Message: "video proctoring storage quota exceeded for this plan; upgrade or reduce the assignment scope",
		}
	}
	return nil
}
