package handlers

import (
	db "ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"database/sql"
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
	var duration int
	err = database.QueryRow(
		`
		SELECT ass.student_id, ass.test_id, COALESCE(t.duration, 0)
		FROM assignments ass
		JOIN tests t ON ass.test_id = t.id
		WHERE ass.id = $1
		`,
		assignmentID,
	).Scan(&ownerID, &testID, &duration)

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

	//  Check if attempt already exists for this assignment (prevent re-attempt)
	var existingAttemptID int
	err = database.QueryRow(
		"SELECT id FROM attempts WHERE assignment_id = $1",
		assignmentID,
	).Scan(&existingAttemptID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "assignment already submitted",
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
		"INSERT INTO attempts (assignment_id, submitted_at) VALUES ($1, NOW()) RETURNING id",
		assignmentID,
	).Scan(&attemptID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create attempt"})
		return
	}

	rows, err := tx.Query(`
		SELECT q.id, q.correct_answer, q.marks, q.neg_marks,
		       CASE q.importance WHEN 'A' THEN 'high' WHEN 'B' THEN 'medium' WHEN 'C' THEN 'low' END,
		       q.difficulty,
		       CASE q.type WHEN 'Theory' THEN 'mcq' WHEN 'Practical' THEN 'integer' END,
		       q.expected_time, q.concept_tag,
		       COALESCE(s.name, 'Uncategorized')
		FROM questions q
		JOIN tests t ON q.test_id = t.id
		LEFT JOIN subjects s ON t.subject_id = s.id
		WHERE q.test_id = $1
	`, testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch questions"})
		return
	}
	defer rows.Close()

	// Maps
	qMap := make(map[int]services.QuestionMetaV2)
	correctMap := make(map[int]string)

	for rows.Next() {
		var q services.QuestionMetaV2
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
			&q.Subject,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "question scan failed"})
			return
		}

		qMap[q.QuestionID] = q
		correctMap[q.QuestionID] = correct
	}

	seenQuestionIDs := make(map[int]bool)
	var totalTimeSpent float64

	for _, ans := range req.Answers {

		_, exists := qMap[ans.QuestionID]
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
		if ans.SelectedAnswer != "" &&
			ans.SelectedAnswer != "A" &&
			ans.SelectedAnswer != "B" &&
			ans.SelectedAnswer != "C" &&
			ans.SelectedAnswer != "D" {
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
			answerSeen,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert answer"})
			return
		}

	}

	_, err = tx.Exec(
		"UPDATE assignments SET status = 'submitted' WHERE id = $1",
		assignmentID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark assignment submitted"})
		return
	}

	//  Commit
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

// list student assignments
func ListStudentAssignments(c *gin.Context) {
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

	database := db.GetDB()

	rows, err := database.Query(`
		SELECT a.id, a.test_id, t.title, a.status, a.assigned_at
		FROM assignments a
		JOIN tests t ON a.test_id = t.id
		WHERE a.student_id = $1
		ORDER BY a.assigned_at DESC
	`, studentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch assignments"})
		return
	}
	defer rows.Close()

	type AssignmentItem struct {
		ID         int    `json:"id"`
		TestID     int    `json:"test_id"`
		TestTitle  string `json:"test_title"`
		Status     string `json:"status"`
		AssignedAt string `json:"assigned_at"`
	}

	var assignments []AssignmentItem
	for rows.Next() {
		var a AssignmentItem
		if err := rows.Scan(&a.ID, &a.TestID, &a.TestTitle, &a.Status, &a.AssignedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan assignment"})
			return
		}
		assignments = append(assignments, a)
	}

	if assignments == nil {
		assignments = []AssignmentItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(assignments),
		"data":  assignments,
	})
}

// get assignment questions (student-facing, excludes correct_answer)
func GetAssignmentQuestions(c *gin.Context) {
	assignmentIDParam := c.Param("id")
	assignmentID, err := strconv.Atoi(assignmentIDParam)
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

	database := db.GetDB()

	// Verify assignment belongs to student and get test info
	var ownerID int
	var testID int
	var testTitle string
	var duration int
	var examDate sql.NullTime

	err = database.QueryRow(`
		SELECT ass.student_id, ass.test_id, t.title, COALESCE(t.duration, 0), t.exam_date
		FROM assignments ass
		JOIN tests t ON ass.test_id = t.id
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

	// Check if already submitted
	var existingAttemptID int
	err = database.QueryRow(
		"SELECT id FROM attempts WHERE assignment_id = $1",
		assignmentID,
	).Scan(&existingAttemptID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "assignment already submitted"})
		return
	}

	// Fetch questions (exclude correct_answer)
	rows, err := database.Query(`
		SELECT id, question_text, option_a, option_b, option_c, option_d,
		       marks, neg_marks, difficulty, type, expected_time, concept_tag
		FROM questions
		WHERE test_id = $1
		ORDER BY id ASC
	`, testID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch questions"})
		return
	}
	defer rows.Close()

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

	var questions []QuestionResponse
	for rows.Next() {
		var q QuestionResponse
		if err := rows.Scan(
			&q.ID, &q.QuestionText, &q.OptionA, &q.OptionB, &q.OptionC, &q.OptionD,
			&q.Marks, &q.NegMarks, &q.Difficulty, &q.Type, &q.ExpectedTime, &q.ConceptTag,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan question"})
			return
		}
		questions = append(questions, q)
	}

	if questions == nil {
		questions = []QuestionResponse{}
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
		"questions":     questions,
	})
}

func helperRoundForResponse(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
