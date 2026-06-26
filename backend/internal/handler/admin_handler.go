package handlers

import (
	"ai-student-diagnostic/backend/internal/helper"
	"ai-student-diagnostic/backend/internal/services"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *AdminHandler) getCoachIDFromUser(userID int) (int, error) {
	var coachID int
	err := h.DB.QueryRow(
		"SELECT id FROM coaches WHERE user_id = $1 AND deleted_at IS NULL",
		userID,
	).Scan(&coachID)

	return coachID, err
}

type AdminHandler struct {
	DB *sql.DB
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{DB: db}
}

type AttemptResult struct {
	AttemptID int             `json:"attempt_id"`
	TestID    int             `json:"test_id"`
	SQI       float64         `json:"sqi_score"`
	Analysis  json.RawMessage `json:"analysis,omitempty"`
}

type SubjectTestResult struct {
	AttemptID int                       `json:"attempt_id"`
	TestID    int                       `json:"test_id"`
	TestTitle string                    `json:"test_title"`
	SQI       float64                   `json:"sqi_score"`
	Analysis  services.DiagnosticPayloadV2 `json:"analysis"`
}

func (h *AdminHandler) GetStudentSQI(c *gin.Context) {
	idParam := c.Param("id")

	studentID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	if role == "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "super-admin has no access to student scores"})
		return
	}

	var tenantID int
	err = h.DB.QueryRow(
		"SELECT tenant_id FROM users WHERE id=$1 AND tenant_id IS NOT NULL",
		userID,
	).Scan(&tenantID)

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// coach validation
	if role == "coach" {
		coachID, err := h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}

		var exists bool
		err = h.DB.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM assignments
				WHERE student_id = $1 AND coach_id = $2
			)
		`, studentID, coachID).Scan(&exists)

		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "not assigned to this student"})
			return
		}
	}

	// validate student
	var name string
	err = h.DB.QueryRow(
		"SELECT name FROM students WHERE id=$1 AND tenant_id=$2",
		studentID, tenantID,
	).Scan(&name)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	// optional query param: include_analysis=true, compute=true
	includeAnalysis := c.Query("include_analysis") == "true"
	compute := c.Query("compute") == "true"

	if compute {
		// Find attempts without attempt_results and compute SQI for each
		uncomputedRows, err := h.DB.Query(`
			SELECT a.id, ass.test_id
			FROM attempts a
			JOIN assignments ass ON a.assignment_id = ass.id
			WHERE ass.student_id = $1
			  AND NOT EXISTS (SELECT 1 FROM attempt_results ar WHERE ar.attempt_id = a.id)
			ORDER BY a.id DESC
		`, studentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch uncomputed attempts"})
			return
		}
		defer uncomputedRows.Close()

		for uncomputedRows.Next() {
			var attemptID, testID int
			if err := uncomputedRows.Scan(&attemptID, &testID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
				return
			}

			payload, err := calculateAttemptSQIAnalysis(h.DB, attemptID, testID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate sqi"})
				return
			}

			analysisJSON, err := json.Marshal(payload)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode analysis"})
				return
			}

			_, err = h.DB.Exec(`
				INSERT INTO attempt_results (attempt_id, sqi_score, raw_score, analysis_json, version)
				VALUES ($1, $2, $3, $4, $5)
			`, attemptID, payload.OverallSQI, payload.ExamSummary.NetScore, analysisJSON, payload.Version)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store result"})
				return
			}
		}
		if err := uncomputedRows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uncomputed attempts"})
			return
		}
	}

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

	rows, err := h.DB.Query(query, studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch results"})
		return
	}
	defer rows.Close()

	var results []AttemptResult
	var total float64

	for rows.Next() {
		var r AttemptResult

		if includeAnalysis {
			if err := rows.Scan(&r.AttemptID, &r.TestID, &r.SQI, &r.Analysis); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
				return
			}
		} else {
			if err := rows.Scan(&r.AttemptID, &r.TestID, &r.SQI); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
				return
			}
		}

		results = append(results, r)
		total += r.SQI
	}

	var avgSQI float64
	if len(results) > 0 {
		avgSQI = total / float64(len(results))
	}

	c.JSON(http.StatusOK, gin.H{
		"student_id":  studentID,
		"name":        name,
		"attempts":    results,
		"average_sqi": helper.Round2V2(avgSQI),
		"total_tests": len(results),
	})
}

func (h *AdminHandler) GetAssignmentResults(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	assignmentID, err := strconv.Atoi(c.Param("assignmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	var exists bool
	err = h.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND tenant_id=$2)",
		studentID, tenantID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	var assignmentStudentID int
	var testID int
	var status string
	var assignedAt string
	var testTitle string
	err = h.DB.QueryRow(`
		SELECT a.student_id, a.test_id, a.status, a.assigned_at, t.title
		FROM assignments a
		JOIN tests t ON a.test_id = t.id
		WHERE a.id = $1 AND a.student_id = $2
	`, assignmentID, studentID).Scan(&assignmentStudentID, &testID, &status, &assignedAt, &testTitle)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	var studentName string
	var studentCode string
	err = h.DB.QueryRow(
		"SELECT name, student_code FROM students WHERE id=$1 AND tenant_id=$2",
		studentID, tenantID,
	).Scan(&studentName, &studentCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	var attemptID int
	var submittedAt sql.NullTime
	err = h.DB.QueryRow(
		"SELECT id, submitted_at FROM attempts WHERE assignment_id=$1",
		assignmentID,
	).Scan(&attemptID, &submittedAt)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"student":    gin.H{"id": studentID, "name": studentName, "student_code": studentCode},
			"testing":    gin.H{"id": testID, "title": testTitle},
			"assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
			"attempt":    nil,
			"sqi_score":  nil,
			"answers":    []interface{}{},
		})
		return
	}

	var sqiScore sql.NullFloat64
	var analysisJSON sql.NullString
	err = h.DB.QueryRow(
		"SELECT sqi_score, analysis_json FROM attempt_results WHERE attempt_id=$1",
		attemptID,
	).Scan(&sqiScore, &analysisJSON)

	answerRows, err := h.DB.Query(`
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch answers"})
		return
	}
	defer answerRows.Close()

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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		a.IsCorrect = a.SelectedAnswer != "" && a.SelectedAnswer == a.CorrectAnswer
		answers = append(answers, a)
	}
	if err := answerRows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse analysis JSON if present
	var analysis interface{}
	if analysisJSON.Valid && analysisJSON.String != "" {
		var payload services.DiagnosticPayloadV2
		if err := json.Unmarshal([]byte(analysisJSON.String), &payload); err == nil {
			analysis = payload
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"student":    gin.H{"id": studentID, "name": studentName, "student_code": studentCode},
		"test":       gin.H{"id": testID, "title": testTitle},
		"assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
		"attempt":    gin.H{"id": attemptID, "submitted_at": submittedAt.Time},
		"sqi_score":  sqiScore.Float64,
		"analysis":   analysis,
		"answers":    answers,
	})
}

