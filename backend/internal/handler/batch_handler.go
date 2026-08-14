package handlers

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────
// Shared batch helpers (used by admin + coach)
// ─────────────────────────────────────────────

func createBatchHelper(c *gin.Context, tenantID int, batchRepo *repository.BatchRepo) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	id, err := batchRepo.Create(tenantID, req.Name)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to create batch")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"batch_id": id})
}

func listBatchesHelper(c *gin.Context, tenantID int, batchRepo *repository.BatchRepo) {
	batches, err := batchRepo.List(tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to list batches")
		return
	}
	if batches == nil {
		batches = []repository.BatchRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": batches})
}

func deleteBatchHelper(c *gin.Context, tenantID int, batchRepo *repository.BatchRepo) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid batch id")
		return
	}
	exists, err := batchRepo.Exists(tenantID, id)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify batch")
		return
	}
	if !exists {
		utils.NotFound(c, "batch not found")
		return
	}
	reassigned, err := batchRepo.Delete(tenantID, id)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to delete batch")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "batch deleted", "students_reassigned": reassigned})
}

func transferStudentBatchHelper(c *gin.Context, tenantID int, studentRepo *repository.StudentRepo, batchRepo *repository.BatchRepo) {
	studentID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid student id")
		return
	}

	var req struct {
		BatchID *int `json:"batch_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}

	exists, err := studentRepo.Exists(studentID, tenantID)
	if err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify student")
		return
	}
	if !exists {
		utils.NotFound(c, "student not found")
		return
	}

	if req.BatchID != nil {
		ok, err := batchRepo.Exists(tenantID, *req.BatchID)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify batch")
			return
		}
		if !ok {
			utils.BadRequest(c, "invalid batch_id for your organization")
			return
		}
	}

	if err := batchRepo.SetStudentBatch(tenantID, studentID, req.BatchID); err != nil {
		utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to transfer student")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "student batch updated"})
}

// ─────────────────────────────────────────────
// AdminHandler batch methods
// ─────────────────────────────────────────────

func (h *AdminHandler) CreateBatch(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}
	createBatchHelper(c, tenantID, h.BatchRepo)
}

func (h *AdminHandler) ListBatches(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}
	listBatchesHelper(c, tenantID, h.BatchRepo)
}

func (h *AdminHandler) DeleteBatch(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}
	deleteBatchHelper(c, tenantID, h.BatchRepo)
}

func (h *AdminHandler) TransferStudentBatch(c *gin.Context) {
	tenantID, err := resolveTenantID(c, h.UserRepo)
	if err != nil {
		utils.InternalError(c, err, "failed to fetch tenant info")
		return
	}
	transferStudentBatchHelper(c, tenantID, h.StudentRepo, h.BatchRepo)
}

// ─────────────────────────────────────────────
// CoachHandler batch methods
// ─────────────────────────────────────────────

func (h *CoachHandler) CreateBatch(c *gin.Context) {
	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}
	createBatchHelper(c, tenantID, h.BatchRepo)
}

func (h *CoachHandler) ListBatches(c *gin.Context) {
	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}
	listBatchesHelper(c, tenantID, h.BatchRepo)
}

func (h *CoachHandler) DeleteBatch(c *gin.Context) {
	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}
	deleteBatchHelper(c, tenantID, h.BatchRepo)
}

func (h *CoachHandler) TransferStudentBatch(c *gin.Context) {
	_, tenantID, err := resolveCoachAndTenant(c, h.CoachRepo)
	if err != nil {
		utils.Unauthorized(c, "coach not found")
		return
	}
	transferStudentBatchHelper(c, tenantID, h.StudentRepo, h.BatchRepo)
}

// ─────────────────────────────────────────────
// Shared helpers for batch assignment expansion
// ─────────────────────────────────────────────

func (h *AdminHandler) scaleBandC() int {
	if h.Cfg != nil {
		return h.Cfg.ScaleBandC
	}
	return 0
}

func (h *CoachHandler) scaleBandC() int {
	if h.Cfg != nil {
		return h.Cfg.ScaleBandC
	}
	return 0
}

// expandBatchTargets merges explicit student_ids with the members of batch_ids,
// deduplicating and validating each batch belongs to the tenant.
func (h *AdminHandler) expandBatchTargets(c *gin.Context, tenantID int, studentIDs, batchIDs []int) ([]int, error) {
	return expandBatchTargets(c, tenantID, studentIDs, batchIDs, h.BatchRepo)
}

func (h *CoachHandler) expandBatchTargets(c *gin.Context, tenantID int, studentIDs, batchIDs []int) ([]int, error) {
	return expandBatchTargets(c, tenantID, studentIDs, batchIDs, h.BatchRepo)
}

func expandBatchTargets(c *gin.Context, tenantID int, studentIDs, batchIDs []int, batchRepo *repository.BatchRepo) ([]int, error) {
	seen := make(map[int]bool)
	out := make([]int, 0, len(studentIDs))

	add := func(id int) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}

	for _, id := range studentIDs {
		add(id)
	}

	for _, bid := range batchIDs {
		exists, err := batchRepo.Exists(tenantID, bid)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to verify batch")
			return nil, err
		}
		if !exists {
			utils.BadRequest(c, "invalid batch_ids: one or more not found in your organization")
			return nil, err
		}
		members, err := batchRepo.MemberIDs(tenantID, bid)
		if err != nil {
			utils.SafeErrorResponse(c, http.StatusInternalServerError, err, "failed to expand batch")
			return nil, err
		}
		for _, m := range members {
			add(m)
		}
	}

	if len(out) == 0 {
		utils.BadRequest(c, "no students or batches provided")
		return nil, errEmptyTargets
	}
	return out, nil
}

var errEmptyTargets = &emptyTargetsError{}

type emptyTargetsError struct{}

func (e *emptyTargetsError) Error() string { return "no targets" }
