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
