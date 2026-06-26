package services

import (
	"ai-student-diagnostic/backend/internal/repository"
)

type TestService struct {
	TestRepo *repository.TestRepo
}

func NewTestService(testRepo *repository.TestRepo) *TestService {
	return &TestService{TestRepo: testRepo}
}

func (s *TestService) Create(tenantID int, title string, subjectID, coachID, duration int, examDate *string) (int, error) {
	return s.TestRepo.Create(tenantID, title, subjectID, coachID, duration, examDate)
}

func (s *TestService) List(tenantID int, coachID *int, search string, limit, offset int) ([]repository.TestRow, int, error) {
	return s.TestRepo.List(tenantID, coachID, search, limit, offset)
}

func (s *TestService) GetDetail(testID, tenantID int) (*repository.TestDetailRow, error) {
	return s.TestRepo.GetDetail(testID, tenantID)
}

func (s *TestService) ListQuestions(testID int, limit, offset int) ([]repository.QuestionRow, int, error) {
	return s.TestRepo.ListQuestions(testID, limit, offset)
}

func (s *TestService) CreateQuestions(testID int, questions []repository.QuestionRequest) ([]int, error) {
	return s.TestRepo.CreateQuestions(testID, questions)
}

func (s *TestService) Update(testID, tenantID int, title string, subjectID, coachID, duration int, examDate *string) (bool, error) {
	return s.TestRepo.Update(testID, tenantID, title, subjectID, coachID, duration, examDate)
}

func (s *TestService) Delete(testID, tenantID int) (bool, error) {
	return s.TestRepo.Delete(testID, tenantID)
}

func (s *TestService) UpdateQuestion(questionID, testID int, req repository.QuestionRequest) (bool, error) {
	return s.TestRepo.UpdateQuestion(questionID, testID, req)
}

func (s *TestService) DeleteQuestion(questionID, testID int) (bool, error) {
	return s.TestRepo.DeleteQuestion(questionID, testID)
}

func (s *TestService) ListByCoach(coachID, tenantID, limit, offset int) ([]repository.TestRow, int, error) {
	return s.TestRepo.ListByCoach(coachID, tenantID, limit, offset)
}

func (s *TestService) CreateSubject(tenantID int, name string) (int, error) {
	return s.TestRepo.CreateSubject(tenantID, name)
}

func (s *TestService) ListSubjects(tenantID int, search string, limit, offset int) ([]repository.SubjectRow, int, error) {
	return s.TestRepo.ListSubjects(tenantID, search, limit, offset)
}
