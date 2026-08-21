package handlers

import (
	"ai-student-diagnostic/backend/internal/helper"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *AdminHandler) ListCoaches(c *gin.Context) {
	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
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
		utils.BadRequest(c, "invalid coach id")
		return
	}

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	coach, err := h.CoachRepo.GetDetail(coachID, tenantID)
	if err != nil {
		utils.NotFound(c, "coach not found")
		return
	}

	c.JSON(http.StatusOK, coach)
}

func (h *AdminHandler) DeleteCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid coach id")
		return
	}

	userID := c.GetInt("user_id")
	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	found, err := h.CoachRepo.SoftDelete(coachID, tenantID, userID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete coach")
		return
	}
	if !found {
		utils.NotFound(c, "coach not found or already deactivated")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach account deactivated"})
}

func (h *AdminHandler) ReactivateCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid coach id")
		return
	}

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	found, err := h.CoachRepo.Reactivate(coachID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to reactivate coach")
		return
	}
	if !found {
		utils.NotFound(c, "coach not found or already active")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach account reactivated"})
}

type UpdateCoachRequest struct {
	Name       string `json:"name" binding:"required"`
	Email      string `json:"email" binding:"required"`
	SubjectIDs []int  `json:"subject_ids" binding:"required"`
}

func (h *AdminHandler) UpdateCoach(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid coach id")
		return
	}

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	var req UpdateCoachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	coach, err := h.CoachRepo.GetDetail(coachID, tenantID)
	if err != nil {
		utils.NotFound(c, "coach not found")
		return
	}

	emailTaken, err := h.UserRepo.EmailExistsForOther(req.Email, coach.UserID)
	if err != nil {
		utils.InternalError(c, err, "failed to verify email")
		return
	}
	if emailTaken {
		utils.BadRequest(c, "email is already in use by another account")
		return
	}

	tx, err := h.CoachRepo.DB.Begin()
	if err != nil {
		utils.InternalError(c, err, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	if err := h.CoachRepo.UpdateName(tx, coachID, tenantID, req.Name); err != nil {
		utils.InternalError(c, err, "failed to update coach name")
		return
	}

	if err := h.UserRepo.UpdateEmail(tx, coach.UserID, req.Email); err != nil {
		utils.InternalError(c, err, "failed to update coach email")
		return
	}

	if err := h.CoachRepo.DeleteCoachSubjects(tx, coachID); err != nil {
		utils.InternalError(c, err, "failed to update coach subjects")
		return
	}

	if len(req.SubjectIDs) > 0 {
		if err := h.CoachRepo.CreateCoachSubjectsInTx(tx, coachID, req.SubjectIDs); err != nil {
			utils.InternalError(c, err, "failed to update coach subjects")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		utils.InternalError(c, err, "failed to save changes")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "coach updated"})
}

func (h *AdminHandler) ListCoachTests(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid coach id")
		return
	}

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	if !verifyCoachExists(c, coachID, tenantID, h.CoachRepo) {
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	tests, total, err := h.TestPaperRepo.ListByCoach(coachID, tenantID, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coach tests")
		return
	}

	if tests == nil {
		tests = []repository.TestRow{}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": tests})
}

func (h *AdminHandler) ListCoachStudents(c *gin.Context) {
	coachID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid coach id")
		return
	}

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	if !verifyCoachExists(c, coachID, tenantID, h.CoachRepo) {
		return
	}

	limit, offset := utils.ParsePagination(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")
	students, total, err := h.StudentRepo.List(tenantID, &coachID, false, search, nil, limit, offset)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coach students")
		return
	}

	if students == nil {
		students = []repository.StudentRow{}
	}

	c.JSON(http.StatusOK, gin.H{"total": total, "limit": limit, "offset": offset, "data": students})
}

type CoachStatsBatchRequest struct {
	CoachIDs []int `json:"coach_ids" binding:"required"`
}

const maxBatchCoachIDs = 500

func (h *AdminHandler) GetCoachStatsBatch(c *gin.Context) {
	role := c.GetString("role")

	if role == "super_admin" {
		utils.Forbidden(c, "super-admin has no access to coach stats")
		return
	}

	var req CoachStatsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if len(req.CoachIDs) == 0 {
		utils.BadRequest(c, "coach_ids must not be empty")
		return
	}
	if len(req.CoachIDs) > maxBatchCoachIDs {
		utils.BadRequest(c, "too many coach_ids")
		return
	}

	tenantID, err := resolveTenantID(c)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}

	result, err := h.CoachRepo.GetCoachStats(req.CoachIDs, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch coach stats")
		return
	}

	if result == nil {
		result = []repository.CoachStatMetric{}
	}
	for i := range result {
		result[i].AverageSQI = helper.Round2V2(result[i].AverageSQI)
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
