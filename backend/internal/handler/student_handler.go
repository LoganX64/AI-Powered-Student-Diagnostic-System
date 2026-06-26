package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type StudentHandler struct {
	StudentRepo    *repository.StudentRepo
	AssignmentRepo *repository.AssignmentRepo
	AttemptRepo    *repository.AttemptRepo
	TestRepo       *repository.TestRepo
}

func NewStudentHandler(
	studentRepo *repository.StudentRepo,
	assignmentRepo *repository.AssignmentRepo,
	attemptRepo *repository.AttemptRepo,
	testRepo *repository.TestRepo,
) *StudentHandler {
	return &StudentHandler{
		StudentRepo:    studentRepo,
		AssignmentRepo: assignmentRepo,
		AttemptRepo:    attemptRepo,
		TestRepo:       testRepo,
	}
}

type StudentLoginRequest struct {
	StudentCode string `json:"student_code" binding:"required"`
}

func (h *StudentHandler) StudentLogin(c *gin.Context) {
	var req StudentLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var studentID int
	err := h.StudentRepo.DB.QueryRow("SELECT id FROM students WHERE student_code = $1", req.StudentCode).Scan(&studentID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid student code"})
		return
	}

	token, err := utils.GenerateToken(0, "student", studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": token})
}

type Answer struct {
	QuestionID        int     `json:"question_id" binding:"required"`
	SelectedAnswer    string  `json:"selected_answer"`
	TimeSpent         float64 `json:"time_spent"`
	Seen              *bool   `json:"seen"`
	MarkedForReview   bool    `json:"marked_for_review"`
	Revisited         bool    `json:"revisited"`
	ChangedAnswer     bool    `json:"changed_answer"`
	WasInitiallyWrong bool    `json:"was_initially_wrong"`
}

type SubmitRequest struct {
	Answers []Answer `json:"answers" binding:"required"`
}

func (h *StudentHandler) SubmitAnswers(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment_id"})
		return
	}

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

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

	var ownerID, testID, duration int
	err = h.AssignmentRepo.DB.QueryRow(`
		SELECT ass.student_id, ass.test_id, COALESCE(t.duration, 0)
		FROM assignments ass JOIN tests t ON ass.test_id = t.id
		WHERE ass.id = $1
	`, assignmentID).Scan(&ownerID, &testID, &duration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment"})
		return
	}

	if ownerID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "assignment does not belong to student"})
		return
	}

	existsAttempt, _ := h.AttemptRepo.ExistsByAssignment(assignmentID)
	if existsAttempt {
		c.JSON(http.StatusConflict, gin.H{"error": "assignment already submitted"})
		return
	}

	tx, err := h.AttemptRepo.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	var attemptID int
	err = tx.QueryRow("INSERT INTO attempts (assignment_id, submitted_at) VALUES ($1, NOW()) RETURNING id", assignmentID).Scan(&attemptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attempt"})
		return
	}

	rows, err := tx.Query("SELECT q.id, q.correct_answer FROM questions q WHERE q.test_id = $1", testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch questions"})
		return
	}
	defer rows.Close()

	correctMap := make(map[int]string)
	for rows.Next() {
		var questionID int
		var correct string
		if err := rows.Scan(&questionID, &correct); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "question scan failed"})
			return
		}
		correctMap[questionID] = correct
	}

	seenQuestionIDs := make(map[int]bool)
	var totalTimeSpent float64

	for _, ans := range req.Answers {
		_, exists := correctMap[ans.QuestionID]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
			return
		}

		if seenQuestionIDs[ans.QuestionID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate question id"})
			return
		}
		seenQuestionIDs[ans.QuestionID] = true

		if ans.TimeSpent < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "time_spent cannot be negative"})
			return
		}

		answerSeen := ans.SelectedAnswer != ""
		if ans.Seen != nil {
			answerSeen = *ans.Seen
		}
		if ans.Seen != nil && !*ans.Seen && ans.SelectedAnswer != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not seen question cannot have selected_answer"})
			return
		}
		if ans.SelectedAnswer != "" && ans.SelectedAnswer != "A" && ans.SelectedAnswer != "B" && ans.SelectedAnswer != "C" && ans.SelectedAnswer != "D" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "selected_answer must be A/B/C/D"})
			return
		}

		if !answerSeen {
			ans.TimeSpent = 0
			ans.MarkedForReview = false
			ans.Revisited = false
			ans.ChangedAnswer = false
			ans.WasInitiallyWrong = false
		}

		totalTimeSpent += ans.TimeSpent
		if duration > 0 && totalTimeSpent > float64(duration) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":            "total time_spent exceeds test duration",
				"total_time_spent": helperRoundForResponse(totalTimeSpent),
				"test_duration":    duration,
			})
			return
		}

		correctAnswer := correctMap[ans.QuestionID]
		isCorrect := answerSeen && ans.SelectedAnswer != "" && ans.SelectedAnswer == correctAnswer

		_, err = tx.Exec(`
			INSERT INTO answer_logs 
			(question_id, attempt_id, selected_answer, is_correct, time_spent, marked_for_review, revisited, changed_answer, was_initially_wrong, seen)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`, ans.QuestionID, attemptID, ans.SelectedAnswer, isCorrect, ans.TimeSpent,
			ans.MarkedForReview, ans.Revisited, ans.ChangedAnswer, ans.WasInitiallyWrong, answerSeen)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert answer"})
			return
		}
	}

	_, err = tx.Exec("UPDATE assignments SET status = 'submitted' WHERE id = $1", assignmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark assignment submitted"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"attempt_id":       attemptID,
		"total_time_spent": helperRoundForResponse(totalTimeSpent),
		"test_duration":    duration,
	})
}

