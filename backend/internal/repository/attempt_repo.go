package repository

import (
	"database/sql"
)

type AttemptRepo struct {
	DB *sql.DB
}

func NewAttemptRepo(db *sql.DB) *AttemptRepo {
	return &AttemptRepo{DB: db}
}

type AnswerDetail struct {
	QuestionID      int     `json:"question_id"`
	QuestionText    string  `json:"question_text"`
	OptionA         string  `json:"option_a"`
	OptionB         string  `json:"option_b"`
	OptionC         string  `json:"option_c"`
	OptionD         string  `json:"option_d"`
	CorrectAnswer   string  `json:"correct_answer"`
	SelectedAnswer  string  `json:"selected_answer"`
	IsCorrect       bool    `json:"is_correct"`
	Marks           float64 `json:"marks"`
	TimeSpent       float64 `json:"time_spent"`
	MarkedForReview bool    `json:"marked_for_review"`
	ChangedAnswer   bool    `json:"changed_answer"`
	Seen            bool    `json:"seen"`
	ConceptTag      string  `json:"concept_tag"`
	Difficulty      string  `json:"difficulty"`
}

type AttemptResultRow struct {
	AttemptID int
	TestID    int
	SQI       float64
	Analysis  sql.NullString
}

func (r *AttemptRepo) GetByAssignment(assignmentID int) (int, sql.NullTime, error) {
	var attemptID int
	var submittedAt sql.NullTime
	err := r.DB.QueryRow(
		"SELECT id, submitted_at FROM attempts WHERE assignment_id=$1",
		assignmentID,
	).Scan(&attemptID, &submittedAt)
	return attemptID, submittedAt, err
}

func (r *AttemptRepo) ExistsByAssignment(assignmentID int) (bool, error) {
	var existingAttemptID int
	err := r.DB.QueryRow(
		"SELECT id FROM attempts WHERE assignment_id = $1",
		assignmentID,
	).Scan(&existingAttemptID)
	return err == nil, err
}

func (r *AttemptRepo) GetSQIResult(attemptID int) (sql.NullFloat64, sql.NullString, error) {
	var sqiScore sql.NullFloat64
	var analysisJSON sql.NullString
	err := r.DB.QueryRow(
		"SELECT sqi_score, analysis_json FROM attempt_results WHERE attempt_id=$1",
		attemptID,
	).Scan(&sqiScore, &analysisJSON)
	return sqiScore, analysisJSON, err
}

func (r *AttemptRepo) GetAnswerDetails(attemptID int) ([]AnswerDetail, error) {
	answerRows, err := r.DB.Query(`
		SELECT al.question_id, q.question_text,
		       q.option_a, q.option_b, q.option_c, q.option_d,
		       q.correct_answer, q.marks, q.concept_tag, q.difficulty,
		       COALESCE(al.selected_answer, ''),
		       COALESCE(al.time_spent, 0),
		       COALESCE(al.marked_for_review, false),
		       COALESCE(al.changed_answer, false),
		       COALESCE(al.seen, true)
		FROM answer_logs al
		JOIN questions q ON al.question_id = q.id
		WHERE al.attempt_id = $1
		ORDER BY q.id
	`, attemptID)
	if err != nil {
		return nil, err
	}
	defer answerRows.Close()

	var answers []AnswerDetail
	for answerRows.Next() {
		var a AnswerDetail
		if err := answerRows.Scan(
			&a.QuestionID, &a.QuestionText,
			&a.OptionA, &a.OptionB, &a.OptionC, &a.OptionD,
			&a.CorrectAnswer, &a.Marks, &a.ConceptTag, &a.Difficulty,
			&a.SelectedAnswer, &a.TimeSpent,
			&a.MarkedForReview, &a.ChangedAnswer, &a.Seen,
		); err != nil {
			return nil, err
		}
		a.IsCorrect = a.SelectedAnswer != "" && a.SelectedAnswer == a.CorrectAnswer
		answers = append(answers, a)
	}
	if err := answerRows.Err(); err != nil {
		return nil, err
	}

	return answers, nil
}

func (r *AttemptRepo) GetUncomputedAttempts(studentID int) ([][2]int, error) {
	uncomputedRows, err := r.DB.Query(`
		SELECT a.id, ass.test_id
		FROM attempts a
		JOIN assignments ass ON a.assignment_id = ass.id
		WHERE ass.student_id = $1
		  AND NOT EXISTS (SELECT 1 FROM attempt_results ar WHERE ar.attempt_id = a.id)
		ORDER BY a.id DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer uncomputedRows.Close()

	var attempts [][2]int
	for uncomputedRows.Next() {
		var attemptID, testID int
		if err := uncomputedRows.Scan(&attemptID, &testID); err != nil {
			return nil, err
		}
		attempts = append(attempts, [2]int{attemptID, testID})
	}
	if err := uncomputedRows.Err(); err != nil {
		return nil, err
	}

	return attempts, nil
}

func (r *AttemptRepo) StoreResult(attemptID int, sqiScore, rawScore float64, analysisJSON []byte, version string) error {
	_, err := r.DB.Exec(`
		INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json, version)
		VALUES ($1, $2, $3, $4, $5)
	`, attemptID, sqiScore, rawScore, analysisJSON, version)
	return err
}

func (r *AttemptRepo) GetResults(studentID int, includeAnalysis bool) ([]AttemptResultRow, error) {
	query := `
		SELECT ar.attempt_id, ass.test_id, ar.sqi_score
	`
	if includeAnalysis {
		query += `, ar.analysis_json`
	}
	query += `
		FROM attempt_results ar
		JOIN attempts a ON ar.attempt_id = a.id
		JOIN assignments ass ON a.assignment_id = ass.id
		WHERE ass.student_id = $1
		ORDER BY a.id DESC
	`

	rows, err := r.DB.Query(query, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AttemptResultRow
	for rows.Next() {
		var r2 AttemptResultRow
		if includeAnalysis {
			if err := rows.Scan(&r2.AttemptID, &r2.TestID, &r2.SQI, &r2.Analysis); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&r2.AttemptID, &r2.TestID, &r2.SQI); err != nil {
				return nil, err
			}
		}
		results = append(results, r2)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
