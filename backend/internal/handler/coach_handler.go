package handlers

import (
	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/internal/queue"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
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
	BatchRepo         *repository.BatchRepo
	JobRepo           *repository.JobRepo
	AttemptService    *services.AttemptService
	AssignmentService *services.AssignmentService
	JobService        *services.JobService
	Queue             queue.Queue
	Cfg               *config.Config
}

func NewCoachHandler(
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	testPaperRepo *repository.TestPaperRepo,
	assignmentRepo *repository.AssignmentRepo,
	attemptRepo *repository.AttemptRepo,
	batchRepo *repository.BatchRepo,
	jobRepo *repository.JobRepo,
	attemptService *services.AttemptService,
	assignmentService *services.AssignmentService,
	jobService *services.JobService,
	q queue.Queue,
	cfg *config.Config,
) *CoachHandler {
	return &CoachHandler{
		StudentRepo:       studentRepo,
		CoachRepo:         coachRepo,
		TestPaperRepo:     testPaperRepo,
		AssignmentRepo:    assignmentRepo,
		AttemptRepo:       attemptRepo,
		BatchRepo:         batchRepo,
		JobRepo:           jobRepo,
		AttemptService:    attemptService,
		AssignmentService: assignmentService,
		JobService:        jobService,
		Queue:             q,
		Cfg:               cfg,
	}
}

func (h *CoachHandler) GetStudentSQI(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student id")
		return
	}

	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
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
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch student SQI")
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *CoachHandler) GetAssignmentResults(c *gin.Context) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student id")
		return
	}

	assignmentID, err := strconv.Atoi(c.Param("assignmentId"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}

	coachID, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}

	exists, err := h.StudentRepo.ExistsActive(studentID, tenantID, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
		return
	}
	if !exists {
		utils.NotFound(c, "student not found")
		return
	}

	testID, status, assignedAt, testTitle, err := h.AssignmentRepo.GetByIDForCoach(assignmentID, studentID, coachID)
	if err != nil {
		utils.NotFound(c, "assignment not found")
		return
	}

	studentName, studentCode, err := h.StudentRepo.GetNameCode(studentID, tenantID)
	if err != nil {
		utils.NotFound(c, "student not found")
		return
	}

	resp, err := buildAssignmentResultsResponse(
		h.AttemptRepo, studentID, assignmentID,
		studentName, studentCode,
		testID, testTitle, status, assignedAt,
	)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch results")
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *CoachHandler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	coachID, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}

	id, code, err := ensureStudentCode(h.StudentRepo, tenantID, req.Name, req.StudentCode, coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create student")
		return
	}

	if req.BatchID != nil {
		ok, err := h.BatchRepo.Exists(tenantID, *req.BatchID)
		if err == nil && ok {
			_ = h.BatchRepo.SetStudentBatch(tenantID, id, req.BatchID)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"student_id": id, "student_code": code})
}

func (h *CoachHandler) CreateTest(c *gin.Context) {
	var req CreateTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	coachID, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}

	id, err := h.TestPaperRepo.Create(tenantID, req.Title, req.SubjectID, coachID, req.Duration, req.ExamDate, req.SubjectName)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create test")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"test_id": id})
}

func (h *CoachHandler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
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
