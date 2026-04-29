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
		"SELECT id FROM coaches WHERE user_id = $1",
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
	AttemptID int                  `json:"attempt_id"`
	TestID    int                  `json:"test_id"`
	TestTitle string               `json:"test_title"`
	SQI       float64              `json:"sqi_score"`
	Analysis  services.SQIAnalysis `json:"analysis"`
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

	// optional query param: include_analysis=true
	includeAnalysis := c.Query("include_analysis") == "true"

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
		"average_sqi": helper.Round(avgSQI, 2),
		"total_tests": len(results),
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

		analysis, err := h.calculateAttemptSQIAnalysis(result.AttemptID, result.TestID)
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

	c.JSON(http.StatusOK, gin.H{
		"student_id":     studentID,
		"student_name":   studentName,
		"subject_id":     subjectID,
		"subject_name":   subjectName,
		"test_id":        testID,
		"results":        results,
		"average_sqi":    helper.Round(averageSQI, 2),
		"total_attempts": len(results),
		"calculation":    "sqi_engine",
	})
}

func (h *AdminHandler) calculateAttemptSQIAnalysis(attemptID int, testID int) (services.SQIAnalysis, error) {
	questionRows, err := h.DB.Query(`
		SELECT id, marks, neg_marks, importance, difficulty, type, expected_time
		FROM questions
		WHERE test_id = $1
		ORDER BY id
	`, testID)
	if err != nil {
		return services.SQIAnalysis{}, err
	}
	defer questionRows.Close()

	var questions []services.QuestionMeta
	for questionRows.Next() {
		var q services.QuestionMeta
		if err := questionRows.Scan(
			&q.QuestionID,
			&q.Marks,
			&q.NegMarks,
			&q.Importance,
			&q.Difficulty,
			&q.Type,
			&q.ExpectedTime,
		); err != nil {
			return services.SQIAnalysis{}, err
		}
		questions = append(questions, q)
	}
	if err := questionRows.Err(); err != nil {
		return services.SQIAnalysis{}, err
	}

	answerRows, err := h.DB.Query(`
		SELECT
			al.question_id,
			COALESCE(al.selected_answer, ''),
			q.correct_answer,
			COALESCE(al.time_spent, 0),
			COALESCE(al.marked_for_review, false),
			COALESCE(al.revisited, false),
			COALESCE(al.changed_answer, false),
			COALESCE(al.was_initially_wrong, false)
		FROM answer_logs al
		JOIN questions q ON al.question_id = q.id
		WHERE al.attempt_id = $1
	`, attemptID)
	if err != nil {
		return services.SQIAnalysis{}, err
	}
	defer answerRows.Close()

	var answers []services.AnswerLog
	for answerRows.Next() {
		var a services.AnswerLog
		if err := answerRows.Scan(
			&a.QuestionID,
			&a.SelectedAnswer,
			&a.CorrectAnswer,
			&a.TimeSpent,
			&a.MarkedForReview,
			&a.Revisited,
			&a.ChangedAnswer,
			&a.WasInitiallyWrong,
		); err != nil {
			return services.SQIAnalysis{}, err
		}
		a.Seen = true
		answers = append(answers, a)
	}
	if err := answerRows.Err(); err != nil {
		return services.SQIAnalysis{}, err
	}

	return services.CalculateSQIAnalysis(questions, answers), nil
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
			err = h.DB.QueryRow("SELECT id FROM coaches WHERE user_id = $1", userID).Scan(&coachID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "coach_id is required, or you must create a coach profile for yourself first"})
				return
			}
		} else {

			var exists bool
			err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1 AND tenant_id=$2)", req.CoachID, tenantID).Scan(&exists)
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
	Title     string `json:"title" binding:"required"`
	SubjectID int    `json:"subject_id" binding:"required"`
	CoachID   int    `json:"coach_id" binding:"required"`
	Duration  int    `json:"duration" binding:"required"`
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
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1 AND tenant_id=$2)", req.CoachID, tenantID).Scan(&exists)
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
		INSERT INTO tests (tenant_id, title, subject_id, coach_id, duration)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id
	`, tenantID, req.Title, req.SubjectID, coachID, req.Duration).Scan(&id)

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

	// validate student belongs to coach and same tenant
	var studentCoachID int
	var studentTenantID int
	err = h.DB.QueryRow(
		"SELECT coach_id, tenant_id FROM students WHERE id=$1",
		req.StudentID,
	).Scan(&studentCoachID, &studentTenantID)

	if err != nil || studentCoachID != coachID || studentTenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "student not found or not in your organization"})
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
