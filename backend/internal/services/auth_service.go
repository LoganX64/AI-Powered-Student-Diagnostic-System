package services

import (
	"fmt"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
)

type AuthService struct {
	UserRepo  *repository.UserRepo
	CoachRepo *repository.CoachRepo
}

func NewAuthService(userRepo *repository.UserRepo, coachRepo *repository.CoachRepo) *AuthService {
	return &AuthService{
		UserRepo:  userRepo,
		CoachRepo: coachRepo,
	}
}

type LoginResult struct {
	UserID   int
	Role     string
	TenantID int32
}

func (s *AuthService) UserLogin(email, password string) (*LoginResult, error) {
	u, err := s.UserRepo.GetByEmailWithCoachCheck(email)
	if err != nil {
		return nil, err
	}

	if err := utils.CheckPassword(password, u.Password); err != nil {
		return nil, err
	}

	return &LoginResult{UserID: u.UserID, Role: u.Role, TenantID: u.TenantID.Int32}, nil
}

// RegisterAdmin creates a tenant and its first admin.
//
// Both inserts share one transaction. They used to be committed separately, so a
// duplicate email failed *after* the tenant row was written and left an orphan
// behind — 13 of them accumulated in dev data before it was caught. The rollback
// path is exercised by TestRegisterAdminRollsBackOnDuplicateEmail.
func (s *AuthService) RegisterAdmin(email, hashedPassword, orgName string) (int, int, error) {
	tx, err := s.UserRepo.DB.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("start transaction: %w", err)
	}
	defer tx.Rollback()

	tenantID, err := s.UserRepo.CreateTenantInTx(tx, orgName)
	if err != nil {
		return 0, 0, fmt.Errorf("create tenant: %w", err)
	}

	userID, err := s.UserRepo.CreateInTx(tx, tenantID, email, hashedPassword, "admin")
	if err != nil {
		if repository.IsDuplicateEmail(err) {
			return 0, 0, fmt.Errorf("%w: %s", repository.ErrDuplicateEmail, email)
		}
		return 0, 0, fmt.Errorf("create admin: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}
	return tenantID, userID, nil
}

func (s *AuthService) CreateAdminForTenant(tenantID int, email, hashedPassword, name string) (int, error) {
	exists, err := s.UserRepo.EmailExistsForOther(email, 0)
	if err != nil {
		return 0, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return 0, fmt.Errorf("%w: %s", repository.ErrDuplicateEmail, email)
	}

	userID, err := s.UserRepo.Create(tenantID, email, hashedPassword, "admin")
	if err != nil {
		return 0, fmt.Errorf("create admin: %w", err)
	}
	return userID, nil
}

func (s *AuthService) RegisterCoach(adminUserID int, email, hashedPassword, name string, subjectIDs []int) (int, int, error) {
	tenantID, err := s.UserRepo.GetTenantID(adminUserID)
	if err != nil {
		return 0, 0, err
	}

	tx, err := s.CoachRepo.DB.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	userID, err := s.UserRepo.CreateInTx(tx, tenantID, email, hashedPassword, "coach")
	if err != nil {
		return 0, 0, err
	}

	coachID, err := s.CoachRepo.CreateInTx(tx, tenantID, userID, name)
	if err != nil {
		return 0, 0, err
	}

	if len(subjectIDs) > 0 {
		if err := s.CoachRepo.CreateCoachSubjectsInTx(tx, coachID, subjectIDs); err != nil {
			return 0, 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return userID, coachID, nil
}


