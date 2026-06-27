package repository

import (
	"database/sql"
	"strconv"
)

type CoachRepo struct {
	DB *sql.DB
}

func NewCoachRepo(db *sql.DB) *CoachRepo {
	return &CoachRepo{DB: db}
}

type CoachRow struct {
	CoachID int    `json:"coach_id"`
	UserID  int    `json:"user_id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
}

type CoachDetailRow struct {
	CoachID   int    `json:"coach_id"`
	UserID    int    `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func (r *CoachRepo) GetIDFromUser(userID int) (int, error) {
	var coachID int
	err := r.DB.QueryRow("SELECT id FROM coaches WHERE user_id = $1 AND deleted_at IS NULL", userID).Scan(&coachID)
	return coachID, err
}

func (r *CoachRepo) GetIDAndTenantFromUser(userID int) (int, int, error) {
	var coachID, tenantID int
	err := r.DB.QueryRow("SELECT id, tenant_id FROM coaches WHERE user_id = $1 AND deleted_at IS NULL", userID).Scan(&coachID, &tenantID)
	return coachID, tenantID, err
}

func (r *CoachRepo) Exists(coachID, tenantID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)", coachID, tenantID).Scan(&exists)
	return exists, err
}

func (r *CoachRepo) List(tenantID int, search string, includeDeactivated bool, limit, offset int) ([]CoachRow, int, error) {
	baseQuery := "FROM coaches c JOIN users u ON c.user_id = u.id WHERE c.tenant_id=$1"
	if !includeDeactivated {
		baseQuery += " AND c.deleted_at IS NULL"
	}

	args := []interface{}{tenantID}
	if search != "" {
		baseQuery += " AND c.name ILIKE $2"
		args = append(args, "%"+search+"%")
	}

	var total int
	err := r.DB.QueryRow("SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery := "SELECT c.id, c.user_id, c.name, u.email " + baseQuery + " ORDER BY c.id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var coaches []CoachRow
	for rows.Next() {
		var c2 CoachRow
		if err := rows.Scan(&c2.CoachID, &c2.UserID, &c2.Name, &c2.Email); err != nil {
			return nil, 0, err
		}
		coaches = append(coaches, c2)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return coaches, total, nil
}

func (r *CoachRepo) GetDetail(coachID, tenantID int) (*CoachDetailRow, error) {
	var coach CoachDetailRow
	err := r.DB.QueryRow(`
		SELECT c.id, c.user_id, c.name, u.email, c.created_at
		FROM coaches c JOIN users u ON c.user_id = u.id
		WHERE c.id = $1 AND c.tenant_id = $2 AND c.deleted_at IS NULL
	`, coachID, tenantID).Scan(&coach.CoachID, &coach.UserID, &coach.Name, &coach.Email, &coach.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &coach, nil
}

func (r *CoachRepo) SoftDelete(coachID, tenantID, deletedBy int) (bool, error) {
	result, err := r.DB.Exec(`
		UPDATE coaches SET deleted_at = NOW(), deleted_by = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`, deletedBy, coachID, tenantID)
	if err != nil {
		return false, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func (r *CoachRepo) Create(tenantID, userID int, name string) (int, error) {
	var id int
	err := r.DB.QueryRow(
		"INSERT INTO coaches (tenant_id, user_id, name) VALUES ($1, $2, $3) RETURNING id",
		tenantID, userID, name,
	).Scan(&id)
	return id, err
}

func (r *CoachRepo) Reactivate(coachID, tenantID int) (bool, error) {
	result, err := r.DB.Exec(
		`UPDATE coaches SET deleted_at = NULL, deleted_by = NULL
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NOT NULL`,
		coachID, tenantID,
	)
	if err != nil {
		return false, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}
