package handlers

import (
	db "ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
	Answers []Answer `json:"answers" binding:"required"`
}

func SubmitAnswers(c *gin.Context) {
	assignmentIDParam := c.Param("id")
	assignmentID, err := strconv.Atoi(assignmentIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment_id"})
		return
	}

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

	var ownerID int
	var testID int
	err = database.QueryRow(
		"SELECT student_id, test_id FROM assignments WHERE id = $1",
		assignmentID,
	).Scan(&ownerID, &testID)

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
		assignmentID,
	).Scan(&attemptID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attempt"})
		return
	}

	rows, err := tx.Query(`
		SELECT id, correct_answer, marks, neg_marks, importance, difficulty, type, expected_time, concept_tag
		FROM questions
		WHERE test_id = $1
	`, testID)

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
			&q.ConceptTag,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "question scan failed"})
			return
		}

		qMap[q.QuestionID] = q
		correctMap[q.QuestionID] = correct
	}

	var questionMetaList []services.QuestionMeta
	var answerLogs []services.AnswerLog
	for _, q := range qMap {
		questionMetaList = append(questionMetaList, q)
	}

	for _, ans := range req.Answers {

		_, exists := qMap[ans.QuestionID]
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

	analysis := services.CalculateSQIAnalysis(questionMetaList, answerLogs)
	analysisJSON, err := json.Marshal(analysis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode analysis"})
		return
	}

	_, err = tx.Exec(`
		INSERT INTO attempt_results (attempt_id, sqi_score, analysis_json)
		VALUES ($1,$2,$3)
	`,
		attemptID,
		analysis.OverallSQI,
		analysisJSON,
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
		"sqi_score":  analysis.OverallSQI,
		"analysis":   analysis,
	})
}
