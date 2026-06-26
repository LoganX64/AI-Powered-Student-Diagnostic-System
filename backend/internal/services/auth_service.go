package services

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"context"
	"database/sql"

	"google.golang.org/api/idtoken"
)

type AuthService struct {
	UserRepo *repository.UserRepo
	CoachRepo *repository.CoachRepo
	DB       *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{
		UserRepo:  repository.NewUserRepo(db),
		CoachRepo: repository.NewCoachRepo(db),
		DB:        db,
	}
}

type LoginResult struct {
	UserID   int
	Role     string
	TenantID int32
}

func (s *AuthService) UserLogin(email, password string) (*LoginResult, error) {
	var userID int
	var hashedPassword string
	var role string
	var tenantID sql.NullInt32

	err := s.DB.QueryRow(`
		SELECT u.id, u.password, u.role, u.tenant_id
		FROM users u
		WHERE u.email = $1
		  AND (
			u.role <> 'coach'
			OR EXISTS (
				SELECT 1
				FROM coaches c
				WHERE c.user_id = u.id AND c.deleted_at IS NULL
			)
		  )
	`, email).Scan(&userID, &hashedPassword, &role, &tenantID)
	if err != nil {
		return nil, err
	}

	if err := utils.CheckPassword(password, hashedPassword); err != nil {
		return nil, err
	}

	return &LoginResult{UserID: userID, Role: role, TenantID: tenantID.Int32}, nil
}

func (s *AuthService) RegisterAdmin(email, hashedPassword, orgName string) (int, int, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	var tenantID int
	err = tx.QueryRow("INSERT INTO tenants (name) VALUES ($1) RETURNING id", orgName).Scan(&tenantID)
	if err != nil {
		return 0, 0, err
	}

	var userID int
	err = tx.QueryRow(`INSERT INTO users (tenant_id, email, password, role) VALUES ($1, $2, $3, 'admin') RETURNING id`, tenantID, email, hashedPassword).Scan(&userID)
	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return tenantID, userID, nil
}

func (s *AuthService) RegisterCoach(adminUserID int, email, hashedPassword, name string) (int, int, error) {
	var tenantID sql.NullInt32
	err := s.DB.QueryRow("SELECT tenant_id FROM users WHERE id = $1", adminUserID).Scan(&tenantID)
	if err != nil || !tenantID.Valid {
		return 0, 0, err
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	var newUserID int
	err = tx.QueryRow(`INSERT INTO users (tenant_id, email, password, role) VALUES ($1, $2, $3, 'coach') RETURNING id`, tenantID.Int32, email, hashedPassword).Scan(&newUserID)
	if err != nil {
		return 0, 0, err
	}

	var coachID int
	err = tx.QueryRow(`INSERT INTO coaches (tenant_id, user_id, name) VALUES ($1, $2, $3) RETURNING id`, tenantID.Int32, newUserID, name).Scan(&coachID)
	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return newUserID, coachID, nil
}

func (s *AuthService) UpdatePassword(userID int, role, currentPassword, newPassword string) error {
	if role == "coach" {
		var active bool
		err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE user_id = $1 AND deleted_at IS NULL)", userID).Scan(&active)
		if err != nil || !active {
			return err
		}
	}

	var currentHash string
	err := s.DB.QueryRow("SELECT password FROM users WHERE id = $1", userID).Scan(&currentHash)
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

	_, err = s.DB.Exec("UPDATE users SET password = $1 WHERE id = $2", newHash, userID)
	return err
}

func (s *AuthService) GoogleLogin(idToken string) (*LoginResult, error) {
	payload, err := idtoken.Validate(context.Background(), idToken, "")
	if err != nil {
		return nil, err
	}

	email := payload.Claims["email"].(string)

	var userID int
	var role string
	var tenantID sql.NullInt32

	err = s.DB.QueryRow(`SELECT id, role, tenant_id FROM users WHERE email = $1`, email).Scan(&userID, &role, &tenantID)
	if err == sql.ErrNoRows {
		name, _ := payload.Claims["name"].(string)
		if name == "" {
			name = "New Organization"
		} else {
			name = name + "'s Organization"
		}

		var newTenantID int
		err = s.DB.QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, name).Scan(&newTenantID)
		if err != nil {
			return nil, err
		}

		err = s.DB.QueryRow(`INSERT INTO users (tenant_id, email, role) VALUES ($1, $2, 'admin') RETURNING id`, newTenantID, email).Scan(&userID)
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
