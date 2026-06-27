package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/utils"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CreateAssignmentRequest struct {
	StudentID int `json:"student_id" binding:"required"`
	TestID    int `json:"test_id" binding:"required"`
	CoachID   int `json:"coach_id" binding:"required"`
}

func (h *AdminHandler) CreateAssignment(c *gin.Context) {
	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	userID := c.GetInt("user_id")

	id, err := h.AssignmentService.CreateAssignment(services.CreateAssignmentInput{
		CallerRole: role,
		CallerID:   userID,
		StudentID:  req.StudentID,
		TestID:     req.TestID,
		CoachID:    req.CoachID,
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

func (h *AdminHandler) ListAssignments(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	role := c.GetString("role")
	var coachID *int
	if role == "coach" {
		cid, err := resolveCoachID(c, h.CoachRepo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "coach not found"})
			return
		}
		coachID = &cid
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	testID := c.Query("test_id")

	assignments, total, err := h.AssignmentRepo.ListAll(tenantID, coachID, testID, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch assignments")
		return
	}

	if assignments == nil {
		assignments = []repository.AssignmentDetailRow{}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": assignments})
}
