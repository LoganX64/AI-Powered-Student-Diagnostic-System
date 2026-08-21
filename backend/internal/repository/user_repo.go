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

type UserLoginRow struct {
	UserID   int
	Password string
	Role     string
	TenantID sql.NullInt32
}

func (r *UserRepo) GetByEmailWithCoachCheck(email string) (*UserLoginRow, error) {
	var u UserLoginRow
	err := r.DB.QueryRow(`
		SELECT u.id, u.password, u.role, u.tenant_id
		FROM users u
		WHERE u.email = $1
		  AND (
			u.role <> 'coach'
			OR EXISTS (
				SELECT 1 FROM coaches c
				WHERE c.user_id = u.id AND c.deleted_at IS NULL
			)
		  )
	`, email).Scan(&u.UserID, &u.Password, &u.Role, &u.TenantID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) CreateTenant(name string) (int, error) {
	var id int
	err := r.DB.QueryRow("INSERT INTO tenants (name) VALUES ($1) RETURNING id", name).Scan(&id)
	return id, err
}

func (r *UserRepo) Create(tenantID int, email, hashedPassword, role string) (int, error) {
	var id int
	err := r.DB.QueryRow(
		"INSERT INTO users (tenant_id, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id",
		tenantID, email, hashedPassword, role,
	).Scan(&id)
	return id, err
}

func (r *UserRepo) CreateInTx(tx *sql.Tx, tenantID int, email, hashedPassword, role string) (int, error) {
	var id int
	err := tx.QueryRow(
		"INSERT INTO users (tenant_id, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id",
		tenantID, email, hashedPassword, role,
	).Scan(&id)
	return id, err
}

func (r *UserRepo) GetPasswordHash(userID int) (string, error) {
	var hash string
	err := r.DB.QueryRow("SELECT password FROM users WHERE id = $1", userID).Scan(&hash)
	return hash, err
}

func (r *UserRepo) UpdatePassword(userID int, newHash string) error {
	_, err := r.DB.Exec("UPDATE users SET password = $1 WHERE id = $2", newHash, userID)
	return err
}

func (r *UserRepo) UpdateEmail(tx *sql.Tx, userID int, email string) error {
	_, err := tx.Exec("UPDATE users SET email = $1 WHERE id = $2", email, userID)
	return err
}

func (r *UserRepo) EmailExistsForOther(email string, excludeUserID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND id != $2)",
		email, excludeUserID,
	).Scan(&exists)
	return exists, err
}

func (r *UserRepo) ExistsByID(userID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	return exists, err
}
