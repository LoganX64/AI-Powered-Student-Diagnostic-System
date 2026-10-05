package repository

import (
	"database/sql"
	"time"
)

type PasswordResetRepo struct {
	DB *sql.DB
}

func NewPasswordResetRepo(db *sql.DB) *PasswordResetRepo {
	return &PasswordResetRepo{DB: db}
}

func (r *PasswordResetRepo) Create(email, tokenHash string, expiresAt time.Time) error {
	_, err := r.DB.Exec("INSERT INTO password_resets (email, token_hash, expires_at) VALUES ($1,$2,$3)", email, tokenHash, expiresAt)
	return err
}

func (r *PasswordResetRepo) FindValid(tokenHash string) (string, error) {
	var email string
	err := r.DB.QueryRow("SELECT email FROM password_resets WHERE token_hash=$1 AND used_at IS NULL AND expires_at > NOW()", tokenHash).Scan(&email)
	return email, err
}

func (r *PasswordResetRepo) MarkUsed(tokenHash string) error {
	_, err := r.DB.Exec("UPDATE password_resets SET used_at = NOW() WHERE token_hash=$1", tokenHash)
	return err
}
