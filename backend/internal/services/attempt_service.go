package services

import (
	"ai-student-diagnostic/backend/internal/repository"
)

type AttemptService struct {
	AttemptRepo *repository.AttemptRepo
}

func NewAttemptService(attemptRepo *repository.AttemptRepo) *AttemptService {
	return &AttemptService{AttemptRepo: attemptRepo}
}

func (s *AttemptService) GetByAssignment(assignmentID int) (int, interface{}, error) {
	return s.AttemptRepo.GetByAssignment(assignmentID)
}

func (s *AttemptService) ExistsByAssignment(assignmentID int) (bool, error) {
	return s.AttemptRepo.ExistsByAssignment(assignmentID)
}

func (s *AttemptService) GetSQIResult(attemptID int) (interface{}, interface{}, error) {
	return s.AttemptRepo.GetSQIResult(attemptID)
}

func (s *AttemptService) GetAnswerDetails(attemptID int) ([]repository.AnswerDetail, error) {
	return s.AttemptRepo.GetAnswerDetails(attemptID)
}
