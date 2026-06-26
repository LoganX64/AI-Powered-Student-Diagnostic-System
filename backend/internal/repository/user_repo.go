package repository

import "database/sql"

type UserRepo struct {
	DB *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{DB: db}
}

func (r *UserRepo) GetTenantID(userID int) (int, error) {
	var tenantID int
	err := r.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	return tenantID, err
}

func (r *UserRepo) EmailExists(email string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", email).Scan(&exists)
	return exists, err
}