func (h *StudentHandler) ListStudentAssignments(c *gin.Context) {
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

	assignments, total, err := h.AssignmentRepo.ListByStudent(studentID, nil, 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch assignments"})
		return
	}

	if assignments == nil {
		assignments = []repository.AssignmentRow{}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "data": assignments})
}

func (h *StudentHandler) GetAssignmentQuestions(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment_id"})
		return
	}

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

	var ownerID, testID int
	var testTitle string
	var duration int
	var examDate sql.NullTime

	err = h.AssignmentRepo.DB.QueryRow(`
		SELECT ass.student_id, ass.test_id, t.title, COALESCE(t.duration, 0), t.exam_date
		FROM assignments ass JOIN tests t ON ass.test_id = t.id
		WHERE ass.id = $1
	`, assignmentID).Scan(&ownerID, &testID, &testTitle, &duration, &examDate)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	if ownerID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "assignment does not belong to student"})
		return
	}

	existsAttempt, _ := h.AttemptRepo.ExistsByAssignment(assignmentID)
	if existsAttempt {
		c.JSON(http.StatusConflict, gin.H{"error": "assignment already submitted"})
		return
	}

	questions, _, err := h.TestRepo.ListQuestions(testID, 1000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch questions"})
		return
	}

	if questions == nil {
		questions = []repository.QuestionRow{}
	}

	type QuestionResponse struct {
		ID           int     `json:"id"`
		QuestionText string  `json:"question_text"`
		OptionA      string  `json:"option_a"`
		OptionB      string  `json:"option_b"`
		OptionC      string  `json:"option_c"`
		OptionD      string  `json:"option_d"`
		Marks        float64 `json:"marks"`
		NegMarks     float64 `json:"neg_marks"`
		Difficulty   string  `json:"difficulty"`
		Type         string  `json:"type"`
		ExpectedTime float64 `json:"expected_time"`
		ConceptTag   string  `json:"concept_tag"`
	}

	var responseQuestions []QuestionResponse
	for _, q := range questions {
		responseQuestions = append(responseQuestions, QuestionResponse{
			ID: q.ID, QuestionText: q.QuestionText,
			OptionA: q.OptionA, OptionB: q.OptionB, OptionC: q.OptionC, OptionD: q.OptionD,
			Marks: q.Marks, NegMarks: q.NegMarks, Difficulty: q.Difficulty, Type: q.Type,
			ExpectedTime: q.ExpectedTime, ConceptTag: q.ConceptTag,
		})
	}

	examDateStr := ""
	if examDate.Valid {
		examDateStr = examDate.Time.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, gin.H{
		"assignment_id": assignmentID,
		"test_title":    testTitle,
		"duration":      duration,
		"exam_date":     examDateStr,
		"questions":     responseQuestions,
	})
}

func helperRoundForResponse(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
