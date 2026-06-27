package handlers

import (
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *AdminHandler) ListCoaches(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")
	includeDeactivated := c.Query("include_deactivated") == "true"

	coaches, total, err := h.CoachRepo.List(tenantID, search, includeDeactivated, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coaches")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": coaches})
}

func (h *AdminHandler) GetCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	coach, err := h.CoachRepo.GetDetail(coachID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	c.JSON(http.StatusOK, coach)
}

func (h *AdminHandler) DeleteCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	found, err := h.CoachRepo.SoftDelete(coachID, tenantID, userID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete coach")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found or already deactivated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach account deactivated"})
}

func (h *AdminHandler) ReactivateCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	found, err := h.CoachRepo.Reactivate(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to reactivate coach")
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found or already active"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach account reactivated"})
}

func (h *AdminHandler) ListCoachTests(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	exists, err := h.CoachRepo.Exists(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	tests, total, err := h.TestPaperRepo.ListByCoach(coachID, tenantID, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coach tests")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tests})
}

func (h *AdminHandler) ListCoachStudents(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coach id"})
		return
	}

	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tenant info"})
		return
	}

	exists, err := h.CoachRepo.Exists(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify coach")
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "coach not found"})
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	students, total, err := h.StudentRepo.List(tenantID, &coachID, false, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coach students")
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": students})
}
