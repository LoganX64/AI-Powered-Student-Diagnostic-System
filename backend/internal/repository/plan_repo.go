package repository

import "database/sql"

type PlanRepo struct {
	DB *sql.DB
}

func NewPlanRepo(db *sql.DB) *PlanRepo {
	return &PlanRepo{DB: db}
}

type PlanRow struct {
	ID                             int    `json:"id"`
	Name                           string `json:"name"`
	Slug                           string `json:"slug"`
	StudentLimit                   int    `json:"student_limit"`
	CoachLimit                     int    `json:"coach_limit"`
	StorageLimitBytes              int64  `json:"storage_limit_bytes"`
	TestLimit                      int    `json:"test_limit"`
	SQIAccess                      bool   `json:"sqi_access"`
	VideoProctoringIncluded        bool   `json:"video_proctoring_included"`
	VideoProctoringLimit           int    `json:"video_proctoring_limit"`
	VideoProctoringPricePerStudent int64  `json:"video_proctoring_price_per_student"`
	PriceMonthly                   int64  `json:"price_monthly"`
	Features                       []byte `json:"features"`
}

func (r *PlanRepo) List() ([]PlanRow, error) {
	rows, err := r.DB.Query(`
		SELECT id, name, slug, student_limit, coach_limit, storage_limit_bytes,
		       test_limit, sqi_access, video_proctoring_included, video_proctoring_limit,
		       video_proctoring_price_per_student, price_monthly, features
		FROM subscription_plans
		ORDER BY price_monthly ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PlanRow
	for rows.Next() {
		var p PlanRow
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.StudentLimit, &p.CoachLimit, &p.StorageLimitBytes,
			&p.TestLimit, &p.SQIAccess, &p.VideoProctoringIncluded, &p.VideoProctoringLimit,
			&p.VideoProctoringPricePerStudent, &p.PriceMonthly, &p.Features,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PlanRepo) GetByID(planID int) (*PlanRow, error) {
	var p PlanRow
	err := r.DB.QueryRow(`
		SELECT id, name, slug, student_limit, coach_limit, storage_limit_bytes,
		       test_limit, sqi_access, video_proctoring_included, video_proctoring_limit,
		       video_proctoring_price_per_student, price_monthly, features
		FROM subscription_plans
		WHERE id = $1
	`, planID).Scan(
		&p.ID, &p.Name, &p.Slug, &p.StudentLimit, &p.CoachLimit, &p.StorageLimitBytes,
		&p.TestLimit, &p.SQIAccess, &p.VideoProctoringIncluded, &p.VideoProctoringLimit,
		&p.VideoProctoringPricePerStudent, &p.PriceMonthly, &p.Features,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlanRepo) GetBySlug(slug string) (*PlanRow, error) {
	var p PlanRow
	err := r.DB.QueryRow(`
		SELECT id, name, slug, student_limit, coach_limit, storage_limit_bytes,
		       test_limit, sqi_access, video_proctoring_included, video_proctoring_limit,
		       video_proctoring_price_per_student, price_monthly, features
		FROM subscription_plans
		WHERE slug = $1
	`, slug).Scan(
		&p.ID, &p.Name, &p.Slug, &p.StudentLimit, &p.CoachLimit, &p.StorageLimitBytes,
		&p.TestLimit, &p.SQIAccess, &p.VideoProctoringIncluded, &p.VideoProctoringLimit,
		&p.VideoProctoringPricePerStudent, &p.PriceMonthly, &p.Features,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlanRepo) Create(p PlanRow) (int, error) {
	if p.Features == nil {
		p.Features = []byte("[]")
	}
	var id int
	err := r.DB.QueryRow(`
		INSERT INTO subscription_plans (
			name, slug, student_limit, coach_limit, storage_limit_bytes, test_limit,
			sqi_access, video_proctoring_included, video_proctoring_limit,
			video_proctoring_price_per_student, price_monthly, features
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id
	`,
		p.Name, p.Slug, p.StudentLimit, p.CoachLimit, p.StorageLimitBytes, p.TestLimit,
		p.SQIAccess, p.VideoProctoringIncluded, p.VideoProctoringLimit,
		p.VideoProctoringPricePerStudent, p.PriceMonthly, p.Features,
	).Scan(&id)
	return id, err
}

func (r *PlanRepo) Update(p PlanRow) error {
	if p.Features == nil {
		p.Features = []byte("[]")
	}
	_, err := r.DB.Exec(`
		UPDATE subscription_plans SET
			name = $1, slug = $2, student_limit = $3, coach_limit = $4,
			storage_limit_bytes = $5, test_limit = $6, sqi_access = $7,
			video_proctoring_included = $8, video_proctoring_limit = $9,
			video_proctoring_price_per_student = $10, price_monthly = $11,
			features = $12, updated_at = NOW()
		WHERE id = $13
	`,
		p.Name, p.Slug, p.StudentLimit, p.CoachLimit, p.StorageLimitBytes, p.TestLimit,
		p.SQIAccess, p.VideoProctoringIncluded, p.VideoProctoringLimit,
		p.VideoProctoringPricePerStudent, p.PriceMonthly, p.Features, p.ID,
	)
	return err
}

func (r *PlanRepo) Delete(planID int) error {
	_, err := r.DB.Exec(`DELETE FROM subscription_plans WHERE id = $1`, planID)
	return err
}
