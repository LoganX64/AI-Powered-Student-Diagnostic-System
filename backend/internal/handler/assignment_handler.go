package handlers

import (
	"ai-student-diagnostic/backend/internal/middleware"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateAssignmentRequest struct {
	StudentID       int             `json:"student_id" binding:"required"`
	TestID          int             `json:"test_id" binding:"required"`
	CoachID         int             `json:"coach_id" binding:"required"`
	IntegrityPolicy json.RawMessage `json:"integrity_policy"`
	EstimatedCost   float64         `json:"estimated_cost"`
	DeliveryMode    string          `json:"delivery_mode"`
}

func (h *AdminHandler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	deliveryMode := services.DeliveryModeForN(1, h.scaleBandC())

	id, err := h.AssignmentService.CreateAssignment(services.CreateAssignmentInput{
		CallerRole:      role,
		CallerID:        userID,
		TenantID:        c.GetInt("tenant_id"),
		StudentID:       req.StudentID,
		TestID:          req.TestID,
		CoachID:         req.CoachID,
		IntegrityPolicy: req.IntegrityPolicy,
		EstimatedCost:   req.EstimatedCost,
		DeliveryMode:    deliveryMode,
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

// CreateBatchAssignmentRequest assigns one test to many students and/or batches.
type CreateBatchAssignmentRequest struct {
	TestID          int             `json:"test_id" binding:"required"`
	StudentIDs      []int           `json:"student_ids"`
	BatchIDs        []int           `json:"batch_ids"`
	CoachID         int             `json:"coach_id"`
	IntegrityPolicy json.RawMessage `json:"integrity_policy"`
	EstimatedCost   float64         `json:"estimated_cost"`
	DeliveryMode    string          `json:"delivery_mode"`
}

func (h *AdminHandler) CreateBatchAssignment(c *gin.Context) {
	var req CreateBatchAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	role := c.GetString("role")
	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	studentIDs, err := h.expandBatchTargets(c, tenantID, req.StudentIDs, req.BatchIDs)
	if err != nil {
		return
	}

	deliveryMode := req.DeliveryMode
	if deliveryMode == "" {
		deliveryMode = services.DeliveryModeForN(len(studentIDs), h.scaleBandC())
	}

	created, err := h.AssignmentService.CreateBatchAssignment(services.CreateBatchAssignmentInput{
		CallerRole:      role,
		CallerID:        userID,
		TenantID:        tenantID,
		TestID:          req.TestID,
		CoachID:         req.CoachID,
		StudentIDs:      studentIDs,
		IntegrityPolicy: req.IntegrityPolicy,
		EstimatedCost:   req.EstimatedCost,
		DeliveryMode:    deliveryMode,
	})
	if err != nil {
		var svcErr *services.CreateAssignmentError
		if errors.As(err, &svcErr) {
			c.JSON(svcErr.Status, gin.H{"error": svcErr.Message})
			return
		}
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create assignments")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"created": created})
}

func (h *CoachHandler) CreateBatchAssignment(c *gin.Context) {
	var req CreateBatchAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}

	studentIDs, err := h.expandBatchTargets(c, tenantID, req.StudentIDs, req.BatchIDs)
	if err != nil {
		return
	}

	deliveryMode := req.DeliveryMode
	if deliveryMode == "" {
		deliveryMode = services.DeliveryModeForN(len(studentIDs), h.scaleBandC())
	}

	created, err := h.AssignmentService.CreateBatchAssignment(services.CreateBatchAssignmentInput{
		CallerRole:      "coach",
		CallerID:        c.GetInt("user_id"),
		TenantID:        tenantID,
		TestID:          req.TestID,
		CoachID:         0,
		StudentIDs:      studentIDs,
		IntegrityPolicy: req.IntegrityPolicy,
		EstimatedCost:   req.EstimatedCost,
		DeliveryMode:    deliveryMode,
	})
	if err != nil {
		var svcErr *services.CreateAssignmentError
		if errors.As(err, &svcErr) {
			c.JSON(svcErr.Status, gin.H{"error": svcErr.Message})
			return
		}
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create assignments")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"created": created})
}

func (h *AdminHandler) ListAssignments(c *gin.Context) {
	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	role := c.GetString("role")
	var coachID *int
	if role == "coach" {
		cid, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			utils.Unauthorized(c, "coach not found")
			return
		}
		coachID = &cid
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	testID := c.Query("test_id")
	status := c.Query("status")
	search := c.Query("search")
	year := c.Query("year")
	subjectID := c.Query("subject_id")
	coachIDStr := c.Query("coach_id")

	filters := repository.AssignmentFilters{
		Status:     status,
		TestID:     testID,
		Search:     search,
		Year:       year,
		SubjectID:  subjectID,
		CoachIDStr: coachIDStr,
	}

	assignments, total, err := h.AssignmentRepo.ListAll(tenantID, coachID, filters, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch assignments")
		return
	}

	if assignments == nil {
		assignments = []repository.AssignmentDetailRow{}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": assignments})
}

func (h *AdminHandler) DeleteAssignment(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	removed, err := h.AssignmentRepo.Delete(assignmentID, tenantID, nil)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete assignment")
		return
	}
	if !removed {
		utils.NotFound(c, "assignment not found")
		return
	}

	// Release any metered proctoring storage for this assignment.
	releaseAssignmentStorage(h.SubscriptionRepo, h.QuotaMW, tenantID, assignmentID)

	c.JSON(http.StatusOK, gin.H{"message": "assignment cancelled"})
}

func (h *CoachHandler) DeleteAssignment(c *gin.Context) {
	assignmentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid assignment id")
		return
	}

	coachID, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}

	removed, err := h.AssignmentRepo.Delete(assignmentID, tenantID, &coachID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete assignment")
		return
	}
	if !removed {
		utils.NotFound(c, "assignment not found")
		return
	}

	// Release any metered proctoring storage for this assignment.
	releaseAssignmentStorage(h.SubscriptionRepo, h.QuotaMW, tenantID, assignmentID)

	c.JSON(http.StatusOK, gin.H{"message": "assignment cancelled"})
}

// releaseAssignmentStorage frees metered storage and invalidates the quota cache.
// Both operations are best-effort; a failure here must not fail the delete.
func releaseAssignmentStorage(subRepo *repository.SubscriptionRepo, quotaMW *middleware.QuotaMiddleware, tenantID, assignmentID int) {
	if subRepo == nil {
		return
	}
	if err := subRepo.ReleaseStorageForAssignment(tenantID, assignmentID); err != nil {
		log.Printf("[ASSIGNMENT] failed to release storage for assignment %d: %v", assignmentID, err)
	}
	if quotaMW != nil {
		quotaMW.Invalidate(tenantID)
	}
}
