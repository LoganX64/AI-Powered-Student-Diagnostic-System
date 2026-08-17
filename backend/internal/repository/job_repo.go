package repository

import (
	"database/sql"
)

type JobRepo struct {
	DB *sql.DB
}

func NewJobRepo(db *sql.DB) *JobRepo {
	return &JobRepo{DB: db}
}

type JobRow struct {
	ID        int     `json:"id"`
	TenantID  int     `json:"tenant_id"`
	Type      string  `json:"type"`
	Payload   []byte  `json:"payload"`
	Total     int     `json:"total"`
	Done      int     `json:"done"`
	Failed    int     `json:"failed"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func (r *JobRepo) Create(tenantID int, jobType string, payload []byte, total int) (int, error) {
	var id int
	err := r.DB.QueryRow(`
		INSERT INTO jobs (tenant_id, type, payload, total, status)
		VALUES ($1,$2,$3,$4,'pending') RETURNING id
	`, tenantID, jobType, payload, total).Scan(&id)
	return id, err
}

func (r *JobRepo) Get(id, tenantID int) (JobRow, error) {
	var j JobRow
	err := r.DB.QueryRow(`
		SELECT id, tenant_id, type, payload, total, done, failed, status, created_at, updated_at
		FROM jobs WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&j.ID, &j.TenantID, &j.Type, &j.Payload, &j.Total,
		&j.Done, &j.Failed, &j.Status, &j.CreatedAt, &j.UpdatedAt,
	)
	return j, err
}

// GetByID loads a job by id + tenant (used by the worker, which carries the
// tenant it enqueued the job with).
func (r *JobRepo) GetByID(id, tenantID int) (JobRow, error) {
	var j JobRow
	err := r.DB.QueryRow(`
		SELECT id, tenant_id, type, payload, total, done, failed, status, created_at, updated_at
		FROM jobs WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&j.ID, &j.TenantID, &j.Type, &j.Payload, &j.Total,
		&j.Done, &j.Failed, &j.Status, &j.CreatedAt, &j.UpdatedAt,
	)
	return j, err
}

func (r *JobRepo) Increment(id, tenantID int, doneDelta, failedDelta int) error {
	_, err := r.DB.Exec(`
		UPDATE jobs
		SET done = done + $3, failed = failed + $4, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID, doneDelta, failedDelta)
	return err
}

func (r *JobRepo) SetStatus(id, tenantID int, status string) error {
	_, err := r.DB.Exec(`
		UPDATE jobs SET status = $2, updated_at = NOW() WHERE id = $1 AND tenant_id = $3
	`, id, status, tenantID)
	return err
}
