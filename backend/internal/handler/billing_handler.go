package handlers

import (
	"ai-student-diagnostic/backend/internal/middleware"
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/storage"
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BillingHandler struct {
	PlanRepo         *repository.PlanRepo
	SubscriptionRepo *repository.SubscriptionRepo
	StudentRepo      *repository.StudentRepo
	CoachRepo        *repository.CoachRepo
	Storage          storage.Storage
	// QuotaMW is optional (nil in tests). Guarded before every use.
	QuotaMW *middleware.QuotaMiddleware
}

func NewBillingHandler(
	planRepo *repository.PlanRepo,
	subscriptionRepo *repository.SubscriptionRepo,
	studentRepo *repository.StudentRepo,
	coachRepo *repository.CoachRepo,
	storageBackend storage.Storage,
	quotaMW *middleware.QuotaMiddleware,
) *BillingHandler {
	return &BillingHandler{
		PlanRepo:         planRepo,
		SubscriptionRepo: subscriptionRepo,
		StudentRepo:      studentRepo,
		CoachRepo:        coachRepo,
		Storage:          storageBackend,
		QuotaMW:          quotaMW,
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
	c.JSON(http.StatusOK, gin.H{"message": "plan assigned"})
}

// GET /admin/subscription
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	sub, err := h.SubscriptionRepo.GetByTenantID(tenantID)
	if err != nil {
		utils.NotFound(c, "no subscription found")
		return
	}
	c.JSON(http.StatusOK, sub)
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
	// Mock Razorpay response: immediately assign the plan locally.
	tenantID := c.GetInt("tenant_id")
	if err := h.SubscriptionRepo.Upsert(tenantID, plan.ID); err != nil {
		utils.InternalError(c, err, "failed to start checkout")
		return
	}
	if h.QuotaMW != nil {
		h.QuotaMW.Invalidate(tenantID)
	}
	c.JSON(http.StatusOK, gin.H{
		"checkout_url":    "https://mock-razorpay.com/checkout/" + req.PlanSlug,
		"subscription_id": "mock_sub_" + strconv.Itoa(tenantID),
	})
}

// POST /admin/subscription/webhook (MOCK)
func (h *BillingHandler) HandleWebhook(c *gin.Context) {
	// Mock webhook handler
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// When the payment service is registered, replace the mock above with:
//   1. Webhook is the SOURCE OF TRUTH; the client checkout callback is only UX.
//   2. Verify `X-Razorpay-Signature` = HMAC-SHA256(rawBody, webhookSecret).
//      CRITICAL: hash the RAW request body bytes — do NOT parse+re-stringify JSON
//      (whitespace/key-order changes break the signature; #1 integration failure).
//   3. Idempotency: dedupe by `event_id` inside the SAME DB transaction as the
//      status update, so retries never double-apply.
//   4. Acknowledge 2xx FAST, then process the event async (queue/worker).
//   5. Use the official `razorpay-go` SDK. Amounts are in PAISE (see ISSUE-6):
//      store/compare as integers, never floats.
//   6. Reconcile periodically against the Razorpay API as a safety net.

// POST /admin/subscription/cancel
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	// Revert to the Free plan so quota checks immediately downgrade limits.
	// Cancelling must restrict access, not leave paid limits in place.
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
