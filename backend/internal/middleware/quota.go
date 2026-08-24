package middleware

import (
	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type QuotaMiddleware struct {
	SubscriptionRepo *repository.SubscriptionRepo
	PlanRepo         *repository.PlanRepo
	// cache fields
	cacheMu  sync.RWMutex
	cache    map[int]subCacheEntry
	cacheTTL time.Duration

	freePlan   *repository.PlanRow
	freePlanMu sync.Mutex
}

func NewQuotaMiddleware(subRepo *repository.SubscriptionRepo, planRepo *repository.PlanRepo) *QuotaMiddleware {
	return &QuotaMiddleware{
		SubscriptionRepo: subRepo,
		PlanRepo:         planRepo,
		cache:            make(map[int]subCacheEntry),
		cacheTTL:         5 * time.Minute,
	}
}

// loadFreePlan returns the cached Free plan, loading it on first use. If the
// Free plan cannot be fetched at all, it returns a hard-coded conservative
// fallback (no premium access, zero caps) so we never grant unlimited access.
func (q *QuotaMiddleware) loadFreePlan() *repository.PlanRow {
	q.freePlanMu.Lock()
	defer q.freePlanMu.Unlock()
	if q.freePlan != nil {
		return q.freePlan
	}
	if q.PlanRepo != nil {
		if p, err := q.PlanRepo.GetBySlug("free"); err == nil && p != nil {
			q.freePlan = p
			return q.freePlan
		}
	}
	return &repository.PlanRow{
		Name:                    "free",
		StudentLimit:            0,
		CoachLimit:              0,
		StorageLimitBytes:       0,
		TestLimit:               0,
		SQIAccess:               false,
		VideoProctoringIncluded: false,
		VideoProctoringLimit:    0,
	}
}

type subCacheEntry struct {
	sub     *repository.SubscriptionRow
	expires time.Time
}

func (q *QuotaMiddleware) subFor(c *gin.Context) (*repository.SubscriptionRow, error) {
	// already loaded this request?
	if v, ok := c.Get("quota_sub"); ok {
		return v.(*repository.SubscriptionRow), nil
	}
	tenantID := c.GetInt("tenant_id")
	// process cache
	q.cacheMu.RLock()
	if e, ok := q.cache[tenantID]; ok && time.Now().Before(e.expires) {
		q.cacheMu.RUnlock()
		c.Set("quota_sub", e.sub)
		return e.sub, nil
	}
	q.cacheMu.RUnlock()
	sub, err := q.SubscriptionRepo.GetByTenantID(tenantID)
	if err != nil {
		// No subscription row: default to Free-plan restrictions instead of
		// failing open (unlimited). Build a synthetic Free row from cached
		// Free-plan defaults; usage counts default to 0.
		free := q.loadFreePlan()
		fallback := &repository.SubscriptionRow{
			TenantID:                tenantID,
			PlanName:                free.Name,
			Status:                  "free_default",
			StudentLimit:            free.StudentLimit,
			CoachLimit:              free.CoachLimit,
			StorageLimitBytes:       free.StorageLimitBytes,
			TestLimit:               free.TestLimit,
			SQIAccess:               free.SQIAccess,
			VideoProctoringIncluded: free.VideoProctoringIncluded,
			VideoProctoringLimit:    free.VideoProctoringLimit,
		}
		q.cacheMu.Lock()
		q.cache[tenantID] = subCacheEntry{sub: fallback, expires: time.Now().Add(q.cacheTTL)}
		q.cacheMu.Unlock()
		c.Set("quota_sub", fallback)
		return fallback, nil
	}
	q.cacheMu.Lock()
	q.cache[tenantID] = subCacheEntry{sub: sub, expires: time.Now().Add(q.cacheTTL)}
	q.cacheMu.Unlock()
	c.Set("quota_sub", sub)
	return sub, nil
}

// Invalidate drops the cached subscription for a tenant so a plan/usage change
// takes effect immediately (instead of waiting for cacheTTL to expire).
func (q *QuotaMiddleware) Invalidate(tenantID int) {
	q.cacheMu.Lock()
	delete(q.cache, tenantID)
	q.cacheMu.Unlock()
}

func (q *QuotaMiddleware) CheckStudentLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := q.subFor(c) // cached, one DB hit per request
		if err != nil {
			c.Next()
			return
		}
		if sub.StudentLimit > 0 && sub.StudentCount >= sub.StudentLimit {
			utils.SafeErrorResponse(c, http.StatusPaymentRequired, nil, "student limit reached for your plan")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (q *QuotaMiddleware) CheckCoachLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := q.subFor(c) // cached, one DB hit per request
		if err != nil {
			c.Next()
			return
		}
		if sub.CoachLimit > 0 && sub.CoachCount >= sub.CoachLimit {
			utils.SafeErrorResponse(c, http.StatusPaymentRequired, nil, "coach limit reached for your plan")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (q *QuotaMiddleware) CheckStorageLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := q.subFor(c) // cached, one DB hit per request
		if err != nil {
			c.Next()
			return
		}
		if sub.StorageLimitBytes > 0 && sub.StorageUsedBytes >= sub.StorageLimitBytes {
			utils.SafeErrorResponse(c, http.StatusPaymentRequired, nil, "storage limit reached for your plan")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (q *QuotaMiddleware) CheckTestLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := q.subFor(c) // cached, one DB hit per request
		if err != nil {
			c.Next()
			return
		}
		// test_limit = -1 means unlimited
		if sub.TestLimit > 0 && sub.TestCountThisMonth >= sub.TestLimit {
			utils.SafeErrorResponse(c, http.StatusPaymentRequired, nil, "monthly test limit reached for your plan")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (q *QuotaMiddleware) CheckSQIAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := q.subFor(c) // cached, one DB hit per request
		if err != nil {
			c.Next()
			return
		}
		if !sub.SQIAccess {
			utils.SafeErrorResponse(c, http.StatusPaymentRequired, nil, "SQI analytics not available on your plan")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (q *QuotaMiddleware) CheckVideoProctoringAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := q.subFor(c) // cached, one DB hit per request
		if err != nil {
			c.Next()
			return
		}
		if !sub.VideoProctoringIncluded {
			utils.SafeErrorResponse(c, http.StatusPaymentRequired, nil, "video proctoring not available on your plan")
			c.Abort()
			return
		}
		c.Next()
	}
}
