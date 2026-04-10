package handlers

import (
	db "ai-student-diagnostic/backend/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Login
type LoginRequest struct {
	StudentCode string `json:"student_code" binding:"required"`
}

func StudentLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database := db.GetDB()

	var studentID int
	err := database.QueryRow("SELECT id FROM students WHERE student_code = $1", req.StudentCode).Scan(&studentID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid student code"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"student_id": studentID})

}

// submit answers
type Answer struct {
	QuestionID        int     `json:"question_id" binding:"required"`
	SelectedAnswer    string  `json:"selected_answer" binding:"required"`
	CorrectAnswer     string  `json:"correct_answer" binding:"required"`
	TimeSpent         float64 `json:"time_spent" binding:"required"`
	MarkedForReview   bool    `json:"marked_for_review"`
	Revisited         bool    `json:"revisited"`
	ChangedAnswer     bool    `json:"changed_answer"`
	WasInitiallyWrong bool    `json:"was_initially_wrong"`
}

type SubmitRequest struct {
	AssignmentID int      `json:"assignment_id" binding:"required"`
	Answers      []Answer `json:"answers" binding:"required"`
}

func SubmitAnswers(c *gin.Context) {
	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	database := db.GetDB()

	// create attempt
	var attemptID int
	err := database.QueryRow(
		"INSERT INTO attempts (assignment_id) VALUES ($1) RETURNING id",
		req.AssignmentID,
	).Scan(&attemptID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attempt"})
		return
	}

	// insert answers
	for _, ans := range req.Answers {
		_, err := database.Exec(`
			INSERT INTO answer_logs 
			(question_id, attempt_id, selected_answer, correct_answer, time_spent, marked_for_review, revisited, changed_answer, was_initially_wrong)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`,
			ans.QuestionID,
			attemptID,
			ans.SelectedAnswer,
			ans.CorrectAnswer,
			ans.TimeSpent,
			ans.MarkedForReview,
			ans.Revisited,
			ans.ChangedAnswer,
			ans.WasInitiallyWrong,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert answer"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"attempt_id": attemptID,
			"status":     "submitted",
		})
	}

}
