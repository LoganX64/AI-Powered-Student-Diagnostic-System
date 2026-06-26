package services

import (
	"ai-student-diagnostic/backend/internal/repository"
)

type CoachService struct {
	CoachRepo *repository.CoachRepo
	UserRepo  *repository.UserRepo
}

func NewCoachService(coachRepo *repository.CoachRepo, userRepo *repository.UserRepo) *CoachService {
	return &CoachService{
		CoachRepo: coachRepo,
		UserRepo:  userRepo,
	}
}

func (s *CoachService) List(tenantID int, search string, includeDeactivated bool, limit, offset int) ([]repository.CoachRow, int, error) {
	return s.CoachRepo.List(tenantID, search, includeDeactivated, limit, offset)
}

func (s *CoachService) GetDetail(coachID, tenantID int) (*repository.CoachDetailRow, error) {
	return s.CoachRepo.GetDetail(coachID, tenantID)
}

func (s *CoachService) SoftDelete(coachID, tenantID, deletedBy int) (bool, error) {
	return s.CoachRepo.SoftDelete(coachID, tenantID, deletedBy)
}

func (s *CoachService) Reactivate(coachID, tenantID int) (bool, error) {
	return s.CoachRepo.Reactivate(coachID, tenantID)
}
