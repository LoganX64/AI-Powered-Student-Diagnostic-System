package repository

import "database/sql"

type SubscriptionRepo struct {
	DB *sql.DB
}

func NewSubscriptionRepo(db *sql.DB) *SubscriptionRepo {
	return &SubscriptionRepo{DB: db}
}

type SubscriptionRow struct {
	ID                     int     `json:"id"`
	TenantID               int     `json:"tenant_id"`
	PlanID                 int     `json:"plan_id"`
	PlanName               string  `json:"plan_name"`
	Status                 string  `json:"status"`
	RazorpaySubscriptionID *string `json:"razorpay_subscription_id"`
	CurrentPeriodStart     *string `json:"current_period_start"`
	CurrentPeriodEnd       *string `json:"current_period_end"`
	StudentCount           int     `json:"student_count"`
	CoachCount             int     `json:"coach_count"`
	StorageUsedBytes       int64   `json:"storage_used_bytes"`
	TestCountThisMonth     int     `json:"test_count_this_month"`
	// Plan limits (joined from subscription_plans)
	StudentLimit            int   `json:"student_limit"`
	CoachLimit              int   `json:"coach_limit"`
	StorageLimitBytes       int64 `json:"storage_limit_bytes"`
	TestLimit               int   `json:"test_limit"`
	SQIAccess               bool  `json:"sqi_access"`
	VideoProctoringIncluded bool  `json:"video_proctoring_included"`
	VideoProctoringLimit    int   `json:"video_proctoring_limit"`
}

// GetByTenantID returns the tenant's subscription joined with its plan limits and
// live usage counts in a SINGLE query
func (r *SubscriptionRepo) GetByTenantID(tenantID int) (*SubscriptionRow, error) {
	var s SubscriptionRow
	err := r.DB.QueryRow(`
		SELECT
			ts.id, ts.tenant_id, ts.plan_id, sp.name, ts.status,
			ts.razorpay_subscription_id, ts.current_period_start, ts.current_period_end,
			COALESCE(su.used_bytes, 0) AS storage_used_bytes,
			(SELECT COUNT(*) FROM students s WHERE s.tenant_id = ts.tenant_id AND s.deleted_at IS NULL) AS student_count,
			(SELECT COUNT(*) FROM coaches c WHERE c.tenant_id = ts.tenant_id AND c.deleted_at IS NULL) AS coach_count,
			(SELECT COUNT(*) FROM tests t WHERE t.tenant_id = ts.tenant_id AND date_trunc('month', t.created_at) = date_trunc('month', now() AT TIME ZONE 'UTC')) AS test_count_this_month,
			sp.student_limit, sp.coach_limit, sp.storage_limit_bytes, sp.test_limit,
			sp.sqi_access, sp.video_proctoring_included, sp.video_proctoring_limit
		FROM tenant_subscriptions ts
		JOIN subscription_plans sp ON sp.id = ts.plan_id
		LEFT JOIN storage_usage su ON su.tenant_id = ts.tenant_id
		WHERE ts.tenant_id = $1
	`, tenantID).Scan(
		&s.ID, &s.TenantID, &s.PlanID, &s.PlanName, &s.Status,
		&s.RazorpaySubscriptionID, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&s.StorageUsedBytes, &s.StudentCount, &s.CoachCount, &s.TestCountThisMonth,
		&s.StudentLimit, &s.CoachLimit, &s.StorageLimitBytes, &s.TestLimit,
		&s.SQIAccess, &s.VideoProctoringIncluded, &s.VideoProctoringLimit,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Upsert sets the tenant's plan, inserting a new subscription row or updating the
// existing one (tenant_id is UNIQUE). New rows default to 'active' status.
func (r *SubscriptionRepo) Upsert(tenantID, planID int) error {
	_, err := r.DB.Exec(`
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, status)
		VALUES ($1, $2, 'active')
		ON CONFLICT (tenant_id) DO UPDATE SET plan_id = $2, status = 'active', updated_at = NOW()
	`, tenantID, planID)
	return err
}

func (r *SubscriptionRepo) UpdateStatus(tenantID int, status string) error {
	_, err := r.DB.Exec(`
		UPDATE tenant_subscriptions SET status = $1, updated_at = NOW()
		WHERE tenant_id = $2
	`, status, tenantID)
	return err
}

func (r *SubscriptionRepo) UpdateRazorpayID(tenantID int, razorpayID string) error {
	_, err := r.DB.Exec(`
		UPDATE tenant_subscriptions SET razorpay_subscription_id = $1, updated_at = NOW()
		WHERE tenant_id = $2
	`, razorpayID, tenantID)
	return err
}

// GetUsage returns the live usage counts + limits for a tenant (same row shape as
// GetByTenantID; reuses the single-query implementation to avoid drift).
func (r *SubscriptionRepo) GetUsage(tenantID int) (*SubscriptionRow, error) {
	// falls in the CURRENT calendar month (server timezone) for this tenant.
	// Storage usage is written by later phases (video proctoring); until then
	// storage_used_bytes reads 0 and CheckStorageLimit is effectively unenforced.
	return r.GetByTenantID(tenantID)
}

func (r *SubscriptionRepo) UpdateStorageUsage(tenantID int, bytes int64) error {
	_, err := r.DB.Exec(`
		INSERT INTO storage_usage (tenant_id, used_bytes, last_calculated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (tenant_id) DO UPDATE SET used_bytes = $2, last_calculated_at = NOW()
	`, tenantID, bytes)
	return err
}
