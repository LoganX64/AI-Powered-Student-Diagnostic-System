package handlers

import (
	db "ai-student-diagnostic/backend/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

// login

type LoginRequest struct {
	StudentCode string `json:"student_code" binding:"required"`
}

func StudentLogin(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	database := db.GetDB()

	var studentID int
	err := database.QueryRow(
		"SELECT id FROM students WHERE student_code = $1",
		req.StudentCode,
	).Scan(&studentID)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid student code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"student_id": studentID,
	})
}

// submit
type Answer struct {
	QuestionID        int     `json:"question_id" binding:"required"`
	SelectedAnswer    string  `json:"selected_answer" binding:"required"`
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

	//  Create attempt
	var attemptID int
	err := database.QueryRow(
		"INSERT INTO attempts (assignment_id) VALUES ($1) RETURNING id",
		req.AssignmentID,
	).Scan(&attemptID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prepare SQI inputs
	var questionMetaList []QuestionMeta
	var answerLogs []AnswerLog

	//  Process answers
	for _, ans := range req.Answers {

		var q QuestionMeta
		var correctAnswer string

		err := database.QueryRow(`
			SELECT id, correct_answer, marks, neg_marks, importance, difficulty, type, expected_time
			FROM questions
			WHERE id = $1
		`, ans.QuestionID).Scan(
			&q.QuestionID,
			&correctAnswer,
			&q.Marks,
			&q.NegMarks,
			&q.Importance,
			&q.Difficulty,
			&q.Type,
			&q.ExpectedTime,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "question fetch failed"})
			return
		}

		questionMetaList = append(questionMetaList, q)

		answerLogs = append(answerLogs, AnswerLog{
			QuestionID:        ans.QuestionID,
			SelectedAnswer:    ans.SelectedAnswer,
			CorrectAnswer:     correctAnswer,
			TimeSpent:         ans.TimeSpent,
			MarkedForReview:   ans.MarkedForReview,
			Revisited:         ans.Revisited,
			ChangedAnswer:     ans.ChangedAnswer,
			WasInitiallyWrong: ans.WasInitiallyWrong,
		})

		isCorrect := ans.SelectedAnswer == correctAnswer

		_, err = database.Exec(`
			INSERT INTO answer_logs 
			(question_id, attempt_id, selected_answer, is_correct, time_spent, marked_for_review, revisited, changed_answer, was_initially_wrong)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`,
			ans.QuestionID,
			attemptID,
			ans.SelectedAnswer,
			isCorrect,
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
	}

	//  Call SQI Engine
	result := CalculateSQI(questionMetaList, answerLogs)

	// Store result
	_, err = database.Exec(`
		INSERT INTO attempt_results (attempt_id, sqi_score)
		VALUES ($1,$2)
	`,
		attemptID,
		result.OverallSQI,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store result"})
		return
	}

	//  Response
	c.JSON(http.StatusOK, gin.H{
		"attempt_id": attemptID,
		"sqi_score":  result.OverallSQI,
	})
}
