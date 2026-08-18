package repository

import (
	"database/sql"
)

type BatchRepo struct {
	DB *sql.DB
}

func NewBatchRepo(db *sql.DB) *BatchRepo {
	return &BatchRepo{DB: db}
}

type BatchRow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Count     int    `json:"student_count"`
}

func (r *BatchRepo) Create(tenantID int, name string) (int, error) {
	var id int
	err := r.DB.QueryRow(
		`INSERT INTO batches (tenant_id, name) VALUES ($1,$2) RETURNING id`,
		tenantID, name,
	).Scan(&id)
	return id, err
}

func (r *BatchRepo) List(tenantID int) ([]BatchRow, error) {
	rows, err := r.DB.Query(`
		SELECT b.id, b.name, b.created_at, COUNT(s.id)
		FROM batches b
		LEFT JOIN students s ON s.batch_id = b.id AND s.deleted_at IS NULL
		WHERE b.tenant_id = $1
		GROUP BY b.id, b.name, b.created_at
		ORDER BY b.id DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BatchRow
	for rows.Next() {
		var b BatchRow
		if err := rows.Scan(&b.ID, &b.Name, &b.CreatedAt, &b.Count); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *BatchRepo) Exists(tenantID, id int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM batches WHERE id=$1 AND tenant_id=$2)`,
		id, tenantID,
	).Scan(&exists)
	return exists, err
}

// Delete removes a batch and clears batch_id on its members.
// Returns the number of students reassigned (batch_id set to NULL).
func (r *BatchRepo) Delete(tenantID, id int) (int, error) {
	res, err := r.DB.Exec(
		`UPDATE students SET batch_id = NULL WHERE batch_id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, raErr := res.RowsAffected()
	if raErr != nil {
		return 0, raErr
	}
	reassigned := int(rowsAffected)

	_, err = r.DB.Exec(
		`DELETE FROM batches WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	return int(reassigned), err
}

// SetStudentBatch transfers a student to a batch (or removes from batch when batchID is nil).
func (r *BatchRepo) SetStudentBatch(tenantID, studentID int, batchID *int) error {
	var query string
	var args []interface{}
	if batchID != nil {
		query = `UPDATE students SET batch_id = $1 WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`
		args = []interface{}{*batchID, studentID, tenantID}
	} else {
		query = `UPDATE students SET batch_id = NULL WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
		args = []interface{}{studentID, tenantID}
	}
	_, err := r.DB.Exec(query, args...)
	return err
}

// MemberIDs returns active student IDs belonging to a batch within a tenant.
func (r *BatchRepo) MemberIDs(tenantID, batchID int) ([]int, error) {
	rows, err := r.DB.Query(
		`SELECT id FROM students WHERE batch_id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		batchID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
