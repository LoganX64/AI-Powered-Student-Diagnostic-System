package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
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

	studentID, err := h.StudentRepo.GetIDByStudentCode(req.StudentCode)
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

	owner, err := h.AssignmentRepo.GetOwnerAndTest(assignmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment"})
		return
	}

	if owner.OwnerID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "assignment does not belong to student"})
		return
	}

	existsAttempt, err := h.AttemptRepo.ExistsByAssignment(assignmentID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to check assignment status")
		return
	}
	if existsAttempt {
		c.JSON(http.StatusConflict, gin.H{"error": "assignment already submitted"})
		return
	}

	correctMap, err := h.AttemptRepo.GetCorrectAnswers(owner.TestID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch questions")
		return
	}

	attemptID, err := h.AttemptRepo.CreateAttempt(assignmentID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create attempt")
		return
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
		if owner.Duration > 0 && totalTimeSpent > float64(owner.Duration) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":            "total time_spent exceeds test duration",
				"total_time_spent": helperRoundForResponse(totalTimeSpent),
				"test_duration":    owner.Duration,
			})
			return
		}

		correctAnswer := correctMap[ans.QuestionID]
		isCorrect := answerSeen && ans.SelectedAnswer != "" && ans.SelectedAnswer == correctAnswer

		err = h.AttemptRepo.InsertAnswerLog(attemptID, ans.QuestionID, ans.SelectedAnswer, isCorrect, ans.TimeSpent,
			ans.MarkedForReview, ans.Revisited, ans.ChangedAnswer, ans.WasInitiallyWrong, answerSeen)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to insert answer")
			return
		}
	}

	if err := h.AssignmentRepo.MarkSubmitted(assignmentID); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to mark assignment submitted")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"attempt_id":       attemptID,
		"total_time_spent": helperRoundForResponse(totalTimeSpent),
		"test_duration":    owner.Duration,
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

	detail, err := h.AssignmentRepo.GetDetailForStudent(assignmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	if detail.OwnerID != studentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "assignment does not belong to student"})
		return
	}

	existsAttempt, err := h.AttemptRepo.ExistsByAssignment(assignmentID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to check assignment status")
		return
	}
	if existsAttempt {
		c.JSON(http.StatusConflict, gin.H{"error": "assignment already submitted"})
		return
	}

	questions, _, err := h.TestRepo.ListQuestions(detail.TestID, 1000, 0)
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
	if detail.ExamDate.Valid {
		examDateStr = detail.ExamDate.Time.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, gin.H{
		"assignment_id": assignmentID,
		"test_title":    detail.TestTitle,
		"duration":      detail.Duration,
		"exam_date":     examDateStr,
		"questions":     responseQuestions,
	})
}

func helperRoundForResponse(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
