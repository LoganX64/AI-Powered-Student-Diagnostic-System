package handlers

import (
	"ai-student-diagnostic/backend/internal/queue"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type sqiJobPayload struct {
	AttemptIDs []int `json:"attempt_ids"`
}

// enqueueComputeJob creates a compute_sqi job and enqueues it for the worker.
func enqueueComputeJob(c *gin.Context, tenantID int, attemptIDs []int, jobRepo *repository.JobRepo, q queue.Queue) {
	if len(attemptIDs) == 0 {
		utils.BadRequest(c, "no attempts to compute")
		return
	}
	payload, err := json.Marshal(sqiJobPayload{AttemptIDs: attemptIDs})
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to encode job")
		return
	}
	jobID, err := jobRepo.Create(tenantID, "compute_sqi", payload, len(attemptIDs))
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create job")
		return
	}
	if err := q.EnqueueCompute(jobID, tenantID); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to enqueue compute job")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "total": len(attemptIDs)})
}

// ─────────────────────────────────────────────
// Admin
// ─────────────────────────────────────────────

func (h *AdminHandler) ComputeSQI(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}
	var req struct {
		AttemptID int `json:"attempt_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	ok, err := h.AttemptRepo.AttemptBelongsToTenant(req.AttemptID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify attempt")
		return
	}
	if !ok {
		utils.NotFound(c, "attempt not found")
		return
	}
	enqueueComputeJob(c, tenantID, []int{req.AttemptID}, h.JobRepo, h.Queue)
}

func (h *AdminHandler) ComputeSQIBatch(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}
	var req struct {
		AttemptIDs []int `json:"attempt_ids"`
		TestID     int   `json:"test_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	var attemptIDs []int
	if req.TestID > 0 {
		ids, err := h.AttemptRepo.AttemptIDsByTest(req.TestID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch attempts")
			return
		}
		attemptIDs = ids
	} else {
		for _, id := range req.AttemptIDs {
			ok, err := h.AttemptRepo.AttemptBelongsToTenant(id, tenantID)
			if err != nil {
				utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify attempt")
				return
			}
			if ok {
				attemptIDs = append(attemptIDs, id)
			}
		}
	}
	enqueueComputeJob(c, tenantID, attemptIDs, h.JobRepo, h.Queue)
}

func (h *AdminHandler) GetJob(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}
	jobID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	job, err := h.JobRepo.Get(jobID, tenantID)
	if err != nil {
		utils.NotFound(c, "job not found")
		return
	}
	c.JSON(http.StatusOK, job)
}

// ─────────────────────────────────────────────
// Coach
// ─────────────────────────────────────────────

func (h *CoachHandler) ComputeSQI(c *gin.Context) {
	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}
	var req struct {
		AttemptID int `json:"attempt_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	ok, err := h.AttemptRepo.AttemptBelongsToTenant(req.AttemptID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify attempt")
		return
	}
	if !ok {
		utils.NotFound(c, "attempt not found")
		return
	}
	enqueueComputeJob(c, tenantID, []int{req.AttemptID}, h.JobRepo, h.Queue)
}

func (h *CoachHandler) ComputeSQIBatch(c *gin.Context) {
	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}
	var req struct {
		AttemptIDs []int `json:"attempt_ids"`
		TestID     int   `json:"test_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	var attemptIDs []int
	if req.TestID > 0 {
		ids, err := h.AttemptRepo.AttemptIDsByTest(req.TestID, tenantID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to fetch attempts")
			return
		}
		attemptIDs = ids
	} else {
		for _, id := range req.AttemptIDs {
			ok, err := h.AttemptRepo.AttemptBelongsToTenant(id, tenantID)
			if err != nil {
				utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify attempt")
				return
			}
			if ok {
				attemptIDs = append(attemptIDs, id)
			}
		}
	}
	enqueueComputeJob(c, tenantID, attemptIDs, h.JobRepo, h.Queue)
}

func (h *CoachHandler) GetJob(c *gin.Context) {
	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}
	jobID, err := parseIDParam(c, "id")
	if err != nil {
		return
	}
	job, err := h.JobRepo.Get(jobID, tenantID)
	if err != nil {
		utils.NotFound(c, "job not found")
		return
	}
	c.JSON(http.StatusOK, job)
}
