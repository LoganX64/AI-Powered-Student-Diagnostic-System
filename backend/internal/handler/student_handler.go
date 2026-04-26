package handlers

import (
	db "ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// student login

type StudentLoginRequest struct {
	StudentCode string `json:"student_code" binding:"required"`
}

func StudentLogin(c *gin.Context) {
	var req StudentLoginRequest

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

	token, err := utils.GenerateToken(0, "student", studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
	})
}

// submit answers
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

	//  Extract JWT claims
	studentIDRaw, exists := c.Get("student_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	studentID, ok := studentIDRaw.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token data"})
		return
	}

	//  Validate assignment ownership
	var ownerID int
	err := database.QueryRow(
		"SELECT student_id FROM assignments WHERE id = $1",
		req.AssignmentID,
	).Scan(&ownerID)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment"})
		return
	}

	if ownerID != studentID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "assignment does not belong to student",
		})
		return
	}

	//  Start transaction
	tx, err := database.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	//  Create attempt
	var attemptID int
	err = tx.QueryRow(
		"INSERT INTO attempts (assignment_id) VALUES ($1) RETURNING id",
		req.AssignmentID,
	).Scan(&attemptID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attempt"})
		return
	}

	//  Collect question IDs
	var qIDs []int
	for _, ans := range req.Answers {
		qIDs = append(qIDs, ans.QuestionID)
	}

	//  Bulk fetch questions
	rows, err := tx.Query(`
		SELECT id, correct_answer, marks, neg_marks, importance, difficulty, type, expected_time
		FROM questions
		WHERE id = ANY($1)
	`, pq.Array(qIDs))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch questions"})
		return
	}
	defer rows.Close()

	// Maps
	qMap := make(map[int]services.QuestionMeta)
	correctMap := make(map[int]string)

	for rows.Next() {
		var q services.QuestionMeta
		var correct string

		err := rows.Scan(
			&q.QuestionID,
			&correct,
			&q.Marks,
			&q.NegMarks,
			&q.Importance,
			&q.Difficulty,
			&q.Type,
			&q.ExpectedTime,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "question scan failed"})
			return
		}

		qMap[q.QuestionID] = q
		correctMap[q.QuestionID] = correct
	}

	//  Prepare SQI input
	var questionMetaList []services.QuestionMeta
	var answerLogs []services.AnswerLog

	for _, ans := range req.Answers {

		q, exists := qMap[ans.QuestionID]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
			return
		}

		correctAnswer := correctMap[ans.QuestionID]
		isCorrect := ans.SelectedAnswer == correctAnswer

		_, err = tx.Exec(`
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

		questionMetaList = append(questionMetaList, q)

		answerLogs = append(answerLogs, services.AnswerLog{
			QuestionID:        ans.QuestionID,
			SelectedAnswer:    ans.SelectedAnswer,
			CorrectAnswer:     correctAnswer,
			TimeSpent:         ans.TimeSpent,
			MarkedForReview:   ans.MarkedForReview,
			Revisited:         ans.Revisited,
			ChangedAnswer:     ans.ChangedAnswer,
			WasInitiallyWrong: ans.WasInitiallyWrong,
		})
	}

	//  Calculate SQI
	result := services.CalculateSQI(questionMetaList, answerLogs)

	//  Store result
	_, err = tx.Exec(`
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

	//  Commit
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"attempt_id": attemptID,
		"sqi_score":  result.OverallSQI,
	})
}
