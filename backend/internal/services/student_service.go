package services

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/helper"
)

type StudentService struct {
	StudentRepo *repository.StudentRepo
	CoachRepo   *repository.CoachRepo
	UserRepo    *repository.UserRepo
}

func NewStudentService(studentRepo *repository.StudentRepo, coachRepo *repository.CoachRepo, userRepo *repository.UserRepo) *StudentService {
	return &StudentService{
		StudentRepo: studentRepo,
		CoachRepo:   coachRepo,
		UserRepo:    userRepo,
	}
}

func (s *StudentService) Create(tenantID int, name, studentCode string, coachID int) (int, error) {
	return s.StudentRepo.Create(tenantID, name, studentCode, coachID)
}

func (s *StudentService) List(tenantID int, coachID *int, includeDeactivated bool, limit, offset int) ([]repository.StudentRow, int, error) {
	return s.StudentRepo.List(tenantID, coachID, includeDeactivated, limit, offset)
}

func (s *StudentService) GetDetail(studentID, tenantID int, coachID *int) (*repository.StudentDetailRow, error) {
	return s.StudentRepo.GetDetail(studentID, tenantID, coachID)
}

func (s *StudentService) SoftDelete(studentID, tenantID, deletedBy int, coachID *int) (bool, error) {
	return s.StudentRepo.SoftDelete(studentID, tenantID, deletedBy, coachID)
}

func (s *StudentService) Reactivate(studentID, tenantID int, coachID *int) (bool, error) {
	return s.StudentRepo.Reactivate(studentID, tenantID, coachID)
}

func (s *StudentService) GetSQI(studentID, tenantID int, includeAnalysis, compute bool) (*SQIResult, error) {
	name, err := s.StudentRepo.GetName(studentID, tenantID)
	if err != nil {
		return nil, err
	}

	// TODO: compute SQI if needed
	_ = compute
	_ = includeAnalysis

	return &SQIResult{
		StudentID: studentID,
		Name:      name,
	}, nil
}

type SQIResult struct {
	StudentID int
	Name      string
}

func (s *StudentService) ListCoachStudents(coachID, tenantID, limit, offset int) ([]repository.StudentRow, int, error) {
	return s.StudentRepo.List(tenantID, &coachID, false, limit, offset)
}

func (s *StudentService) GetStudentSQIAverage(studentID int) float64 {
	// placeholder - actual SQI calculation is in attempt_service
	return 0
}

func RoundSQI(v float64) float64 {
	return helper.Round2V2(v)
}
