package services

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"context"
	"database/sql"

	"google.golang.org/api/idtoken"
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

func (s *AuthService) RegisterAdmin(email, hashedPassword, orgName string) (int, int, error) {
	tenantID, err := s.UserRepo.CreateTenant(orgName)
	if err != nil {
		return 0, 0, err
	}

	userID, err := s.UserRepo.Create(tenantID, email, hashedPassword, "admin")
	if err != nil {
		return 0, 0, err
	}

	return tenantID, userID, nil
}

func (s *AuthService) RegisterCoach(adminUserID int, email, hashedPassword, name string) (int, int, error) {
	tenantID, err := s.UserRepo.GetTenantID(adminUserID)
	if err != nil {
		return 0, 0, err
	}

	userID, err := s.UserRepo.Create(tenantID, email, hashedPassword, "coach")
	if err != nil {
		return 0, 0, err
	}

	coachID, err := s.CoachRepo.Create(tenantID, userID, name)
	if err != nil {
		return 0, 0, err
	}

	return userID, coachID, nil
}

func (s *AuthService) UpdatePassword(userID int, role, currentPassword, newPassword string) error {
	if role == "coach" {
		_, err := s.CoachRepo.GetIDFromUser(userID)
		if err != nil {
			return err
		}
	}

	currentHash, err := s.UserRepo.GetPasswordHash(userID)
	if err != nil {
		return err
	}

	if err := utils.CheckPassword(currentPassword, currentHash); err != nil {
		return err
	}

	newHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.UserRepo.UpdatePassword(userID, newHash)
}

func (s *AuthService) GoogleLogin(idToken string) (*LoginResult, error) {
	payload, err := idtoken.Validate(context.Background(), idToken, "")
	if err != nil {
		return nil, err
	}

	email := payload.Claims["email"].(string)

	userID, role, tenantID, err := s.UserRepo.GetByEmail(email)
	if err == sql.ErrNoRows {
		name, _ := payload.Claims["name"].(string)
		if name == "" {
			name = "New Organization"
		} else {
			name = name + "'s Organization"
		}

		newTenantID, err := s.UserRepo.CreateTenant(name)
		if err != nil {
			return nil, err
		}

		userID, err = s.UserRepo.Create(newTenantID, email, "", "admin")
		if err != nil {
			return nil, err
		}
		role = "admin"
		tenantID = sql.NullInt32{Int32: int32(newTenantID), Valid: true}
	} else if err != nil {
		return nil, err
	}

	if role == "coach" || role == "student" {
		return nil, sql.ErrNoRows
	}

	return &LoginResult{UserID: userID, Role: role, TenantID: tenantID.Int32}, nil
}
