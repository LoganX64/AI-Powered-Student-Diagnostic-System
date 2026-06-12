package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"database/sql"
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
		"SELECT id, tenant_id FROM coaches WHERE user_id = $1",
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
	rows, err := h.DB.Query(`
		SELECT ar.attempt_id, ass.test_id, ar.sqi_score
		FROM attempt_results ar
		JOIN attempts a ON ar.attempt_id = a.id
		JOIN assignments ass ON a.assignment_id = ass.id
		WHERE ass.student_id = $1
	`, studentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch results"})
		return
	}
	defer rows.Close()

	var results []AttemptResult
	var total float64

	for rows.Next() {
		var r AttemptResult

		if err := rows.Scan(&r.AttemptID, &r.TestID, &r.SQI); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
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
		"average_sqi": avgSQI,
		"total_tests": len(results),
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

func (h *CoachHandler) CreateTest(c *gin.Context) {
	var req CreateTestRequest

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

	// validate student belongs to coach and tenant
	var studentCoachID int
	err = h.DB.QueryRow(
		"SELECT coach_id FROM students WHERE id=$1 AND tenant_id=$2",
		req.StudentID, tenantID,
	).Scan(&studentCoachID)

	if err != nil || studentCoachID != coachID {
		c.JSON(http.StatusForbidden, gin.H{"error": "student not owned by coach"})
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

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM students WHERE tenant_id=$1 AND coach_id=$2", tenantID, coachID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := h.DB.Query(
		"SELECT id, name, student_code, coach_id FROM students WHERE tenant_id=$1 AND coach_id=$2 ORDER BY id DESC LIMIT $3 OFFSET $4",
		tenantID, coachID, limit, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type StudentRow struct {
		StudentID   int    `json:"student_id"`
		Name        string `json:"name"`
		StudentCode string `json:"student_code"`
		CoachID     int    `json:"coach_id"`
	}

	var students []StudentRow
	for rows.Next() {
		var s StudentRow
		if err := rows.Scan(&s.StudentID, &s.Name, &s.StudentCode, &s.CoachID); err != nil {
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

	baseQuery := "FROM assignments a JOIN students s ON a.student_id = s.id JOIN tests t ON a.test_id = t.id WHERE s.tenant_id=$1 AND a.coach_id=$2"
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
		ID          int     `json:"id"`
		StudentID   int     `json:"student_id"`
		StudentName string  `json:"student_name"`
		StudentCode string  `json:"student_code"`
		TestID      int     `json:"test_id"`
		TestTitle   string  `json:"test_title"`
		CoachID     int     `json:"coach_id"`
		Status      string  `json:"status"`
		AssignedAt  string  `json:"assigned_at"`
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
