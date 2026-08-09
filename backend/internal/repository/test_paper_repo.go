package repository

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
)

var ErrSubjectDeactivated = errors.New("subject already exists but is deactivated")

type TestPaperRepo struct {
	DB *sql.DB
}

func NewTestPaperRepo(db *sql.DB) *TestPaperRepo {
	return &TestPaperRepo{DB: db}
}

type TestRow struct {
	TestID      int     `json:"test_id"`
	Title       string  `json:"title"`
	SubjectID   int     `json:"subject_id"`
	CoachID     int     `json:"coach_id"`
	Duration    int     `json:"duration"`
	SubjectName string  `json:"subject_name"`
	CoachName   string  `json:"coach_name"`
	ExamDate    *string `json:"exam_date"`
}

type TestDetailRow struct {
	TestID      int     `json:"test_id"`
	Title       string  `json:"title"`
	SubjectID   int     `json:"subject_id"`
	CoachID     int     `json:"coach_id"`
	Duration    int     `json:"duration"`
	CreatedAt   string  `json:"created_at"`
	SubjectName string  `json:"subject_name"`
	CoachName   string  `json:"coach_name"`
	ExamDate    *string `json:"exam_date"`
}

type QuestionRow struct {
	ID            int     `json:"id"`
	QuestionText  string  `json:"question_text"`
	OptionA       string  `json:"option_a"`
	OptionB       string  `json:"option_b"`
	OptionC       string  `json:"option_c"`
	OptionD       string  `json:"option_d"`
	CorrectAnswer string  `json:"correct_answer"`
	Marks         float64 `json:"marks"`
	NegMarks      float64 `json:"neg_marks"`
	Importance    string  `json:"importance"`
	Difficulty    string  `json:"difficulty"`
	Type          string  `json:"type"`
	ExpectedTime  float64 `json:"expected_time"`
	ConceptTag    string  `json:"concept_tag"`
}

type QuestionRequest struct {
	QuestionText  string  `json:"question_text" binding:"required"`
	OptionA       string  `json:"option_a" binding:"required"`
	OptionB       string  `json:"option_b" binding:"required"`
	OptionC       string  `json:"option_c" binding:"required"`
	OptionD       string  `json:"option_d" binding:"required"`
	CorrectAnswer string  `json:"correct_answer" binding:"required"`
	Marks         float64 `json:"marks" binding:"required"`
	NegMarks      float64 `json:"neg_marks" binding:"required"`
	Importance    string  `json:"importance"`
	Difficulty    string  `json:"difficulty"`
	Type          string  `json:"type"`
	ExpectedTime  float64 `json:"expected_time"`
	ConceptTag    string  `json:"concept_tag"`
}

type SubjectRow struct {
	SubjectID int    `json:"subject_id"`
	Name      string `json:"name"`
}

func (r *TestPaperRepo) Create(tenantID int, title string, subjectID, coachID, duration int, examDate *string, subjectName string) (int, error) {
	var id int
	err := r.DB.QueryRow(`
		INSERT INTO tests (tenant_id, title, subject_id, coach_id, duration, exam_date, subject_name)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id
	`, tenantID, title, subjectID, coachID, duration, examDate, subjectName).Scan(&id)
	return id, err
}

func (r *TestPaperRepo) Exists(testID, tenantID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)", testID, tenantID).Scan(&exists)
	return exists, err
}

func (r *TestPaperRepo) ExistsOwnedByCoach(testID, coachID, tenantID int) (bool, error) {
	var exists bool
	err := r.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND coach_id=$2 AND tenant_id=$3 AND deleted_at IS NULL)", testID, coachID, tenantID).Scan(&exists)
	return exists, err
}

