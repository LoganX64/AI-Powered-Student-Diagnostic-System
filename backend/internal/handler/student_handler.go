package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type StudentHandler struct {
	StudentRepo    *repository.StudentRepo
	AssignmentRepo *repository.AssignmentRepo
	AttemptRepo    *repository.AttemptRepo
	TestPaperRepo  *repository.TestPaperRepo
	AttemptService *services.AttemptService
}

func NewStudentHandler(
	studentRepo *repository.StudentRepo,
	assignmentRepo *repository.AssignmentRepo,
	attemptRepo *repository.AttemptRepo,
	testPaperRepo *repository.TestPaperRepo,
	attemptService *services.AttemptService,
) *StudentHandler {
	return &StudentHandler{
		StudentRepo:    studentRepo,
		AssignmentRepo: assignmentRepo,
		AttemptRepo:    attemptRepo,
		TestPaperRepo:  testPaperRepo,
		AttemptService: attemptService,
	}
}

type StudentLoginRequest struct {
	StudentCode string `json:"student_code" binding:"required"`
}

func (h *StudentHandler) StudentLogin(c *gin.Context) {
	var req StudentLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	studentID, tenantID, err := h.StudentRepo.GetIDByStudentCode(req.StudentCode)
	if err != nil {
		utils.Unauthorized(c, "invalid student code")
		return
	}

	token, err := utils.GenerateToken(0, "student", studentID, tenantID)
	if err != nil {
		utils.InternalError(c, err, "token generation failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

type SubmitRequest struct {
	Answers []services.AnswerInput `json:"answers" binding:"required"`
}

func (h *StudentHandler) SubmitAnswers(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment_id")
		return
	}

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		if err.Error() == "unauthorized" {
			utils.Unauthorized(c, "unauthorized")
		} else {
			utils.Unauthorized(c, "invalid token data")
		}
		return
	}

	result, err := h.AttemptService.SubmitAnswers(assignmentID, studentID, req.Answers)
	if err != nil {
		var svcErr *services.SubmitAnswersError
		if errors.As(err, &svcErr) {
			c.JSON(svcErr.Status, gin.H{"error": svcErr.Message})
			return
		}
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to submit answers")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"attempt_id":       result.AttemptID,
		"total_time_spent": result.TotalTimeSpent,
		"test_duration":    result.TestDuration,
	})
}

func (h *StudentHandler) ListStudentAssignments(c *gin.Context) {
	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		if err.Error() == "unauthorized" {
			utils.Unauthorized(c, "unauthorized")
		} else {
			utils.Unauthorized(c, "invalid token data")
		}
		return
	}

	assignments, total, err := h.AssignmentRepo.ListByStudent(studentID, nil, 100, 0)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch assignments")
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
		utils.BadRequest(c, "invalid assignment_id")
		return
	}

	studentID, err := getStudentIDFromContext(c)
	if err != nil {
		if err.Error() == "unauthorized" {
			utils.Unauthorized(c, "unauthorized")
		} else {
			utils.Unauthorized(c, "invalid token data")
		}
		return
	}

	detail, err := h.AssignmentRepo.GetDetailForStudent(assignmentID)
	if err != nil {
		utils.NotFound(c, "assignment not found")
		return
	}

	if detail.OwnerID != studentID {
		utils.Forbidden(c, "assignment does not belong to student")
		return
	}

	existsAttempt, err := h.AttemptRepo.ExistsByAssignment(assignmentID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to check assignment status")
		return
	}
	if existsAttempt {
		utils.Conflict(c, "assignment already submitted")
		return
	}

	questions, _, err := h.TestPaperRepo.ListQuestions(detail.TestID, 1000, 0)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch questions")
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