func (h *AdminHandler) GetStudentSubjectResults(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	subjectID, err := strconv.Atoi(c.Param("subject_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject id"})
		return
	}

	var testID int
	if testIDParam := c.Query("test_id"); testIDParam != "" {
		testID, err = strconv.Atoi(testIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id"})
			return
		}
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	if role == "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "super-admin has no access to student scores"})
		return
	}

	var tenantID int
	err = h.DB.QueryRow(
		"SELECT tenant_id FROM users WHERE id=$1 AND tenant_id IS NOT NULL",
		userID,
	).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var studentName string
	err = h.DB.QueryRow(
		"SELECT name FROM students WHERE id=$1 AND tenant_id=$2",
		studentID, tenantID,
	).Scan(&studentName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	var subjectName string
	err = h.DB.QueryRow(
		"SELECT name FROM subjects WHERE id=$1 AND tenant_id=$2",
		subjectID, tenantID,
	).Scan(&subjectName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subject not found"})
		return
	}

	if role == "coach" {
		coachID, err := h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}

		var exists bool
		err = h.DB.QueryRow(`
			SELECT EXISTS(
				SELECT 1
				FROM students
				WHERE id = $1 AND tenant_id = $2 AND coach_id = $3
			)
		`, studentID, tenantID, coachID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "not assigned to this student"})
			return
		}
	}

	query := `
		SELECT a.id, t.id, COALESCE(t.title, '')
		FROM attempts a
		JOIN assignments ass ON a.assignment_id = ass.id
		JOIN tests t ON ass.test_id = t.id
		WHERE ass.student_id = $1
		  AND t.subject_id = $2
		  AND t.tenant_id = $3
	`
	args := []any{studentID, subjectID, tenantID}
	if testID > 0 {
		query += " AND t.id = $4"
		args = append(args, testID)
	}
	query += " ORDER BY a.id DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch attempts"})
		return
	}
	defer rows.Close()

	var results []SubjectTestResult
	var totalSQI float64

	for rows.Next() {
		var result SubjectTestResult
		if err := rows.Scan(&result.AttemptID, &result.TestID, &result.TestTitle); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}

			analysis, err := calculateAttemptSQIAnalysis(h.DB, result.AttemptID, result.TestID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate sqi"})
			return
		}

		result.SQI = analysis.OverallSQI
		result.Analysis = analysis
		results = append(results, result)
		totalSQI += result.SQI
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read attempts"})
		return
	}

	var averageSQI float64
	if len(results) > 0 {
		averageSQI = totalSQI / float64(len(results))
	}

	response := gin.H{
		"student_id":     studentID,
		"student_name":   studentName,
		"subject_id":     subjectID,
		"subject_name":   subjectName,
		"results":        results,
		"average_sqi":    helper.Round2V2(averageSQI),
		"total_attempts": len(results),
		"calculation":    "sqi_engine",
	}
	if testID > 0 {
		response["filter_test_id"] = testID
	}

	c.JSON(http.StatusOK, response)
}

