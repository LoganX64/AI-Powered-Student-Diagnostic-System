package services

import (
	"ai-student-diagnostic/backend/internal/repository"
)

type AssignmentService struct {
	AssignmentRepo *repository.AssignmentRepo
	StudentRepo    *repository.StudentRepo
	TestRepo       *repository.TestRepo
}

func NewAssignmentService(assignmentRepo *repository.AssignmentRepo, studentRepo *repository.StudentRepo, testRepo *repository.TestRepo) *AssignmentService {
	return &AssignmentService{
		AssignmentRepo: assignmentRepo,
		StudentRepo:    studentRepo,
		TestRepo:       testRepo,
	}
}

func (s *AssignmentService) Create(studentID, testID, coachID int) (int, error) {
	return s.AssignmentRepo.Create(studentID, testID, coachID)
}

func (s *AssignmentService) ListByStudent(studentID int, coachID *int, limit, offset int) ([]repository.AssignmentRow, int, error) {
	return s.AssignmentRepo.ListByStudent(studentID, coachID, limit, offset)
}

func (s *AssignmentService) ListAll(tenantID int, coachID *int, testIDStr string, limit, offset int) ([]repository.AssignmentDetailRow, int, error) {
	return s.AssignmentRepo.ListAll(tenantID, coachID, testIDStr, limit, offset)
}

func (s *AssignmentService) GetByID(assignmentID, studentID int) (int, int, string, string, string, error) {
	return s.AssignmentRepo.GetByID(assignmentID, studentID)
}

func (s *AssignmentService) GetByIDForCoach(assignmentID, studentID, coachID int) (int, string, string, string, error) {
	return s.AssignmentRepo.GetByIDForCoach(assignmentID, studentID, coachID)
}
