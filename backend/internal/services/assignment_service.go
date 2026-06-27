package services

import (
	"ai-student-diagnostic/backend/internal/repository"
)

type AssignmentService struct {
	AssignmentRepo *repository.AssignmentRepo
	StudentRepo    *repository.StudentRepo
	TestPaperRepo       *repository.TestPaperRepo
	CoachRepo      *repository.CoachRepo
	UserRepo       *repository.UserRepo
}

func NewAssignmentService(assignmentRepo *repository.AssignmentRepo, studentRepo *repository.StudentRepo, testPaperRepo *repository.TestPaperRepo, coachRepo *repository.CoachRepo, userRepo *repository.UserRepo) *AssignmentService {
	return &AssignmentService{
		AssignmentRepo: assignmentRepo,
		StudentRepo:    studentRepo,
		TestPaperRepo:       testPaperRepo,
		CoachRepo:      coachRepo,
		UserRepo:       userRepo,
	}
}

// ─────────────────────────────────────────────
// CreateAssignment
// ─────────────────────────────────────────────

type CreateAssignmentInput struct {
	CallerRole string // "admin" or "coach"
	CallerID   int    // user_id from JWT
	StudentID  int
	TestID     int
	CoachID    int // used when CallerRole == "admin"
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
		tenantID, err = s.UserRepo.GetTenantID(input.CallerID)
		if err != nil {
			return 0, &CreateAssignmentError{Status: 500, Message: "failed to fetch tenant info"}
		}
	} else {
		return 0, &CreateAssignmentError{Status: 403, Message: "only admin or coach can assign tests"}
	}

	studentCoachID, studentTenantID, err := s.StudentRepo.GetCoachIDAndTenantID(input.StudentID)
	if err != nil || studentCoachID != coachID || studentTenantID != tenantID {
		return 0, &CreateAssignmentError{Status: 403, Message: "student not found, deactivated, or not in your organization"}
	}

	testCoachID, testTenantID, err := s.StudentRepo.GetTestCoachAndTenant(input.TestID)
	if err != nil || testCoachID != coachID || testTenantID != tenantID {
		return 0, &CreateAssignmentError{Status: 403, Message: "test not found or not in your organization"}
	}

	id, err := s.AssignmentRepo.Create(input.StudentID, input.TestID, coachID)
	if err != nil {
		return 0, &CreateAssignmentError{Status: 500, Message: "failed to create assignment"}
	}

	return id, nil
}