func calculateAttemptSQIAnalysis(db *sql.DB, attemptID int, testID int) (services.DiagnosticPayloadV2, error) {
	questionRows, err := db.Query(`
		SELECT q.id, q.marks, q.neg_marks,
		       CASE q.importance WHEN 'A' THEN 'high' WHEN 'B' THEN 'medium' WHEN 'C' THEN 'low' END,
		       q.difficulty,
		       CASE q.type WHEN 'Theory' THEN 'mcq' WHEN 'Practical' THEN 'integer' END,
		       q.expected_time, q.concept_tag,
		       COALESCE(s.name, 'Uncategorized')
		FROM questions q
		JOIN tests t ON q.test_id = t.id
		LEFT JOIN subjects s ON t.subject_id = s.id
		WHERE q.test_id = $1
		ORDER BY q.id
	`, testID)
	if err != nil {
		return services.DiagnosticPayloadV2{}, err
	}
	defer questionRows.Close()

	var questions []services.QuestionMetaV2
	for questionRows.Next() {
		var q services.QuestionMetaV2
		if err := questionRows.Scan(
			&q.QuestionID,
			&q.Marks,
			&q.NegMarks,
			&q.Importance,
			&q.Difficulty,
			&q.Type,
			&q.ExpectedTime,
			&q.ConceptTag,
			&q.Subject,
		); err != nil {
			return services.DiagnosticPayloadV2{}, err
		}
		questions = append(questions, q)
	}
	if err := questionRows.Err(); err != nil {
		return services.DiagnosticPayloadV2{}, err
	}

	answerRows, err := db.Query(`
		SELECT
			al.question_id,
			COALESCE(al.selected_answer, ''),
			q.correct_answer,
			COALESCE(al.time_spent, 0),
			COALESCE(al.marked_for_review, false),
			COALESCE(al.revisited, false),
			COALESCE(al.changed_answer, false),
			COALESCE(al.was_initially_wrong, false),
			COALESCE(al.seen, true)
		FROM answer_logs al
		JOIN questions q ON al.question_id = q.id
		WHERE al.attempt_id = $1
	`, attemptID)
	if err != nil {
		return services.DiagnosticPayloadV2{}, err
	}
	defer answerRows.Close()

	var answers []services.AnswerLogV2
	for answerRows.Next() {
		var a services.AnswerLogV2
		if err := answerRows.Scan(
			&a.QuestionID,
			&a.SelectedAnswer,
			&a.CorrectAnswer,
			&a.TimeSpent,
			&a.MarkedForReview,
			&a.Revisited,
			&a.ChangedAnswer,
			&a.WasInitiallyWrong,
			&a.Seen,
		); err != nil {
			return services.DiagnosticPayloadV2{}, err
		}
		answers = append(answers, a)
	}
	if err := answerRows.Err(); err != nil {
		return services.DiagnosticPayloadV2{}, err
	}

	// Get test duration and check for negative marking
	var duration int
	var hasNegMarking bool
	err = db.QueryRow(`
		SELECT COALESCE(t.duration, 0),
		       EXISTS(SELECT 1 FROM questions WHERE test_id = $1 AND neg_marks > 0)
		FROM tests t WHERE t.id = $1
	`, testID).Scan(&duration, &hasNegMarking)
	if err != nil {
		return services.DiagnosticPayloadV2{}, err
	}

	cfg := services.ExamConfigV2{
		ExamType:           "competitive",
		HasNegativeMarking: hasNegMarking,
		TotalDuration:      float64(duration),
	}

	return services.Analyze(questions, answers, cfg), nil
}

// add student
type CreateStudentRequest struct {
	Name        string `json:"name" binding:"required"`
	StudentCode string `json:"student_code" binding:"required"`
	CoachID     int    `json:"coach_id"`
}

func (h *AdminHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	var coachID int
	var tenantID int
	err := h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err = h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
	} else if role == "admin" {

		if req.CoachID == 0 {
			err = h.DB.QueryRow("SELECT id FROM coaches WHERE user_id = $1 AND deleted_at IS NULL", userID).Scan(&coachID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "coach_id is required, or you must create a coach profile for yourself first"})
				return
			}
		} else {

			var exists bool
			err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)", req.CoachID, tenantID).Scan(&exists)
			if err != nil || !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach_id for your organization"})
				return
			}
			coachID = req.CoachID
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	var id int
	err = h.DB.QueryRow(`
		INSERT INTO students (tenant_id, name, student_code, coach_id)
		VALUES ($1,$2,$3,$4)
		RETURNING id
	`, tenantID, req.Name, req.StudentCode, coachID).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"student_id": id})
}

// add subject

type CreateSubjectRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *AdminHandler) CreateSubject(c *gin.Context) {
	var req CreateSubjectRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	if role != "admin" && role != "coach" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin or coach can create subjects"})
		return
	}

	var tenantID int
	err := h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	var id int
	err = h.DB.QueryRow(`
		INSERT INTO subjects (tenant_id, name)
		VALUES ($1, $2) RETURNING id
	`, tenantID, req.Name).Scan(&id)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "subject already exists in your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subject_id": id})
}

// create test
type CreateTestRequest struct {
	Title     string  `json:"title" binding:"required"`
	SubjectID int     `json:"subject_id" binding:"required"`
	CoachID   int     `json:"coach_id" binding:"required"`
	Duration  int     `json:"duration" binding:"required"`
	ExamDate  *string `json:"exam_date"`
}

func (h *AdminHandler) CreateTest(c *gin.Context) {
	var req CreateTestRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	var coachID int
	var tenantID int
	err := h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		var err error
		coachID, err = h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
	} else if role == "admin" {

		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)", req.CoachID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach_id for your organization"})
			return
		}
		coachID = req.CoachID
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	var id int
	err = h.DB.QueryRow(`
		INSERT INTO tests (tenant_id, title, subject_id, coach_id, duration, exam_date)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id
	`, tenantID, req.Title, req.SubjectID, coachID, req.Duration, req.ExamDate).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"test_id": id})
}

// Add question
type CreateQuestionRequest struct {
	QuestionText string `json:"question_text" binding:"required"`

	OptionA string `json:"option_a" binding:"required"`
	OptionB string `json:"option_b" binding:"required"`
	OptionC string `json:"option_c" binding:"required"`
	OptionD string `json:"option_d" binding:"required"`

	CorrectAnswer string  `json:"correct_answer" binding:"required"`
	Marks         float64 `json:"marks" binding:"required"`
	NegMarks      float64 `json:"neg_marks" binding:"required"`

	Importance   string  `json:"importance"`
	Difficulty   string  `json:"difficulty"`
	Type         string  `json:"type"`
	ExpectedTime float64 `json:"expected_time"`
	ConceptTag   string  `json:"concept_tag"`
}

func parseQuestionRequests(c *gin.Context) ([]CreateQuestionRequest, error) {
	body, err := c.GetRawData()
	if err != nil {
		return nil, err
	}

	var batch []CreateQuestionRequest
	if err := json.Unmarshal(body, &batch); err == nil {
		return batch, nil
	}

	var single CreateQuestionRequest
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}

	return []CreateQuestionRequest{single}, nil
}

func validateQuestionRequest(req CreateQuestionRequest) string {
	if req.QuestionText == "" ||
		req.OptionA == "" ||
		req.OptionB == "" ||
		req.OptionC == "" ||
		req.OptionD == "" {
		return "question_text and all options are required"
	}

	options := map[string]bool{
		req.OptionA: true,
		req.OptionB: true,
		req.OptionC: true,
		req.OptionD: true,
	}
	if len(options) != 4 {
		return "duplicate options not allowed"
	}

	if req.CorrectAnswer != "A" &&
		req.CorrectAnswer != "B" &&
		req.CorrectAnswer != "C" &&
		req.CorrectAnswer != "D" {
		return "correct_answer must be A/B/C/D"
	}

	return ""
}

func createQuestionsForTest(database *sql.DB, testID int, questions []CreateQuestionRequest) ([]int, error) {
	tx, err := database.Begin()
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
			testID,
			req.QuestionText,
			req.OptionA,
			req.OptionB,
			req.OptionC,
			req.OptionD,
			req.CorrectAnswer,
			req.Marks,
			req.NegMarks,
			req.Importance,
			req.Difficulty,
			req.Type,
			req.ExpectedTime,
			req.ConceptTag,
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

func (h *AdminHandler) CreateQuestion(c *gin.Context) {
	testIDParam := c.Param("id")
	testID, err := strconv.Atoi(testIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id"})
		return
	}

	questions, err := parseQuestionRequests(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid payload",
		})
		return
	}
	if len(questions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one question is required"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	var tenantID int
	err = h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	var coachID int
	if role == "coach" {
		var err error
		coachID, err = h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		// Verify test belongs to coach AND same tenant
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND coach_id=$2 AND tenant_id=$3)", testID, coachID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {

		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND tenant_id=$2)", testID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	for i, question := range questions {
		if validationErr := validateQuestionRequest(question); validationErr != "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":    validationErr,
				"position": i,
			})
			return
		}
	}

	questionIDs, err := createQuestionsForTest(h.DB, testID, questions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "failed to create questions",
		})
		return
	}

	response := gin.H{
		"question_ids": questionIDs,
		"count":        len(questionIDs),
		"message":      "questions created successfully",
	}
	if len(questionIDs) == 1 {
		response["question_id"] = questionIDs[0]
	}

	c.JSON(http.StatusOK, response)
}

// create assignment
type CreateAssignmentRequest struct {
	StudentID int `json:"student_id" binding:"required"`
	TestID    int `json:"test_id" binding:"required"`
	CoachID   int `json:"coach_id" binding:"required"`
}

