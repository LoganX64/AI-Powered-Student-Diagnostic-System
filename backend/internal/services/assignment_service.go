package services

import (
	"ai-student-diagnostic/backend/internal/repository"
	"errors"
	"log"

	"github.com/lib/pq"
)

type AssignmentService struct {
	AssignmentRepo *repository.AssignmentRepo
	StudentRepo    *repository.StudentRepo
	TestPaperRepo       *repository.TestPaperRepo
	CoachRepo      *repository.CoachRepo
	UserRepo       *repository.UserRepo
	RedisEnabled   bool
}

func NewAssignmentService(assignmentRepo *repository.AssignmentRepo, studentRepo *repository.StudentRepo, testPaperRepo *repository.TestPaperRepo, coachRepo *repository.CoachRepo, userRepo *repository.UserRepo, redisEnabled bool) *AssignmentService {
	return &AssignmentService{
		AssignmentRepo: assignmentRepo,
		StudentRepo:    studentRepo,
		TestPaperRepo:       testPaperRepo,
		CoachRepo:      coachRepo,
		UserRepo:       userRepo,
		RedisEnabled:   redisEnabled,
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
