package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

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

	//  Validate student
	var name string
	err = h.DB.QueryRow(
		"SELECT name FROM students WHERE id = $1",
		studentID,
	).Scan(&name)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	//  Fetch all SQI results
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

		err := rows.Scan(&r.AttemptID, &r.TestID, &r.SQI)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}

		results = append(results, r)
		total += r.SQI
	}

	//  Calculate average SQI
	var avgSQI float64
	if len(results) > 0 {
		avgSQI = total / float64(len(results))
	}

	// Response
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
}

func (h *AdminHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	exists, err := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM students WHERE student_code=$1)",
		req.StudentCode,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "validation failed"})
		return
	}

	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student_code already exists"})
		return
	}

	var id int
	err = h.DB.QueryRow(`
		INSERT INTO students (name, student_code)
		VALUES ($1,$2) RETURNING id
	`, req.Name, req.StudentCode).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create student"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"student_id": id})
}

// Add Coach
type CreateCoachRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

func (h *AdminHandler) CreateCoach(c *gin.Context) {
	var req CreateCoachRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	exists, _ := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM coaches WHERE email=$1)",
		req.Email,
	)

	if exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already exists"})
		return
	}

	var id int
	err := h.DB.QueryRow(`
		INSERT INTO coaches (name, email)
		VALUES ($1,$2) RETURNING id
	`, req.Name, req.Email).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create coach"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"coach_id": id})
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

	var id int
	err := h.DB.QueryRow(`
		INSERT INTO subjects (name)
		VALUES ($1) RETURNING id
	`, req.Name).Scan(&id)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "subject already exists"})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// validate subject
	subjectExists, err := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM subjects WHERE id=$1)",
		req.SubjectID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !subjectExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
		return
	}

	// validate coach
	coachExists, err := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1)",
		req.CoachID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !coachExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach_id"})
		return
	}

	var id int
	err = h.DB.QueryRow(`
		INSERT INTO tests (title, subject_id, coach_id, duration)
		VALUES ($1,$2,$3,$4)
		RETURNING id
	`,
		req.Title,
		req.SubjectID,
		req.CoachID,
		req.Duration,
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   err.Error(),
			"message": "failed to create test",
		})
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

	// validate test
	exists, err := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1)",
		req.TestID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id"})
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
	// student
	studentExists, _ := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM students WHERE id=$1)",
		req.StudentID,
	)

	// test
	testExists, _ := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM tests WHERE id=$1)",
		req.TestID,
	)

	// coach
	coachExists, _ := repository.Exists(
		h.DB,
		"SELECT EXISTS(SELECT 1 FROM coaches WHERE id=$1)",
		req.CoachID,
	)

	if !studentExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student_id"})
		return
	}
	if !testExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test_id"})
		return
	}
	if !coachExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach_id"})
		return
	}

	duplicate, _ := repository.Exists(
		h.DB,
		`SELECT EXISTS(
		SELECT 1 FROM assignments 
		WHERE student_id=$1 AND test_id=$2 AND coach_id=$3
	)`,
		req.StudentID,
		req.TestID,
		req.CoachID,
	)

	if duplicate {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "assignment already exists",
		})
		return
	}

	var id int
	err := h.DB.QueryRow(`
		INSERT INTO assignments (student_id, test_id, coach_id)
		VALUES ($1,$2,$3) RETURNING id
	`,
		req.StudentID,
		req.TestID,
		req.CoachID,
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create assignment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"assignment_id": id})
}
