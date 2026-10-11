package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
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
	SQI       sql.NullFloat64
	Analysis  []byte
}

type AnswerInput struct {
	QuestionID        int
	SelectedAnswer    string
	TimeSpent         float64
	Seen              *bool
	MarkedForReview   bool
	Revisited         bool
	ChangedAnswer     bool
	WasInitiallyWrong bool
}

type SubmitAnswersResult struct {
	AttemptID      int
	TotalTimeSpent float64
}

func (r *AttemptRepo) SubmitAnswersTx(
	assignmentID int,
	correctMap map[int]string,
	answers []AnswerInput,
	onCommit func(tx *sql.Tx) error,
) (*SubmitAnswersResult, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var attemptID int
	err = tx.QueryRow(
		"INSERT INTO attempts (assignment_id, submitted_at) VALUES ($1, NOW()) RETURNING id",
		assignmentID,
	).Scan(&attemptID)
	if err != nil {
		return nil, err
	}

	seenQuestionIDs := make(map[int]bool)
	var totalTimeSpent float64

	for _, ans := range answers {
		if _, exists := correctMap[ans.QuestionID]; !exists {
			return nil, fmt.Errorf("invalid question id")
		}
		if seenQuestionIDs[ans.QuestionID] {
			return nil, fmt.Errorf("duplicate question id")
		}
		seenQuestionIDs[ans.QuestionID] = true

		answerSeen := ans.SelectedAnswer != ""
		if ans.Seen != nil {
			answerSeen = *ans.Seen
		}
		if !answerSeen {
			ans.TimeSpent = 0
			ans.MarkedForReview = false
			ans.Revisited = false
			ans.ChangedAnswer = false
			ans.WasInitiallyWrong = false
		}

		totalTimeSpent += ans.TimeSpent

		correctAnswer := correctMap[ans.QuestionID]
		isCorrect := answerSeen && ans.SelectedAnswer != "" && ans.SelectedAnswer == correctAnswer

		_, err = tx.Exec(`
			INSERT INTO answer_logs
			(question_id, attempt_id, selected_answer, is_correct, time_spent,
			 marked_for_review, revisited, changed_answer, was_initially_wrong, seen)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			ans.QuestionID, attemptID, ans.SelectedAnswer, isCorrect, ans.TimeSpent,
			ans.MarkedForReview, ans.Revisited, ans.ChangedAnswer, ans.WasInitiallyWrong, answerSeen,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := onCommit(tx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &SubmitAnswersResult{
		AttemptID:      attemptID,
		TotalTimeSpent: totalTimeSpent,
	}, nil
}

func (r *AttemptRepo) GetCorrectAnswers(testID int) (map[int]string, error) {
	rows, err := r.DB.Query("SELECT q.id, q.correct_answer FROM questions q WHERE q.test_id = $1", testID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	correctMap := make(map[int]string)
	for rows.Next() {
		var questionID int
		var correct string
		if err := rows.Scan(&questionID, &correct); err != nil {
			return nil, err
		}
		correctMap[questionID] = correct
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return correctMap, nil
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

// HasSubmittedAttempt reports whether a terminal (submitted/completed) attempt
// already exists for the assignment. In-progress attempts are allowed so that a
// student can resume a server-timed exam after a refresh.
func (r *AttemptRepo) HasSubmittedAttempt(assignmentID int) (bool, error) {
	var n int
	err := r.DB.QueryRow(
		"SELECT 1 FROM attempts WHERE assignment_id = $1 AND status IN ('submitted', 'completed') LIMIT 1",
		assignmentID,
	).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *AttemptRepo) GetSQIResult(attemptID int) (sql.NullFloat64, []byte, error) {
	var sqiScore sql.NullFloat64
	var analysisJSON []byte
	err := r.DB.QueryRow(
		"SELECT sqi_score, analysis_json FROM attempt_results WHERE attempt_id=$1",
		attemptID,
	).Scan(&sqiScore, &analysisJSON)
	return sqiScore, analysisJSON, err
}

// ReleasableResult is everything the release flow needs to state a score to a
// student: the attempt, its owning assignment, the display names, and the score
// fields lifted out of analysis_json.
type ReleasableResult struct {
	AttemptID   int
	AssignmentID int
	StudentID   int
	StudentName string
	StudentCode string
	StudentEmail sql.NullString
	TestTitle   string
	RawScore    float64
	MaxScore    float64
	Percent     float64
	ReleasedAt  sql.NullTime
	// HasResult distinguishes "the SQI job never ran" from "the student scored
	// zero". Both leave the score fields at zero, so the numeric values cannot
	// answer this.
	HasResult   bool
}

// GetReleasableResult loads the result for an attempt together with the student
// and test it belongs to.
//
// Scoped by assignmentID as well as attemptID so a caller cannot release a
// result by guessing an attempt id that belongs to a different assignment —
// the handler has already authorised the student, and this keeps the attempt
// tied to the assignment that authorisation was granted for.
//
// Percent and max score come from analysis_json.exam_summary, which is where the
// SQI job writes them. A result whose summary lacks them falls back to zero, so
// the email never renders an empty or NaN score.
func (r *AttemptRepo) GetReleasableResult(attemptID, assignmentID int) (*ReleasableResult, error) {
	var out ReleasableResult
	var summary []byte
	var resultID sql.NullInt64
	err := r.DB.QueryRow(`
		SELECT a.id, a.assignment_id, ass.student_id,
		       COALESCE(s.name, ''), COALESCE(s.student_code, ''), s.email,
		       COALESCE(t.title, ''),
		       COALESCE(r.raw_score, 0), COALESCE(r.analysis_json, '{}'::jsonb),
		       r.released_at, r.id
		FROM attempts a
		JOIN assignments ass ON ass.id = a.assignment_id
		JOIN students s ON s.id = ass.student_id
		LEFT JOIN tests t ON t.id = ass.test_id
		LEFT JOIN attempt_results r ON r.attempt_id = a.id
		WHERE a.id = $1 AND a.assignment_id = $2
	`, attemptID, assignmentID).
		Scan(&out.AttemptID, &out.AssignmentID, &out.StudentID,
			&out.StudentName, &out.StudentCode, &out.StudentEmail,
			&out.TestTitle, &out.RawScore, &summary, &out.ReleasedAt, &resultID)
	if err != nil {
		return nil, err
	}
	out.HasResult = resultID.Valid

	var examSummary examSummaryShape
	if len(summary) > 0 {
		var payload struct {
			ExamSummary examSummaryShape `json:"exam_summary"`
		}
		if err := json.Unmarshal(summary, &payload); err == nil {
			examSummary = payload.ExamSummary
		}
	}
	out.MaxScore = examSummary.MaxPossibleScore
	out.Percent = examSummary.ScorePercent
	return &out, nil
}

// examSummaryShape mirrors just the two score fields needed for the release
// email, so a change elsewhere in the diagnostic payload cannot break it.
type examSummaryShape struct {
	MaxPossibleScore float64 `json:"max_possible_score"`
	ScorePercent     float64 `json:"score_percent"`
}

// MarkResultReleased stamps the release audit columns.
//
// Conditional on released_at IS NULL so a concurrent or repeated release
// reports false rather than overwriting the original timestamp and author.
func (r *AttemptRepo) MarkResultReleased(attemptID, releasedBy int) (bool, error) {
	res, err := r.DB.Exec(
		`UPDATE attempt_results SET released_at = NOW(), released_by = $2
		 WHERE attempt_id = $1 AND released_at IS NULL`,
		attemptID, releasedBy,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetSubmittedAttemptID returns the terminal attempt for an assignment.
//
// Picks the latest attempt so a retest (a second assignment for the same test)
// or any future repeat resolves to the most recent sitting.
func (r *AttemptRepo) GetSubmittedAttemptID(assignmentID int) (int, error) {
	var id int
	err := r.DB.QueryRow(
		`SELECT id FROM attempts
		 WHERE assignment_id = $1 AND status IN ('submitted', 'completed')
		 ORDER BY id DESC LIMIT 1`,
		assignmentID,
	).Scan(&id)
	return id, err
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

type AnswerLogForAnalysis struct {
	QuestionID        int
	SelectedAnswer    string
	CorrectAnswer     string
	TimeSpent         float64
	MarkedForReview   bool
	Revisited         bool
	ChangedAnswer     bool
	WasInitiallyWrong bool
	Seen              bool
}

func (r *AttemptRepo) GetAnswerLogsForAnalysis(attemptID int) ([]AnswerLogForAnalysis, error) {
	rows, err := r.DB.Query(`
		SELECT al.question_id,
		       COALESCE(al.selected_answer, ''),
		       COALESCE(q.correct_answer, ''),
		       COALESCE(al.time_spent, 0),
		       COALESCE(al.marked_for_review, false),
		       COALESCE(al.revisited, false),
		       COALESCE(al.changed_answer, false),
		       COALESCE(al.was_initially_wrong, false),
		       COALESCE(al.seen, true)
		FROM answer_logs al
		JOIN questions q ON al.question_id = q.id
		WHERE al.attempt_id = $1
		ORDER BY q.id
	`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AnswerLogForAnalysis
	for rows.Next() {
		var l AnswerLogForAnalysis
		if err := rows.Scan(
			&l.QuestionID, &l.SelectedAnswer, &l.CorrectAnswer,
			&l.TimeSpent, &l.MarkedForReview, &l.Revisited,
			&l.ChangedAnswer, &l.WasInitiallyWrong, &l.Seen,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

// execer is satisfied by both *sql.DB and *sql.Tx so the upsert can run inside
// or outside a transaction.
type execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func (r *AttemptRepo) StoreResultTx(tx execer, attemptID int, sqiScore, rawScore float64, analysisJSON []byte, version string) error {
	_, err := tx.Exec(`
		INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json, version)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (attempt_id) DO UPDATE SET
			sqi_score = EXCLUDED.sqi_score,
			raw_score = EXCLUDED.raw_score,
			analysis_json = EXCLUDED.analysis_json,
			version = EXCLUDED.version,
			updated_at = CURRENT_TIMESTAMP
	`, attemptID, sqiScore, rawScore, analysisJSON, version)
	return err
}

func (r *AttemptRepo) StoreResult(attemptID int, sqiScore, rawScore float64, analysisJSON []byte, version string) error {
	return r.StoreResultTx(r.DB, attemptID, sqiScore, rawScore, analysisJSON, version)
}

func (r *AttemptRepo) GetAverageSQI(studentID int) (float64, error) {
	var avg sql.NullFloat64
	err := r.DB.QueryRow(`
		SELECT AVG(sqi_score) FROM (
			SELECT ar.sqi_score
			FROM attempt_results ar
			JOIN attempts a ON ar.attempt_id = a.id
			JOIN assignments ass ON a.assignment_id = ass.id
			WHERE ass.student_id = $1
			ORDER BY a.id DESC
			LIMIT 100
		) sub
	`, studentID).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

type StudentSQIMetric struct {
	StudentID  int
	AverageSQI float64
	TotalTests int
}

func (r *AttemptRepo) GetStudentSQIMetrics(studentIDs []int, tenantID, coachID int) ([]StudentSQIMetric, error) {
	rows, err := r.DB.Query(`
		SELECT sub.student_id, AVG(sub.sqi_score) AS avg_sqi, COUNT(*) AS total_tests
		FROM (
			SELECT ass.student_id, ar.sqi_score,
				   ROW_NUMBER() OVER (PARTITION BY ass.student_id ORDER BY a.id DESC) AS rn
			FROM attempt_results ar
			JOIN attempts a ON ar.attempt_id = a.id
			JOIN assignments ass ON a.assignment_id = ass.id
			JOIN students s ON ass.student_id = s.id
			WHERE ass.student_id = ANY($1)
			  AND s.tenant_id = $2
			  AND ($3 = 0 OR s.coach_id = $3)
		) sub
		WHERE sub.rn <= 100
		GROUP BY sub.student_id
	`, pq.Array(studentIDs), tenantID, coachID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []StudentSQIMetric
	for rows.Next() {
		var m StudentSQIMetric
		if err := rows.Scan(&m.StudentID, &m.AverageSQI, &m.TotalTests); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return metrics, nil
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
		LIMIT 100
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
