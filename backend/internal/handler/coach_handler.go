package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/internal/types"
	"ai-student-diagnostic/backend/utils"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CoachHandler struct {
	StudentRepo       *repository.StudentRepo
	CoachRepo         *repository.CoachRepo
	TestPaperRepo     *repository.TestPaperRepo
	AssignmentRepo    *repository.AssignmentRepo
	AttemptRepo       *repository.AttemptRepo
	AttemptService    *services.AttemptService
	AssignmentService *services.AssignmentService
}

func NewCoachHandler(
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	testPaperRepo *repository.TestPaperRepo,
	assignmentRepo *repository.AssignmentRepo,
	attemptRepo *repository.AttemptRepo,
	attemptService *services.AttemptService,
	assignmentService *services.AssignmentService,
) *CoachHandler {
	return &CoachHandler{
		StudentRepo:       studentRepo,
		CoachRepo:         coachRepo,
		TestPaperRepo:     testPaperRepo,
		AssignmentRepo:    assignmentRepo,
		AttemptRepo:       attemptRepo,
		AttemptService:    attemptService,
		AssignmentService: assignmentService,
	}
}

func (h *CoachHandler) GetStudentSQI(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	includeAnalysis := c.Query("include_analysis") == "true"
	compute := c.Query("compute") == "true"

	result, err := h.AttemptService.GetStudentSQI(services.GetStudentSQIInput{
		StudentID:       studentID,
		TenantID:        tenantID,
		IncludeAnalysis: includeAnalysis,
		Compute:         compute,
	})
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *CoachHandler) GetAssignmentResults(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student id"})
		return
	}

	assignmentID, err := strconv.Atoi(c.Param("assignmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment id"})
		return
	}

	coachID, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	exists, err := h.StudentRepo.ExistsActive(studentID, tenantID, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	testID, status, assignedAt, testTitle, err := h.AssignmentRepo.GetByIDForCoach(assignmentID, studentID, coachID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	studentName, studentCode, err := h.StudentRepo.GetNameCode(studentID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "student not found"})
		return
	}

	attemptID, submittedAt, err := h.AttemptRepo.GetByAssignment(assignmentID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"student":    gin.H{"id": studentID, "name": studentName, "student_code": studentCode},
			"test":       gin.H{"id": testID, "title": testTitle},
			"assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
			"attempt":    nil,
			"sqi_score":  nil,
			"answers":    []interface{}{},
		})
		return
	}

	sqiScore, analysisJSON, err := h.AttemptRepo.GetSQIResult(attemptID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch SQI result")
		return
	}
	answers, err := h.AttemptRepo.GetAnswerDetails(attemptID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch answer details")
		return
	}

	var analysis interface{}
	if analysisJSON.Valid && analysisJSON.String != "" {
		var payload types.DiagnosticPayloadV2
		if err := json.Unmarshal([]byte(analysisJSON.String), &payload); err == nil {
			analysis = payload
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"student":    gin.H{"id": studentID, "name": studentName, "student_code": studentCode},
		"test":       gin.H{"id": testID, "title": testTitle},
		"assignment": gin.H{"id": assignmentID, "status": status, "assigned_at": assignedAt},
		"attempt":    gin.H{"id": attemptID, "submitted_at": submittedAt.Time},
		"sqi_score":  sqiScore.Float64,
		"analysis":   analysis,
		"answers":    answers,
	})
}

func (h *CoachHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	coachID, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	id, err := h.StudentRepo.Create(tenantID, req.Name, req.StudentCode, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create student")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"student_id": id})
}

func (h *CoachHandler) CreateTest(c *gin.Context) {
	var req CreateTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	coachID, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
		return
	}

	id, err := h.TestPaperRepo.Create(tenantID, req.Title, req.SubjectID, coachID, req.Duration, req.ExamDate)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create test")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"test_id": id})
}

func (h *CoachHandler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	userID := c.GetInt("user_id")

	id, err := h.AssignmentService.CreateAssignment(services.CreateAssignmentInput{
		CallerRole: "coach",
		CallerID:   userID,
		StudentID:  req.StudentID,
		TestID:     req.TestID,
	})
	if err != nil {
		var svcErr *services.CreateAssignmentError
		if errors.As(err, &svcErr) {
			c.JSON(svcErr.Status, gin.H{"error": svcErr.Message})
			return
		}
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create assignment")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"assignment_id": id})
}