func (r *TestPaperRepo) List(tenantID int, coachID *int, search string, limit, offset int) ([]TestRow, int, error) {
	var where string
	var args []interface{}

	if coachID != nil {
		where = "t.tenant_id=$1 AND t.coach_id=$2 AND t.deleted_at IS NULL"
		args = []interface{}{tenantID, *coachID}
	} else {
		where = "t.tenant_id=$1 AND t.deleted_at IS NULL"
		args = []interface{}{tenantID}
	}

	if search != "" {
		where += " AND t.title ILIKE $" + strconv.Itoa(len(args)+1)
		args = append(args, "%"+search+"%")
	}

	countQuery := "SELECT COUNT(*) FROM tests t WHERE " + where
	dataQuery := "SELECT t.id, t.title, t.subject_id, t.coach_id, t.duration, COALESCE(t.subject_name, ''), COALESCE(c.name, ''), t.exam_date FROM tests t LEFT JOIN coaches c ON t.coach_id = c.id WHERE " + where

	var total int
	err := r.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery += " ORDER BY t.id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tests []TestRow
	for rows.Next() {
		var t TestRow
		if err := rows.Scan(&t.TestID, &t.Title, &t.SubjectID, &t.CoachID, &t.Duration, &t.SubjectName, &t.CoachName, &t.ExamDate); err != nil {
			return nil, 0, err
		}
		tests = append(tests, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return tests, total, nil
}

func (r *TestPaperRepo) GetDetail(testID, tenantID int) (*TestDetailRow, error) {
	var t TestDetailRow
	err := r.DB.QueryRow(
		`SELECT t.id, t.title, t.subject_id, t.coach_id, t.duration, t.created_at,
		        COALESCE(t.subject_name, ''), COALESCE(c.name, ''), t.exam_date
		 FROM tests t
		 LEFT JOIN coaches c ON t.coach_id = c.id
		 WHERE t.id=$1 AND t.tenant_id=$2`,
		testID, tenantID,
	).Scan(&t.TestID, &t.Title, &t.SubjectID, &t.CoachID, &t.Duration, &t.CreatedAt, &t.SubjectName, &t.CoachName, &t.ExamDate)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TestPaperRepo) ListQuestions(testID int, limit, offset int) ([]QuestionRow, int, error) {
	var total int
	err := r.DB.QueryRow("SELECT COUNT(*) FROM questions WHERE test_id=$1", testID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.DB.Query(
		`SELECT id, question_text, option_a, option_b, option_c, option_d,
		        correct_answer, marks, neg_marks, importance, difficulty, type, expected_time, concept_tag
		 FROM questions WHERE test_id=$1 ORDER BY id ASC LIMIT $2 OFFSET $3`,
		testID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var questions []QuestionRow
	for rows.Next() {
		var q QuestionRow
		if err := rows.Scan(
			&q.ID, &q.QuestionText, &q.OptionA, &q.OptionB, &q.OptionC, &q.OptionD,
			&q.CorrectAnswer, &q.Marks, &q.NegMarks, &q.Importance, &q.Difficulty, &q.Type, &q.ExpectedTime, &q.ConceptTag,
		); err != nil {
			return nil, 0, err
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return questions, total, nil
}

func (r *TestPaperRepo) CreateQuestions(testID int, questions []QuestionRequest) ([]int, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	questionIDs := make([]int, 0, len(questions))
	for _, req := range questions {
		var id int
		err = tx.QueryRow(`
			INSERT INTO questions
			(test_id, question_text, option_a, option_b, option_c, option_d,
			 correct_answer, marks, neg_marks, importance, difficulty, type, expected_time, concept_tag)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			RETURNING id
		`,
			testID, req.QuestionText, req.OptionA, req.OptionB, req.OptionC, req.OptionD,
			req.CorrectAnswer, req.Marks, req.NegMarks, req.Importance, req.Difficulty,
			req.Type, req.ExpectedTime, req.ConceptTag,
		).Scan(&id)
		if err != nil {
			return nil, err
		}
		questionIDs = append(questionIDs, id)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return questionIDs, nil
}

func ValidateQuestionRequest(req QuestionRequest) string {
	if req.QuestionText == "" || req.OptionA == "" || req.OptionB == "" || req.OptionC == "" || req.OptionD == "" {
		return "question_text and all options are required"
	}
	options := map[string]bool{req.OptionA: true, req.OptionB: true, req.OptionC: true, req.OptionD: true}
	if len(options) != 4 {
		return "duplicate options not allowed"
	}
	if req.CorrectAnswer != "A" && req.CorrectAnswer != "B" && req.CorrectAnswer != "C" && req.CorrectAnswer != "D" {
		return "correct_answer must be A/B/C/D"
	}
	return ""
}

func (r *TestPaperRepo) Update(testID, tenantID int, title string, subjectID, coachID, duration int, examDate *string, subjectName string) (bool, error) {
	result, err := r.DB.Exec(
		`UPDATE tests SET title=$1, subject_id=$2, coach_id=$3, duration=$4, exam_date=$5, subject_name=$6 WHERE id=$7 AND tenant_id=$8`,
		title, subjectID, coachID, duration, examDate, subjectName, testID, tenantID,
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

func (r *TestPaperRepo) Delete(testID, tenantID, deletedBy int) (bool, error) {
	result, err := r.DB.Exec("UPDATE tests SET deleted_at=NOW(), deleted_by=$3 WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL", testID, tenantID, deletedBy)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *TestPaperRepo) UpdateQuestion(questionID, testID int, req QuestionRequest) (bool, error) {
	result, err := r.DB.Exec(
		`UPDATE questions SET
			question_text=$1, option_a=$2, option_b=$3, option_c=$4, option_d=$5,
			correct_answer=$6, marks=$7, neg_marks=$8, importance=$9, difficulty=$10,
			type=$11, expected_time=$12, concept_tag=$13
		 WHERE id=$14 AND test_id=$15`,
		req.QuestionText, req.OptionA, req.OptionB, req.OptionC, req.OptionD,
		req.CorrectAnswer, req.Marks, req.NegMarks, req.Importance, req.Difficulty,
		req.Type, req.ExpectedTime, req.ConceptTag, questionID, testID,
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

func (r *TestPaperRepo) DeleteQuestion(questionID, testID int) (bool, error) {
	result, err := r.DB.Exec(`DELETE FROM questions WHERE id=$1 AND test_id=$2`, questionID, testID)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *TestPaperRepo) CoachTenantID(coachID int) (int, error) {
	var tenantID int
	err := r.DB.QueryRow("SELECT tenant_id FROM coaches WHERE id=$1", coachID).Scan(&tenantID)
	return tenantID, err
}

func (r *TestPaperRepo) ListByCoach(coachID, tenantID, limit, offset int) ([]TestRow, int, error) {
	var total int
	err := r.DB.QueryRow("SELECT COUNT(*) FROM tests WHERE coach_id=$1 AND tenant_id=$2 AND deleted_at IS NULL", coachID, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.DB.Query(`
		SELECT t.id, t.title, t.subject_id, t.duration, COALESCE(t.subject_name, ''), t.exam_date
		FROM tests t
		WHERE t.coach_id = $1 AND t.tenant_id = $2 AND t.deleted_at IS NULL
		ORDER BY t.id DESC LIMIT $3 OFFSET $4
	`, coachID, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tests []TestRow
	for rows.Next() {
		var t TestRow
		if err := rows.Scan(&t.TestID, &t.Title, &t.SubjectID, &t.Duration, &t.SubjectName, &t.ExamDate); err != nil {
			return nil, 0, err
		}
		tests = append(tests, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return tests, total, nil
}

func (r *TestPaperRepo) GetSubjectName(testID int) (string, error) {
	var name string
	err := r.DB.QueryRow(
		`SELECT COALESCE(s.name, '') FROM tests t LEFT JOIN subjects s ON t.subject_id = s.id WHERE t.id = $1`,
		testID,
	).Scan(&name)
	return name, err
}

func (r *TestPaperRepo) GetCoachAndTenant(testID int) (int, int, error) {
	var coachID, tenantID int
	err := r.DB.QueryRow(
		"SELECT coach_id, tenant_id FROM tests WHERE id=$1",
		testID,
	).Scan(&coachID, &tenantID)
	return coachID, tenantID, err
}

func (r *TestPaperRepo) GetDuration(testID int) (int, error) {
	var duration int
	err := r.DB.QueryRow("SELECT COALESCE(duration, 0) FROM tests WHERE id = $1", testID).Scan(&duration)
	return duration, err
}

func (r *TestPaperRepo) CreateSubject(tenantID int, name string) (int, int, error) {
	// Check for a soft-deleted subject with the same name
	var deactivatedID int
	err := r.DB.QueryRow(
		"SELECT id FROM subjects WHERE tenant_id=$1 AND name=$2 AND deleted_at IS NOT NULL",
		tenantID, name,
	).Scan(&deactivatedID)
	if err == nil {
		return 0, deactivatedID, ErrSubjectDeactivated
	}
	if err != sql.ErrNoRows {
		return 0, 0, err
	}

	var id int
	err = r.DB.QueryRow(`INSERT INTO subjects (tenant_id, name) VALUES ($1, $2) RETURNING id`, tenantID, name).Scan(&id)
	if err != nil && strings.Contains(err.Error(), "duplicate") {
		return 0, 0, err
	}
	return id, 0, err
}

func (r *TestPaperRepo) DeleteSubject(subjectID, tenantID, deletedBy int) (bool, error) {
	result, err := r.DB.Exec(
		`UPDATE subjects SET deleted_at=NOW(), deleted_by=$3 WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		subjectID, tenantID, deletedBy,
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

func (r *TestPaperRepo) ReactivateSubject(subjectID, tenantID int) (bool, error) {
	result, err := r.DB.Exec(
		`UPDATE subjects SET deleted_at=NULL, deleted_by=NULL WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NOT NULL`,
		subjectID, tenantID,
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

func (r *TestPaperRepo) ListSubjects(tenantID int, search string, limit, offset int) ([]SubjectRow, int, error) {
	countQuery := "SELECT COUNT(*) FROM subjects WHERE tenant_id=$1 AND deleted_at IS NULL"
	dataQuery := "SELECT id, name FROM subjects WHERE tenant_id=$1 AND deleted_at IS NULL"
	args := []interface{}{tenantID}

	if search != "" {
		countQuery += " AND name ILIKE $2"
		dataQuery += " AND name ILIKE $2"
		args = append(args, "%"+search+"%")
	}

	var total int
	err := r.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery += " ORDER BY id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subjects []SubjectRow
	for rows.Next() {
		var s SubjectRow
		if err := rows.Scan(&s.SubjectID, &s.Name); err != nil {
			return nil, 0, err
		}
		subjects = append(subjects, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return subjects, total, nil
}
