package handlers

import (
	"database/sql"
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
	AttemptID int     `json:"attempt_id"`
	TestID    int     `json:"test_id"`
	SQI       float64 `json:"sqi_score"`
}

// get student SQI results and average for a given student ID.
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
	err = h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1 AND tenant_id IS NOT NULL", userID).Scan(&tenantID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied: organization info missing"})
		return
	}

	// Coach specific assignment check
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
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied: you are not assigned to this student"})
			return
		}
	}

	//  validation
	var name string
	err = h.DB.QueryRow(
		"SELECT name FROM students WHERE id = $1 AND tenant_id = $2",
		studentID, tenantID,
	).Scan(&name)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found in your organization"})
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

	//  Average
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
	TestID       int    `json:"test_id" binding:"required"`
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

func (h *AdminHandler) CreateQuestion(c *gin.Context) {
	var req CreateQuestionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	var tenantID int
	err := h.DB.QueryRow("SELECT tenant_id FROM users WHERE id=$1", userID).Scan(&tenantID)
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
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND coach_id=$2 AND tenant_id=$3)", req.TestID, coachID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "test not found or not owned by you"})
			return
		}
	} else if role == "admin" {

		var exists bool
		err = h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1 AND tenant_id=$2)", req.TestID, tenantID).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id for your organization"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized role"})
		return
	}

	// validate required fields
	if req.QuestionText == "" ||
		req.OptionA == "" ||
		req.OptionB == "" ||
		req.OptionC == "" ||
		req.OptionD == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "question_text and all options are required",
		})
		return
	}

	// validate duplicate options
	if req.OptionA == req.OptionB ||
		req.OptionA == req.OptionC ||
		req.OptionA == req.OptionD {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "duplicate options not allowed",
		})
		return
	}

	// validate correct answer
	if req.CorrectAnswer != "A" &&
		req.CorrectAnswer != "B" &&
		req.CorrectAnswer != "C" &&
		req.CorrectAnswer != "D" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "correct_answer must be A/B/C/D",
		})
		return
	}

	var id int
	err = h.DB.QueryRow(`
		INSERT INTO questions 
		(test_id, question_text, option_a, option_b, option_c, option_d,
		 correct_answer, marks, neg_marks, importance, difficulty, type, expected_time, concept_tag)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id
	`,
		req.TestID,
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

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "failed to create question",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"question_id": id})
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
