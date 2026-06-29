package services

import (
	"context"
	"database/sql"

	"google.golang.org/api/idtoken"
)

func (s *AuthService) GoogleLogin(idToken string) (*LoginResult, error) {
	payload, err := idtoken.Validate(context.Background(), idToken, "")
	if err != nil {
		return nil, err
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return nil, sql.ErrNoRows
	}

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
