package handlers

import (
	"ai-student-diagnostic/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *AdminHandler) CreateSubject(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	role := c.GetString("role")
	if role != "admin" && role != "coach" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin or coach can create subjects"})
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	id, err := h.TestPaperRepo.CreateSubject(tenantID, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject already exists in your organization"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subject_id": id})
}

func (h *AdminHandler) ListSubjects(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	subjects, total, err := h.TestPaperRepo.ListSubjects(tenantID, search, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch subjects")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": subjects})
}
