package handlers

import (
	"ai-student-diagnostic/backend/internal/helper"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CoachHandler struct {
	DB *sql.DB
}

func parseCoachPagination(c *gin.Context) (int, int) {
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

func NewCoachHandler(db *sql.DB) *CoachHandler {
	return &CoachHandler{DB: db}
}

func (h *CoachHandler) getCoachDetailsFromUser(userID int) (int, int, error) {
	var coachID int
	var tenantID int
	err := h.DB.QueryRow(
		"SELECT id, tenant_id FROM coaches WHERE user_id = $1 AND deleted_at IS NULL",
		userID,
	).Scan(&coachID, &tenantID)

	return coachID, tenantID, err
}

// GetStudentSQI
func (h *CoachHandler) GetStudentSQI(c *gin.Context) {
	idParam := c.Param("id")

	studentID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	//  Validate student belongs to coach/tenant
	var name string
	err = h.DB.QueryRow(
		"SELECT name FROM students WHERE id = $1 AND coach_id = $2 AND tenant_id = $3",
		studentID, coachID, tenantID,
	).Scan(&name)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found or access denied"})
		return
	}

	//  Fetch SQI results
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

func (h *CoachHandler) GetAssignmentResults(c *gin.Context) {
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
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	var exists bool
	err = h.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND tenant_id=$2 AND coach_id=$3)",
		studentID, tenantID, coachID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	var testID int
	var status string
	var assignedAt string
	var testTitle string
	err = h.DB.QueryRow(`
		SELECT a.test_id, a.status, a.assigned_at, t.title
		FROM assignments a
		JOIN tests t ON a.test_id = t.id
		WHERE a.id = $1 AND a.student_id = $2 AND a.coach_id = $3
	`, assignmentID, studentID, coachID).Scan(&testID, &status, &assignedAt, &testTitle)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	var studentName string
	var studentCode string
	err = h.DB.QueryRow(
		"SELECT name, student_code FROM students WHERE id=$1 AND tenant_id=$2 AND coach_id=$3",
		studentID, tenantID, coachID,
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
			"test":       gin.H{"id": testID, "title": testTitle},
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

func (h *CoachHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
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

type CreateCoachTestRequest struct {
	Title     string  `json:"title" binding:"required"`
	SubjectID int     `json:"subject_id" binding:"required"`
	Duration  int     `json:"duration" binding:"required"`
	ExamDate  *string `json:"exam_date"`
}

func (h *CoachHandler) CreateTest(c *gin.Context) {
	var req CreateCoachTestRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
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

func (h *CoachHandler) CreateQuestion(c *gin.Context) {
	testIDParam := c.Param("id")
	testID, err := strconv.Atoi(testIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id"})
		return
	}

	questions, err := parseQuestionRequests(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if len(questions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one question is required"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	// validate test and tenant isolation
	exists, err := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND coach_id=$2 AND tenant_id=$3)",
		testID, coachID, tenantID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id or access denied"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "failed to create questions"})
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

func (h *CoachHandler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	// validate student belongs to coach and tenant and is not deactivated
	var studentCoachID int
	err = h.DB.QueryRow(
		"SELECT coach_id FROM students WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL",
		req.StudentID, tenantID,
	).Scan(&studentCoachID)

	if err != nil || studentCoachID != coachID {
		c.JSON(http.StatusForbidden, gin.H{"error": "student not owned by coach, or deactivated"})
		return
	}

	// validate test belongs to coach and tenant
	var testCoachID int
	err = h.DB.QueryRow(
		"SELECT coach_id FROM tests WHERE id=$1 AND tenant_id=$2",
		req.TestID, tenantID,
	).Scan(&testCoachID)

	if err != nil || testCoachID != coachID {
		c.JSON(http.StatusForbidden, gin.H{"error": "test not owned by coach"})
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

func (h *CoachHandler) ListTests(c *gin.Context) {
	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := parseCoachPagination(c)
	search := c.Query("search")

	baseQuery := "FROM tests WHERE tenant_id=$1 AND coach_id=$2"
	countQuery := "SELECT COUNT(*) " + baseQuery
	dataQuery := "SELECT id, title, subject_id, coach_id, duration, exam_date " + baseQuery

	args := []interface{}{tenantID, coachID}

	if search != "" {
		baseQuery += " AND title ILIKE $" + strconv.Itoa(len(args)+1)
		countQuery = "SELECT COUNT(*) " + baseQuery
		dataQuery = "SELECT id, title, subject_id, coach_id, duration, exam_date " + baseQuery
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
		TestID    int     `json:"test_id"`
		Title     string  `json:"title"`
		SubjectID int     `json:"subject_id"`
		CoachID   int     `json:"coach_id"`
		Duration  int     `json:"duration"`
		ExamDate  *string `json:"exam_date"`
	}

	var tests []TestRow
	for rows.Next() {
		var t TestRow
		if err := rows.Scan(&t.TestID, &t.Title, &t.SubjectID, &t.CoachID, &t.Duration, &t.ExamDate); err != nil {
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

func (h *CoachHandler) ListStudents(c *gin.Context) {
	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := parseCoachPagination(c)

	includeDeactivated := c.Query("include_deactivated") == "true"

	var total int
	if includeDeactivated {
		err = h.DB.QueryRow("SELECT COUNT(*) FROM students WHERE tenant_id=$1 AND coach_id=$2", tenantID, coachID).Scan(&total)
	} else {
		err = h.DB.QueryRow("SELECT COUNT(*) FROM students WHERE tenant_id=$1 AND coach_id=$2 AND deleted_at IS NULL", tenantID, coachID).Scan(&total)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	query := "SELECT id, name, student_code, coach_id, deleted_at FROM students WHERE tenant_id=$1 AND coach_id=$2"
	if !includeDeactivated {
		query += " AND deleted_at IS NULL"
	}
	query += " ORDER BY id DESC LIMIT $3 OFFSET $4"

	rows, err := h.DB.Query(query, tenantID, coachID, limit, offset)
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

func (h *CoachHandler) GetStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
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
		WHERE st.id = $1 AND st.tenant_id = $2 AND st.coach_id = $3
	`, studentID, tenantID, coachID).Scan(
		&s.StudentID, &s.Name, &s.StudentCode, &s.CoachID, &s.CoachName,
		&s.CreatedAt, &s.DeletedAt, &s.DeletedByEmail, &s.DeletedByName, &s.DeletedByRole,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	c.JSON(http.StatusOK, s)
}

func (h *CoachHandler) ListStudentAssignments(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	var exists bool
	err = h.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND tenant_id=$2 AND coach_id=$3)",
		studentID, tenantID, coachID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	limit, offset := parseCoachPagination(c)

	var total int
	err = h.DB.QueryRow(
		"SELECT COUNT(*) FROM assignments WHERE student_id=$1 AND coach_id=$2",
		studentID, coachID,
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
		WHERE a.student_id = $1 AND a.coach_id = $2
		ORDER BY a.id DESC
		LIMIT $3 OFFSET $4
	`, studentID, coachID, limit, offset)
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

func (h *CoachHandler) DeleteStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	result, err := h.DB.Exec(
		`UPDATE students SET deleted_at = NOW(), deleted_by = $1
		 WHERE id = $2 AND tenant_id = $3 AND coach_id = $4 AND deleted_at IS NULL`,
		userID, studentID, tenantID, coachID,
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

func (h *CoachHandler) ReactivateStudent(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	result, err := h.DB.Exec(
		`UPDATE students SET deleted_at = NULL, deleted_by = NULL
		 WHERE id = $1 AND tenant_id = $2 AND coach_id = $3 AND deleted_at IS NOT NULL`,
		studentID, tenantID, coachID,
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

func (h *CoachHandler) ListSubjects(c *gin.Context) {
	userID := c.GetInt("user_id")
	_, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := parseCoachPagination(c)

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM subjects WHERE tenant_id=$1", tenantID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := h.DB.Query(
		"SELECT id, name FROM subjects WHERE tenant_id=$1 ORDER BY id DESC LIMIT $2 OFFSET $3",
		tenantID, limit, offset,
	)
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

func (h *CoachHandler) ListAssignments(c *gin.Context) {
	userID := c.GetInt("user_id")
	coachID, tenantID, err := h.getCoachDetailsFromUser(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := parseCoachPagination(c)
	testID := c.Query("test_id")

	baseQuery := "FROM assignments a JOIN students s ON a.student_id = s.id JOIN tests t ON a.test_id = t.id WHERE s.tenant_id=$1 AND a.coach_id=$2 AND s.deleted_at IS NULL"
	countQuery := "SELECT COUNT(*) " + baseQuery
	dataQuery := `SELECT a.id, a.student_id, s.name, s.student_code, a.test_id, t.title, a.coach_id, a.status, a.assigned_at ` + baseQuery

	args := []interface{}{tenantID, coachID}

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
