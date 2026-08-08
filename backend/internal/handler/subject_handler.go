package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *AdminHandler) CreateSubject(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	role := c.GetString("role")
	if role != "admin" && role != "coach" {
		utils.Forbidden(c, "only admin or coach can create subjects")
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	id, err := h.TestPaperRepo.CreateSubject(tenantID, req.Name)
	if err != nil {
		utils.BadRequest(c, "subject already exists in your organization")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subject_id": id})
}

func (h *AdminHandler) ListSubjects(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	subjects, total, err := h.TestPaperRepo.ListSubjects(tenantID, search, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch subjects")
		return
	}

	if subjects == nil {
		subjects = []repository.SubjectRow{}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": subjects})
}

func (h *AdminHandler) DeleteSubject(c *gin.Context) {
	subjectID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid subject id")
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	found, err := h.TestPaperRepo.DeleteSubject(subjectID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete subject")
		return
	}
	if !found {
		utils.NotFound(c, "subject not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subject deleted"})
}
