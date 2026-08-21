package repository

import (
	"database/sql"
	"strconv"

	"github.com/lib/pq"
)

type CoachRepo struct {
	DB *sql.DB
}

func NewCoachRepo(db *sql.DB) *CoachRepo {
	return &CoachRepo{DB: db}
}

type CoachSubject struct {
	SubjectID   int    `json:"subject_id"`
	SubjectName string `json:"subject_name"`
}

type CoachRow struct {
	CoachID   int             `json:"coach_id"`
	UserID    int             `json:"user_id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	CreatedAt string          `json:"created_at"`
	DeletedAt *string         `json:"deleted_at"`
	Subjects  []CoachSubject  `json:"subjects"`
}

type CoachDetailRow struct {
	CoachID        int             `json:"coach_id"`
	UserID         int             `json:"user_id"`
	Name           string          `json:"name"`
	Email          string          `json:"email"`
	CreatedAt      string          `json:"created_at"`
	DeletedAt      *string         `json:"deleted_at"`
	DeletedByName  *string         `json:"deleted_by_name"`
	DeletedByEmail *string         `json:"deleted_by_email"`
	DeletedByRole  *string         `json:"deleted_by_role"`
	Subjects       []CoachSubject  `json:"subjects"`
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
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1 AND tenant_id=$2)", coachID, tenantID).Scan(&exists)
	return exists, err
}

func (r *CoachRepo) List(tenantID int, search string, includeDeactivated bool, limit, offset int) ([]CoachRow, int, error) {
	baseQuery := "FROM coaches c JOIN users u ON c.user_id = u.id WHERE c.tenant_id=$1"
	if includeDeactivated {
		baseQuery += " AND c.deleted_at IS NOT NULL"
	} else {
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

	dataQuery := "SELECT c.id, c.user_id, c.name, u.email, c.created_at, c.deleted_at " + baseQuery + " ORDER BY c.id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var coaches []CoachRow
	for rows.Next() {
		var c2 CoachRow
		if err := rows.Scan(&c2.CoachID, &c2.UserID, &c2.Name, &c2.Email, &c2.CreatedAt, &c2.DeletedAt); err != nil {
			return nil, 0, err
		}
		coaches = append(coaches, c2)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(coaches) > 0 {
		ids := make([]int, len(coaches))
		for i, c := range coaches {
			ids[i] = c.CoachID
		}
		subjectMap, err := r.GetCoachSubjectsBatch(ids)
		if err != nil {
			return nil, 0, err
		}
		for i := range coaches {
			coaches[i].Subjects = subjectMap[coaches[i].CoachID]
			if coaches[i].Subjects == nil {
				coaches[i].Subjects = []CoachSubject{}
			}
		}
	}

	return coaches, total, nil
}

type CoachStatMetric struct {
	CoachID      int     `json:"coach_id"`
	StudentCount int     `json:"student_count"`
	AverageSQI   float64 `json:"avg_sqi"`
}

func (r *CoachRepo) GetCoachStats(coachIDs []int, tenantID int) ([]CoachStatMetric, error) {
	rows, err := r.DB.Query(`
		SELECT s.coach_id,
		       COUNT(DISTINCT s.id) AS student_count,
		       COALESCE(AVG(p.student_avg), 0) AS avg_sqi
		FROM students s
		LEFT JOIN (
			SELECT main.student_id, AVG(main.sqi_score) AS student_avg
			FROM (
				SELECT s2.coach_id, ass.student_id, ar.sqi_score,
				       ROW_NUMBER() OVER (PARTITION BY s2.coach_id, ass.student_id ORDER BY a.id DESC) AS rn
				FROM attempt_results ar
				JOIN attempts a ON ar.attempt_id = a.id
				JOIN assignments ass ON a.assignment_id = ass.id
				JOIN students s2 ON ass.student_id = s2.id
				WHERE s2.coach_id = ANY($1) AND s2.tenant_id = $2 AND s2.deleted_at IS NULL
			) main
			WHERE main.rn <= 100
			GROUP BY main.student_id
		) p ON p.student_id = s.id
		WHERE s.coach_id = ANY($1) AND s.tenant_id = $2 AND s.deleted_at IS NULL
		GROUP BY s.coach_id
	`, pq.Array(coachIDs), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []CoachStatMetric
	for rows.Next() {
		var m CoachStatMetric
		if err := rows.Scan(&m.CoachID, &m.StudentCount, &m.AverageSQI); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return metrics, nil
}

func (r *CoachRepo) GetDetail(coachID, tenantID int) (*CoachDetailRow, error) {
	var coach CoachDetailRow
	err := r.DB.QueryRow(`
		SELECT c.id, c.user_id, c.name, u.email, c.created_at,
		       c.deleted_at, du.email, dc.name, du.role
		FROM coaches c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN users du ON c.deleted_by = du.id
		LEFT JOIN coaches dc ON dc.user_id = du.id
		WHERE c.id = $1 AND c.tenant_id = $2
	`, coachID, tenantID).Scan(
		&coach.CoachID, &coach.UserID, &coach.Name, &coach.Email, &coach.CreatedAt,
		&coach.DeletedAt, &coach.DeletedByEmail, &coach.DeletedByName, &coach.DeletedByRole,
	)
	if err != nil {
		return nil, err
	}

	subjects, err := r.GetCoachSubjects(coachID)
	if err != nil {
		return nil, err
	}
	coach.Subjects = subjects
	if coach.Subjects == nil {
		coach.Subjects = []CoachSubject{}
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
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *CoachRepo) CreateCoachSubjects(coachID int, subjectIDs []int) error {
	if len(subjectIDs) == 0 {
		return nil
	}
	stmt, err := r.DB.Prepare("INSERT INTO coach_subjects (coach_id, subject_id) VALUES ($1, $2) ON CONFLICT DO NOTHING")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, sid := range subjectIDs {
		if _, err := stmt.Exec(coachID, sid); err != nil {
			return err
		}
	}
	return nil
}

func (r *CoachRepo) GetCoachSubjects(coachID int) ([]CoachSubject, error) {
	rows, err := r.DB.Query(`
		SELECT cs.subject_id, s.name
		FROM coach_subjects cs
		JOIN subjects s ON cs.subject_id = s.id
		WHERE cs.coach_id = $1 AND s.deleted_at IS NULL
		ORDER BY s.name
	`, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subjects []CoachSubject
	for rows.Next() {
		var cs CoachSubject
		if err := rows.Scan(&cs.SubjectID, &cs.SubjectName); err != nil {
			return nil, err
		}
		subjects = append(subjects, cs)
	}
	return subjects, rows.Err()
}

func (r *CoachRepo) GetCoachSubjectsBatch(coachIDs []int) (map[int][]CoachSubject, error) {
	if len(coachIDs) == 0 {
		return map[int][]CoachSubject{}, nil
	}
	rows, err := r.DB.Query(`
		SELECT cs.coach_id, cs.subject_id, s.name
		FROM coach_subjects cs
		JOIN subjects s ON cs.subject_id = s.id
		WHERE cs.coach_id = ANY($1) AND s.deleted_at IS NULL
		ORDER BY s.name
	`, pq.Array(coachIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[int][]CoachSubject)
	for rows.Next() {
		var coachID int
		var cs CoachSubject
		if err := rows.Scan(&coachID, &cs.SubjectID, &cs.SubjectName); err != nil {
			return nil, err
		}
		result[coachID] = append(result[coachID], cs)
	}
	return result, rows.Err()
}

func (r *CoachRepo) Create(tenantID, userID int, name string) (int, error) {
	var id int
	err := r.DB.QueryRow(
		"INSERT INTO coaches (tenant_id, user_id, name) VALUES ($1, $2, $3) RETURNING id",
		tenantID, userID, name,
	).Scan(&id)
	return id, err
}

func (r *CoachRepo) CreateInTx(tx *sql.Tx, tenantID, userID int, name string) (int, error) {
	var id int
	err := tx.QueryRow(
		"INSERT INTO coaches (tenant_id, user_id, name) VALUES ($1, $2, $3) RETURNING id",
		tenantID, userID, name,
	).Scan(&id)
	return id, err
}

func (r *CoachRepo) CreateCoachSubjectsInTx(tx *sql.Tx, coachID int, subjectIDs []int) error {
	if len(subjectIDs) == 0 {
		return nil
	}
	stmt, err := tx.Prepare("INSERT INTO coach_subjects (coach_id, subject_id) VALUES ($1, $2) ON CONFLICT DO NOTHING")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, sid := range subjectIDs {
		if _, err := stmt.Exec(coachID, sid); err != nil {
			return err
		}
	}
	return nil
}

func (r *CoachRepo) UpdateName(tx *sql.Tx, coachID, tenantID int, name string) error {
	_, err := tx.Exec(
		"UPDATE coaches SET name = $1 WHERE id = $2 AND tenant_id = $3",
		name, coachID, tenantID,
	)
	return err
}

func (r *CoachRepo) DeleteCoachSubjects(tx *sql.Tx, coachID int) error {
	_, err := tx.Exec("DELETE FROM coach_subjects WHERE coach_id = $1", coachID)
	return err
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
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}
