package handlers

import (
	"ai-student-diagnostic/backend/internal/middleware"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/internal/storage"
	"ai-student-diagnostic/backend/utils"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// storageOverageRatePerGBPaise bills ₹50/GB (5000 paise) for video proctoring
// storage used beyond the plan cap (TEMPORARY model, re-evaluate on review).
const (
	storageOverageRatePerGBPaise = 5000
	bytesPerGB                   = 1024 * 1024 * 1024
)

// overageCostPaise converts overage bytes into the per-GB overage charge (paise).
func overageCostPaise(overageBytes int64) int64 {
	if overageBytes <= 0 {
		return 0
	}
	gb := float64(overageBytes) / float64(bytesPerGB)
	return int64(math.Ceil(gb)) * storageOverageRatePerGBPaise
}

type BillingHandler struct {
	PlanRepo            *repository.PlanRepo
	SubscriptionRepo    *repository.SubscriptionRepo
	StudentRepo         *repository.StudentRepo
	CoachRepo           *repository.CoachRepo
	TenantRepo          *repository.TenantRepo
	Storage             storage.Storage
	QuotaMW             *middleware.QuotaMiddleware
	NotificationService *services.NotificationService
}

func NewBillingHandler(
	planRepo *repository.PlanRepo,
	subscriptionRepo *repository.SubscriptionRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	tenantRepo *repository.TenantRepo,
	storageBackend storage.Storage,
	quotaMW *middleware.QuotaMiddleware,
	notifService *services.NotificationService,
) *BillingHandler {
	return &BillingHandler{
		PlanRepo:            planRepo,
		SubscriptionRepo:    subscriptionRepo,
		StudentRepo:         studentRepo,
		CoachRepo:           coachRepo,
		TenantRepo:          tenantRepo,
		Storage:             storageBackend,
		QuotaMW:             quotaMW,
		NotificationService: notifService,
	}
}

// GET /super-admin/plans
func (h *BillingHandler) ListPlans(c *gin.Context) {
	plans, err := h.PlanRepo.List()
	if err != nil {
		utils.InternalError(c, err, "failed to fetch plans")
		return
	}
	if plans == nil {
		plans = []repository.PlanRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": plans})
}

// POST /super-admin/plans
func (h *BillingHandler) CreatePlan(c *gin.Context) {
	var req repository.PlanRow
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	id, err := h.PlanRepo.Create(req)
	if err != nil {
		utils.InternalError(c, err, "failed to create plan")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"plan_id": id})
}

// PUT /super-admin/plans/:id
func (h *BillingHandler) UpdatePlan(c *gin.Context) {
	planID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid plan id")
		return
	}
	var req repository.PlanRow
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	req.ID = planID
	if err := h.PlanRepo.Update(req); err != nil {
		utils.InternalError(c, err, "failed to update plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "plan updated"})
}

// DELETE /super-admin/plans/:id
func (h *BillingHandler) DeletePlan(c *gin.Context) {
	planID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid plan id")
		return
	}
	if err := h.PlanRepo.Delete(planID); err != nil {
		utils.InternalError(c, err, "failed to delete plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "plan deleted"})
}

// PUT /super-admin/tenants/:id/subscription { plan_id }
func (h *BillingHandler) AssignPlan(c *gin.Context) {
	tenantID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "invalid tenant id")
		return
	}
	var req struct {
		PlanID int `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	if err := h.SubscriptionRepo.Upsert(tenantID, req.PlanID); err != nil {
		utils.InternalError(c, err, "failed to assign plan")
		return
	}
	if h.QuotaMW != nil {
		h.QuotaMW.Invalidate(tenantID)
	}
	h.notifyPlanChanged(tenantID, req.PlanID)
	c.JSON(http.StatusOK, gin.H{"message": "plan assigned"})
}

// notifyPlanChanged tells the platform inbox that a tenant's plan moved. The
// names are cosmetic, so a failed lookup falls back to the stored ids.
func (h *BillingHandler) notifyPlanChanged(tenantID, planID int) {
	if h.NotificationService == nil {
		return
	}
	tenantName := ""
	if h.TenantRepo != nil {
		if tenant, err := h.TenantRepo.GetByID(tenantID); err != nil {
			log.Printf("[NOTIFICATION] tenant lookup failed for plan change %d: %v", tenantID, err)
		} else if tenant != nil {
			tenantName = tenant.Name
		}
	}
	planName := ""
	if h.PlanRepo != nil {
		if plan, err := h.PlanRepo.GetByID(planID); err != nil {
			log.Printf("[NOTIFICATION] plan lookup failed for plan change %d: %v", planID, err)
		} else if plan != nil {
			planName = plan.Name
		}
	}
	if err := h.NotificationService.NotifyPlanChanged(tenantID, tenantName, planID, planName); err != nil {
		log.Printf("[NOTIFICATION] plan changed notify failed for tenant %d: %v", tenantID, err)
	}
}

// GET /admin/subscription
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	sub, err := h.SubscriptionRepo.GetByTenantID(tenantID)
	if err != nil {
		utils.NotFound(c, "no subscription found")
		return
	}
	c.JSON(http.StatusOK, struct {
		repository.SubscriptionRow
		AdditionalCostPaise int64 `json:"additional_cost_paise"`
	}{
		SubscriptionRow:     *sub,
		AdditionalCostPaise: overageCostPaise(sub.OverageBytes),
	})
}

// POST /admin/subscription/checkout { plan_slug } (MOCK)
func (h *BillingHandler) CreateCheckout(c *gin.Context) {
	var req struct {
		PlanSlug string `json:"plan_slug" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid payload")
		return
	}
	plan, err := h.PlanRepo.GetBySlug(req.PlanSlug)
	if err != nil {
		utils.NotFound(c, "plan not found")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if err := h.SubscriptionRepo.Upsert(tenantID, plan.ID); err != nil {
		utils.InternalError(c, err, "failed to start checkout")
		return
	}
	if h.QuotaMW != nil {
		h.QuotaMW.Invalidate(tenantID)
	}

	var overage int64
	if cur, cerr := h.SubscriptionRepo.GetByTenantID(tenantID); cerr == nil {
		overage = cur.OverageBytes
	}
	c.JSON(http.StatusOK, gin.H{
		"checkout_url":          "https://mock-razorpay.com/checkout/" + req.PlanSlug,
		"subscription_id":       "mock_sub_" + strconv.Itoa(tenantID),
		"overage_bytes":         overage,
		"additional_cost_paise": overageCostPaise(overage),
	})
}

// POST /admin/subscription/webhook (MOCK)
func (h *BillingHandler) HandleWebhook(c *gin.Context) {
	// Mock webhook handler
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /admin/subscription/cancel
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	if h.PlanRepo != nil {
		if free, ferr := h.PlanRepo.GetBySlug("free"); ferr == nil && free != nil {
			if uerr := h.SubscriptionRepo.Upsert(tenantID, free.ID); uerr != nil {
				utils.InternalError(c, uerr, "failed to revert plan on cancel")
				return
			}
		}
	}

	if err := h.SubscriptionRepo.UpdateStatus(tenantID, "cancelled"); err != nil {
		utils.InternalError(c, err, "failed to cancel subscription")
		return
	}
	if h.QuotaMW != nil {
		h.QuotaMW.Invalidate(tenantID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "subscription cancelled"})
}