func (h *AdminHandler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	var coachID int
	var err error

	if role == "coach" {
		coachID, err = h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
	} else if role == "admin" {
		coachID = req.CoachID
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin or coach can assign tests"})
		return
	}

	var tenantID int
	err = h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	// validate student belongs to coach and same tenant and is not deactivated
	var studentCoachID int
	var studentTenantID int
	err = h.DB.QueryRow(
		"SELECT coach_id, tenant_id FROM students WHERE id=$1 AND deleted_at IS NULL",
		req.StudentID,
	).Scan(&studentCoachID, &studentTenantID)

	if err != nil || studentCoachID != coachID || studentTenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "student not found, deactivated, or not in your organization"})
		return
	}

	// validate test belongs to coach and same tenant
	var testCoachID int
	var testTenantID int
	err = h.DB.QueryRow(
		"SELECT coach_id, tenant_id FROM tests WHERE id=$1",
		req.TestID,
	).Scan(&testCoachID, &testTenantID)

	if err != nil || testCoachID != coachID || testTenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not in your organization"})
		return
	}

	var id int
	err = h.DB.QueryRow(`
		INSERT INTO assignments (student_id, test_id, coach_id)
		VALUES ($1,$2,$3)
		RETURNING id
	`, req.StudentID, req.TestID, coachID).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"assignment_id": id})
}

// ─── List endpoints ────────────────────────────────────────────────────────────

func (h *AdminHandler) getTenantID(userID int) (int, error) {
	var tenantID int
	err := h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	return tenantID, err
}

