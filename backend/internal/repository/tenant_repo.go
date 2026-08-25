package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type TenantRepo struct {
	DB *sql.DB
}

func NewTenantRepo(db *sql.DB) *TenantRepo {
	return &TenantRepo{DB: db}
}

type TenantRow struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	CreatedAt    string  `json:"created_at"`
	SuspendedAt  *string `json:"suspended_at"`
	PlanID       *int    `json:"plan_id"`
	StudentCount int     `json:"student_count"`
	CoachCount   int     `json:"coach_count"`
	UserCount    int     `json:"user_count"`
}

func (r *TenantRepo) List(search string, limit, offset int) ([]TenantRow, int, error) {
	where := " WHERE ($1 = '' OR t.name ILIKE '%' || $1 || '%')"

	var total int
	if err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM tenants t"+where, search,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	query := `
		SELECT
			t.id, t.name, t.created_at, t.suspended_at, ts.plan_id,
			(SELECT COUNT(*) FROM students s WHERE s.tenant_id = t.id),
			(SELECT COUNT(*) FROM coaches c WHERE c.tenant_id = t.id AND c.deleted_at IS NULL),
			(SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id)
		FROM tenants t
		LEFT JOIN tenant_subscriptions ts ON ts.tenant_id = t.id` + where + `
		ORDER BY t.id DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.DB.Query(query, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	items := []TenantRow{}
	for rows.Next() {
		var t TenantRow
		var suspendedAt sql.NullTime
		var planID sql.NullInt32
		if err := rows.Scan(
			&t.ID, &t.Name, &t.CreatedAt, &suspendedAt, &planID,
			&t.StudentCount, &t.CoachCount, &t.UserCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan tenant row: %w", err)
		}
		if suspendedAt.Valid {
			v := suspendedAt.Time.Format("2006-01-02T15:04:05Z")
			t.SuspendedAt = &v
		}
		if planID.Valid {
			v := int(planID.Int32)
			t.PlanID = &v
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tenant rows: %w", err)
	}
	if items == nil {
		items = []TenantRow{}
	}
	return items, total, nil
}

func (r *TenantRepo) GetByID(tenantID int) (*TenantRow, error) {
	query := `
		SELECT
			t.id, t.name, t.created_at, t.suspended_at, ts.plan_id,
			(SELECT COUNT(*) FROM students s WHERE s.tenant_id = t.id),
			(SELECT COUNT(*) FROM coaches c WHERE c.tenant_id = t.id AND c.deleted_at IS NULL),
			(SELECT COUNT(*) FROM users u WHERE u.tenant_id = t.id)
		FROM tenants t
		LEFT JOIN tenant_subscriptions ts ON ts.tenant_id = t.id
		WHERE t.id = $1`

	var t TenantRow
	var suspendedAt sql.NullTime
	var planID sql.NullInt32
	if err := r.DB.QueryRow(query, tenantID).Scan(
		&t.ID, &t.Name, &t.CreatedAt, &suspendedAt, &planID,
		&t.StudentCount, &t.CoachCount, &t.UserCount,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	if suspendedAt.Valid {
		v := suspendedAt.Time.Format("2006-01-02T15:04:05Z")
		t.SuspendedAt = &v
	}
	if planID.Valid {
		v := int(planID.Int32)
		t.PlanID = &v
	}
	return &t, nil
}

func (r *TenantRepo) Update(tenantID int, name string) error {
	res, err := r.DB.Exec("UPDATE tenants SET name = $1 WHERE id = $2", name, tenantID)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("update tenant rows affected: %w", err)
	} else if n == 0 {
		return fmt.Errorf("tenant %d not found", tenantID)
	}
	return nil
}

func (r *TenantRepo) Suspend(tenantID int) error {
	res, err := r.DB.Exec("UPDATE tenants SET suspended_at = NOW() WHERE id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("suspend tenant: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("suspend tenant rows affected: %w", err)
	} else if n == 0 {
		return fmt.Errorf("tenant %d not found", tenantID)
	}
	return nil
}

func (r *TenantRepo) Reactivate(tenantID int) error {
	res, err := r.DB.Exec("UPDATE tenants SET suspended_at = NULL WHERE id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("reactivate tenant: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("reactivate tenant rows affected: %w", err)
	} else if n == 0 {
		return fmt.Errorf("tenant %d not found", tenantID)
	}
	return nil
}

func (r *TenantRepo) GetAdmins(tenantID int) ([]UserRow, error) {
	rows, err := r.DB.Query(`
		SELECT id, email, role, created_at
		FROM users
		WHERE tenant_id = $1 AND role = 'admin'
		ORDER BY id ASC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant admins: %w", err)
	}
	defer rows.Close()

	admins := []UserRow{}
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan admin row: %w", err)
		}
		admins = append(admins, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin rows: %w", err)
	}
	if admins == nil {
		admins = []UserRow{}
	}
	return admins, nil
}

func (r *TenantRepo) GetGlobalStats() (map[string]int, error) {
	stats := map[string]int{}
	queries := map[string]string{
		"tenants":  "SELECT COUNT(*) FROM tenants",
		"users":    "SELECT COUNT(*) FROM users",
		"students": "SELECT COUNT(*) FROM students",
		"coaches":  "SELECT COUNT(*) FROM coaches WHERE deleted_at IS NULL",
	}
	for key, q := range queries {
		var n int
		if err := r.DB.QueryRow(q).Scan(&n); err != nil {
			return nil, fmt.Errorf("global stat %s: %w", key, err)
		}
		stats[key] = n
	}
	return stats, nil
}

func (r *TenantRepo) GetSettings(tenantID int) (map[string]interface{}, error) {
	rows, err := r.DB.Query(
		"SELECT setting_key, setting_value FROM tenant_settings WHERE tenant_id = $1",
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("get tenant settings: %w", err)
	}
	defer rows.Close()

	settings := map[string]interface{}{}
	for rows.Next() {
		var key string
		var raw []byte
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, fmt.Errorf("scan tenant setting: %w", err)
		}
		var val interface{}
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, fmt.Errorf("unmarshal tenant setting %s: %w", key, err)
		}
		settings[key] = val
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant settings: %w", err)
	}
	if settings == nil {
		settings = map[string]interface{}{}
	}
	return settings, nil
}

func (r *TenantRepo) UpsertSetting(tenantID int, key string, value interface{}) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal setting value: %w", err)
	}
	_, err = r.DB.Exec(`
		INSERT INTO tenant_settings (tenant_id, setting_key, setting_value)
		VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (tenant_id, setting_key) DO UPDATE SET
			setting_value = EXCLUDED.setting_value,
			updated_at = NOW()`,
		tenantID, key, string(raw))
	if err != nil {
		return fmt.Errorf("upsert tenant setting: %w", err)
	}
	return nil
}
