package repository

import (
	"database/sql"
	"fmt"
)

type ProfileRepo struct {
	DB *sql.DB
}

func NewProfileRepo(db *sql.DB) *ProfileRepo {
	return &ProfileRepo{DB: db}
}

type ProfileRow struct {
	UserID     int     `json:"user_id"`
	Email      string  `json:"email"`
	Role       string  `json:"role"`
	DisplayName *string `json:"display_name"`
	Phone      *string `json:"phone"`
	AvatarURL  *string `json:"avatar_url"`
	CreatedAt  string  `json:"created_at"`
	TenantID   *int    `json:"tenant_id"`
	TenantName *string `json:"tenant_name"`
}

func (r *ProfileRepo) GetByUserID(userID int) (*ProfileRow, error) {
	query := `
		SELECT
			u.id, u.email, u.role, up.display_name, up.phone, up.avatar_url,
			u.created_at, u.tenant_id, t.name
		FROM users u
		LEFT JOIN user_profiles up ON up.user_id = u.id
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE u.id = $1`

	var p ProfileRow
	var displayName, phone, avatarURL sql.NullString
	var tenantID sql.NullInt32
	var tenantName sql.NullString

	if err := r.DB.QueryRow(query, userID).Scan(
		&p.UserID, &p.Email, &p.Role, &displayName, &phone, &avatarURL,
		&p.CreatedAt, &tenantID, &tenantName,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile for user %d not found", userID)
		}
		return nil, fmt.Errorf("get profile: %w", err)
	}

	if displayName.Valid {
		v := displayName.String
		p.DisplayName = &v
	}
	if phone.Valid {
		v := phone.String
		p.Phone = &v
	}
	if avatarURL.Valid {
		v := avatarURL.String
		p.AvatarURL = &v
	}
	if tenantID.Valid {
		v := int(tenantID.Int32)
		p.TenantID = &v
	}
	if tenantName.Valid {
		v := tenantName.String
		p.TenantName = &v
	}
	return &p, nil
}

func (r *ProfileRepo) UpdateProfile(userID int, displayName, phone string) error {
	_, err := r.DB.Exec(`
		INSERT INTO user_profiles (user_id, display_name, phone)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			phone = EXCLUDED.phone,
			updated_at = NOW()`,
		userID, nullIfEmpty(displayName), nullIfEmpty(phone))
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

func (r *ProfileRepo) UpdateEmail(userID int, email string) error {
	res, err := r.DB.Exec("UPDATE users SET email = $1 WHERE id = $2", email, userID)
	if err != nil {
		return fmt.Errorf("update email: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("update email rows affected: %w", err)
	} else if n == 0 {
		return fmt.Errorf("user %d not found", userID)
	}
	return nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