func parsePagination(c *gin.Context) (int, int) {
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 10000 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

func (h *AdminHandler) ListTests(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := parsePagination(c)
	search := c.Query("search")

	baseQuery := "FROM tests t LEFT JOIN subjects s ON t.subject_id = s.id LEFT JOIN coaches c ON t.coach_id = c.id WHERE t.tenant_id=$1"
	countQuery := "SELECT COUNT(*) " + baseQuery
	dataQuery := "SELECT t.id, t.title, t.subject_id, t.coach_id, t.duration, COALESCE(s.name, ''), COALESCE(c.name, ''), t.exam_date " + baseQuery

	args := []interface{}{tenantID}

	if search != "" {
		baseQuery += " AND t.title ILIKE $" + strconv.Itoa(len(args)+1)
		countQuery = "SELECT COUNT(*) " + baseQuery
		dataQuery = "SELECT t.id, t.title, t.subject_id, t.coach_id, t.duration, COALESCE(s.name, ''), COALESCE(c.name, ''), t.exam_date " + baseQuery
		args = append(args, "%"+search+"%")
	}

	var total int
	err = h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dataQuery += " ORDER BY id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := h.DB.Query(dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

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

	var tests []TestRow
	for rows.Next() {
		var t TestRow
		if err := rows.Scan(&t.TestID, &t.Title, &t.SubjectID, &t.CoachID, &t.Duration, &t.SubjectName, &t.CoachName, &t.ExamDate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		tests = append(tests, t)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tests})
}

func (h *AdminHandler) GetTest(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	testID := c.Param("id")

	var test struct {
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

	err = h.DB.QueryRow(
		`SELECT t.id, t.title, t.subject_id, t.coach_id, t.duration, t.created_at,
		        COALESCE(s.name, ''), COALESCE(c.name, ''), t.exam_date
		 FROM tests t
		 LEFT JOIN subjects s ON t.subject_id = s.id
		 LEFT JOIN coaches c ON t.coach_id = c.id
		 WHERE t.id=$1 AND t.tenant_id=$2`,
		testID, tenantID,
	).Scan(&test.TestID, &test.Title, &test.SubjectID, &test.CoachID, &test.Duration, &test.CreatedAt, &test.SubjectName, &test.CoachName, &test.ExamDate)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	c.JSON(http.StatusOK, test)
}

func (h *AdminHandler) GetTestQuestions(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	testID := c.Param("id")

	var exists bool
	err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND tenant_id=$2)", testID, tenantID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	limit, offset := parsePagination(c)

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM questions WHERE test_id=$1", testID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := h.DB.Query(
		`SELECT id, question_text, option_a, option_b, option_c, option_d,
		        correct_answer, marks, neg_marks, importance, difficulty, type, expected_time, concept_tag
		 FROM questions WHERE test_id=$1 ORDER BY id ASC LIMIT $2 OFFSET $3`,
		testID, limit, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

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

	var questions []QuestionRow
	for rows.Next() {
		var q QuestionRow
		if err := rows.Scan(
			&q.ID, &q.QuestionText, &q.OptionA, &q.OptionB, &q.OptionC, &q.OptionD,
			&q.CorrectAnswer, &q.Marks, &q.NegMarks, &q.Importance, &q.Difficulty, &q.Type, &q.ExpectedTime, &q.ConceptTag,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": questions})
}

func (h *AdminHandler) ListStudents(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := parsePagination(c)

	includeDeactivated := c.Query("include_deactivated") == "true"

	var total int
	if includeDeactivated {
		err = h.DB.QueryRow("SELECT COUNT(*) FROM students WHERE tenant_id=$1", tenantID).Scan(&total)
	} else {
		err = h.DB.QueryRow("SELECT COUNT(*) FROM students WHERE tenant_id=$1 AND deleted_at IS NULL", tenantID).Scan(&total)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	query := "SELECT id, name, student_code, coach_id, deleted_at FROM students WHERE tenant_id=$1"
	if !includeDeactivated {
		query += " AND deleted_at IS NULL"
	}
	query += " ORDER BY id DESC LIMIT $2 OFFSET $3"

	rows, err := h.DB.Query(query, tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type StudentRow struct {
		StudentID   int     `json:"student_id"`
		Name        string  `json:"name"`
		StudentCode string  `json:"student_code"`
		CoachID     int     `json:"coach_id"`
		DeletedAt   *string `json:"deleted_at"`
	}

	var students []StudentRow
	for rows.Next() {
		var s StudentRow
		if err := rows.Scan(&s.StudentID, &s.Name, &s.StudentCode, &s.CoachID, &s.DeletedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": students})
}

func (h *AdminHandler) GetStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	type StudentDetailRow struct {
		StudentID      int     `json:"student_id"`
		Name           string  `json:"name"`
		StudentCode    string  `json:"student_code"`
		CoachID        int     `json:"coach_id"`
		CoachName      string  `json:"coach_name"`
		CreatedAt      string  `json:"created_at"`
		DeletedAt      *string `json:"deleted_at"`
		DeletedByName  *string `json:"deleted_by_name"`
		DeletedByEmail *string `json:"deleted_by_email"`
		DeletedByRole  *string `json:"deleted_by_role"`
	}

	var s StudentDetailRow
	err = h.DB.QueryRow(`
		SELECT st.id, st.name, st.student_code, st.coach_id, COALESCE(c.name, ''),
		       st.created_at, st.deleted_at, u.email, dco.name, u.role
		FROM students st
		LEFT JOIN coaches c ON st.coach_id = c.id
		LEFT JOIN users u ON st.deleted_by = u.id
		LEFT JOIN coaches dco ON dco.user_id = u.id
		WHERE st.id = $1 AND st.tenant_id = $2
	`, studentID, tenantID).Scan(
		&s.StudentID, &s.Name, &s.StudentCode, &s.CoachID, &s.CoachName,
		&s.CreatedAt, &s.DeletedAt, &s.DeletedByEmail, &s.DeletedByName, &s.DeletedByRole,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	c.JSON(http.StatusOK, s)
}

func (h *AdminHandler) ListStudentAssignments(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	var exists bool
	err = h.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND tenant_id=$2)",
		studentID, tenantID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	limit, offset := parsePagination(c)

	var total int
	err = h.DB.QueryRow(
		"SELECT COUNT(*) FROM assignments WHERE student_id=$1", studentID,
	).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := h.DB.Query(`
		SELECT a.id, a.test_id, t.title, a.status, a.assigned_at,
		       (a.status = 'submitted') AS submitted
		FROM assignments a
		JOIN tests t ON a.test_id = t.id
		WHERE a.student_id = $1
		ORDER BY a.id DESC
		LIMIT $2 OFFSET $3
	`, studentID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type AssignmentRow struct {
		ID         int    `json:"id"`
		TestID     int    `json:"test_id"`
		TestTitle  string `json:"test_title"`
		Status     string `json:"status"`
		AssignedAt string `json:"assigned_at"`
		Submitted  bool   `json:"submitted"`
	}

	var assignments []AssignmentRow
	for rows.Next() {
		var a AssignmentRow
		if err := rows.Scan(&a.ID, &a.TestID, &a.TestTitle, &a.Status, &a.AssignedAt, &a.Submitted); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": assignments})
}

func (h *AdminHandler) DeleteStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	result, err := h.DB.Exec(
		`UPDATE students SET deleted_at = NOW(), deleted_by = $1
		 WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`,
		userID, studentID, tenantID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found or already deactivated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "student account deactivated"})
}

func (h *AdminHandler) ReactivateStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	result, err := h.DB.Exec(
		`UPDATE students SET deleted_at = NULL, deleted_by = NULL
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NOT NULL`,
		studentID, tenantID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found or already active"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "student account reactivated"})
}

func (h *AdminHandler) ListCoaches(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := parsePagination(c)
	search := c.Query("search")
	includeDeactivated := c.Query("include_deactivated") == "true"

	baseQuery := "FROM coaches c JOIN users u ON c.user_id = u.id WHERE c.tenant_id=$1"
	if !includeDeactivated {
		baseQuery += " AND c.deleted_at IS NULL"
	}
	countQuery := "SELECT COUNT(*) " + baseQuery
	dataQuery := "SELECT c.id, c.user_id, c.name, u.email " + baseQuery

	args := []interface{}{tenantID}

	if search != "" {
		baseQuery += " AND c.name ILIKE $" + strconv.Itoa(len(args)+1)
		countQuery = "SELECT COUNT(*) " + baseQuery
		dataQuery = "SELECT c.id, c.user_id, c.name, u.email " + baseQuery
		args = append(args, "%"+search+"%")
	}

	var total int
	err = h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dataQuery += " ORDER BY c.id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := h.DB.Query(dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type CoachRow struct {
		CoachID int    `json:"coach_id"`
		UserID  int    `json:"user_id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
	}

	var coaches []CoachRow
	for rows.Next() {
		var c2 CoachRow
		if err := rows.Scan(&c2.CoachID, &c2.UserID, &c2.Name, &c2.Email); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		coaches = append(coaches, c2)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": coaches})
}

func (h *AdminHandler) GetCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	type CoachDetailRow struct {
		CoachID   int    `json:"coach_id"`
		UserID    int    `json:"user_id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		CreatedAt string `json:"created_at"`
	}

	var coach CoachDetailRow
	err = h.DB.QueryRow(`
		SELECT c.id, c.user_id, c.name, u.email, c.created_at
		FROM coaches c
		JOIN users u ON c.user_id = u.id
		WHERE c.id = $1 AND c.tenant_id = $2 AND c.deleted_at IS NULL
	`, coachID, tenantID).Scan(&coach.CoachID, &coach.UserID, &coach.Name, &coach.Email, &coach.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	c.JSON(http.StatusOK, coach)
}

func (h *AdminHandler) DeleteCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	result, err := h.DB.Exec(`
		UPDATE coaches
		SET deleted_at = NOW(), deleted_by = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`, userID, coachID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found or already deactivated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach account deactivated"})
}

func (h *AdminHandler) ReactivateCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	result, err := h.DB.Exec(
		`UPDATE coaches SET deleted_at = NULL, deleted_by = NULL
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NOT NULL`,
		coachID, tenantID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found or already active"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach account reactivated"})
}

func (h *AdminHandler) ListCoachTests(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	var exists bool
	err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)", coachID, tenantID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := parsePagination(c)

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM tests WHERE coach_id=$1 AND tenant_id=$2", coachID, tenantID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := h.DB.Query(`
		SELECT t.id, t.title, t.subject_id, t.duration, COALESCE(s.name, ''), t.exam_date, t.created_at
		FROM tests t
		LEFT JOIN subjects s ON t.subject_id = s.id
		WHERE t.coach_id = $1 AND t.tenant_id = $2
		ORDER BY t.id DESC
		LIMIT $3 OFFSET $4
	`, coachID, tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type TestRow struct {
		TestID      int     `json:"test_id"`
		Title       string  `json:"title"`
		SubjectID   int     `json:"subject_id"`
		Duration    int     `json:"duration"`
		SubjectName string  `json:"subject_name"`
		ExamDate    *string `json:"exam_date"`
		CreatedAt   string  `json:"created_at"`
	}

	var tests []TestRow
	for rows.Next() {
		var t TestRow
		if err := rows.Scan(&t.TestID, &t.Title, &t.SubjectID, &t.Duration, &t.SubjectName, &t.ExamDate, &t.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		tests = append(tests, t)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tests})
}

func (h *AdminHandler) ListCoachStudents(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	var exists bool
	err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL)", coachID, tenantID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := parsePagination(c)

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM students WHERE coach_id=$1 AND tenant_id=$2 AND deleted_at IS NULL", coachID, tenantID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := h.DB.Query(`
		SELECT id, name, student_code, created_at
		FROM students
		WHERE coach_id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		ORDER BY id DESC
		LIMIT $3 OFFSET $4
	`, coachID, tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type StudentRow struct {
		StudentID   int    `json:"student_id"`
		Name        string `json:"name"`
		StudentCode string `json:"student_code"`
		CreatedAt   string `json:"created_at"`
	}

	var students []StudentRow
	for rows.Next() {
		var s StudentRow
		if err := rows.Scan(&s.StudentID, &s.Name, &s.StudentCode, &s.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": students})
}

func (h *AdminHandler) ListSubjects(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := parsePagination(c)
	search := c.Query("search")

	baseQuery := "FROM subjects WHERE tenant_id=$1"
	countQuery := "SELECT COUNT(*) " + baseQuery
	dataQuery := "SELECT id, name " + baseQuery

	args := []interface{}{tenantID}

	if search != "" {
		baseQuery += " AND name ILIKE $" + strconv.Itoa(len(args)+1)
		countQuery = "SELECT COUNT(*) " + baseQuery
		dataQuery = "SELECT id, name " + baseQuery
		args = append(args, "%"+search+"%")
	}

	var total int
	err = h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dataQuery += " ORDER BY id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := h.DB.Query(dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type SubjectRow struct {
		SubjectID int    `json:"subject_id"`
		Name      string `json:"name"`
	}

	var subjects []SubjectRow
	for rows.Next() {
		var s SubjectRow
		if err := rows.Scan(&s.SubjectID, &s.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		subjects = append(subjects, s)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": subjects})
}

func (h *AdminHandler) ListAssignments(c *gin.Context) {
	userID := c.GetInt("user_id")
	tenantID, err := h.getTenantID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := parsePagination(c)
	testID := c.Query("test_id")

	baseQuery := "FROM assignments a JOIN students s ON a.student_id = s.id JOIN tests t ON a.test_id = t.id WHERE s.tenant_id=$1 AND s.deleted_at IS NULL"
	countQuery := "SELECT COUNT(*) " + baseQuery
	dataQuery := `SELECT a.id, a.student_id, s.name, s.student_code, a.test_id, t.title, a.coach_id, a.status, a.assigned_at ` + baseQuery

	args := []interface{}{tenantID}

	if testID != "" {
		baseQuery += " AND a.test_id=$" + strconv.Itoa(len(args)+1)
		countQuery = "SELECT COUNT(*) " + baseQuery
		dataQuery = `SELECT a.id, a.student_id, s.name, s.student_code, a.test_id, t.title, a.coach_id, a.status, a.assigned_at ` + baseQuery
		args = append(args, testID)
	}

	var total int
	err = h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dataQuery += " ORDER BY a.id DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := h.DB.Query(dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type AssignmentRow struct {
		ID          int    `json:"id"`
		StudentID   int    `json:"student_id"`
		StudentName string `json:"student_name"`
		StudentCode string `json:"student_code"`
		TestID      int    `json:"test_id"`
		TestTitle   string `json:"test_title"`
		CoachID     int    `json:"coach_id"`
		Status      string `json:"status"`
		AssignedAt  string `json:"assigned_at"`
	}

	var assignments []AssignmentRow
	for rows.Next() {
		var a AssignmentRow
		if err := rows.Scan(&a.ID, &a.StudentID, &a.StudentName, &a.StudentCode, &a.TestID, &a.TestTitle, &a.CoachID, &a.Status, &a.AssignedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": assignments})
}

func (h *AdminHandler) UpdateTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}

	var req CreateTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	var tenantID int
	err = h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND coach_id=$2 AND tenant_id=$3)", testID, coachID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND tenant_id=$2)", testID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	var coachTenantID int
	err = h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", req.CoachID).Scan(&coachTenantID)
	if err != nil || coachTenantID != tenantID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coach_id does not belong to your organization"})
		return
	}

	result, err := h.DB.Exec(
		`UPDATE tests SET title=$1, subject_id=$2, coach_id=$3, duration=$4, exam_date=$5 WHERE id=$6 AND tenant_id=$7`,
		req.Title, req.SubjectID, req.CoachID, req.Duration, req.ExamDate, testID, tenantID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test updated successfully"})
}

func (h *AdminHandler) DeleteTest(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	var tenantID int
	err = h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND coach_id=$2 AND tenant_id=$3)", testID, coachID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND tenant_id=$2)", testID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	result, err := h.DB.Exec("DELETE FROM tests WHERE id=$1 AND tenant_id=$2", testID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test deleted successfully"})
}

func (h *AdminHandler) UpdateQuestion(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}

	questionID, err := strconv.Atoi(c.Param("qid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}

	var req CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if validationErr := validateQuestionRequest(req); validationErr != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": validationErr})
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	var tenantID int
	err = h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND coach_id=$2 AND tenant_id=$3)", testID, coachID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND tenant_id=$2)", testID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	result, err := h.DB.Exec(
		`UPDATE questions SET
			question_text=$1, option_a=$2, option_b=$3, option_c=$4, option_d=$5,
			correct_answer=$6, marks=$7, neg_marks=$8, importance=$9, difficulty=$10,
			type=$11, expected_time=$12, concept_tag=$13
		 WHERE id=$14 AND test_id=$15`,
		req.QuestionText, req.OptionA, req.OptionB, req.OptionC, req.OptionD,
		req.CorrectAnswer, req.Marks, req.NegMarks,
		req.Importance, req.Difficulty, req.Type, req.ExpectedTime, req.ConceptTag,
		questionID, testID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "question updated successfully"})
}

func (h *AdminHandler) DeleteQuestion(c *gin.Context) {
	testID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test id"})
		return
	}

	questionID, err := strconv.Atoi(c.Param("qid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	var tenantID int
	err = h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	if role == "coach" {
		coachID, err := h.getCoachIDFromUser(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND coach_id=$2 AND tenant_id=$3)", testID, coachID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {
		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND tenant_id=$2)", testID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	result, err := h.DB.Exec(
		`DELETE FROM questions WHERE id=$1 AND test_id=$2`,
		questionID, testID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "question deleted successfully"})
}
