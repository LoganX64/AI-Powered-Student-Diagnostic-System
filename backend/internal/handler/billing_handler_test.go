package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"
	"ai-student-diagnostic/backend/internal/services"
	"ai-student-diagnostic/backend/internal/testutil"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func newBillingHandler(db *sql.DB) *BillingHandler {
	userRepo := repository.NewUserRepo(db)
	return NewBillingHandler(
		repository.NewPlanRepo(db),
		repository.NewSubscriptionRepo(db),
		nil, nil,
		repository.NewTenantRepo(db),
		nil, nil,
		services.NewNotificationService(repository.NewNotificationRepo(db), userRepo),
	)
}

func TestBillingListPlans(t *testing.T) {
	db := testutil.OpenTestDB(t)
	h := newBillingHandler(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.ListPlans(c)
	if w.Code != http.StatusOK {
		t.Fatalf("ListPlans expected 200, got %d", w.Code)
	}
}

// TestBillingListPlansFeaturesArray guards against serializing the JSONB
// features column as a base64 string (the []byte default), which makes the
// billing page crash on features.map().
func TestBillingListPlansFeaturesArray(t *testing.T) {
	db := testutil.OpenTestDB(t)
	h := newBillingHandler(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.ListPlans(c)
	if w.Code != http.StatusOK {
		t.Fatalf("ListPlans expected 200, got %d", w.Code)
	}

	var resp struct {
		Data []struct {
			Features json.RawMessage `json:"features"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("expected at least one plan")
	}
	var features []string
	if err := json.Unmarshal(resp.Data[0].Features, &features); err != nil {
		t.Fatalf("features is not a JSON array: %v (raw=%q)", err, string(resp.Data[0].Features))
	}
}

func TestBillingGetSubscription(t *testing.T) {
	db := testutil.OpenTestDB(t)
	h := newBillingHandler(db)

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants ORDER BY id LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant:", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", tid)
	h.GetSubscription(c)
	if w.Code != http.StatusOK {
		t.Fatalf("GetSubscription expected 200, got %d", w.Code)
	}
}

func TestBillingAssignPlan(t *testing.T) {
	db := testutil.OpenTestDB(t)
	h := newBillingHandler(db)

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants ORDER BY id LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant:", err)
	}
	var origPlan int
	db.QueryRow(`SELECT plan_id FROM tenant_subscriptions WHERE tenant_id = $1`, tid).Scan(&origPlan)
	defer db.Exec(`UPDATE tenant_subscriptions SET plan_id = $1, updated_at = NOW() WHERE tenant_id = $2`, origPlan, tid)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", tid)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(tid)}}
	body, _ := json.Marshal(map[string]int{"plan_id": 2}) // starter
	c.Request = httptest.NewRequest(http.MethodPut, "/x", bytes.NewReader(body))
	h.AssignPlan(c)
	if w.Code != http.StatusOK {
		t.Fatalf("AssignPlan expected 200, got %d", w.Code)
	}

	var newPlan int
	db.QueryRow(`SELECT plan_id FROM tenant_subscriptions WHERE tenant_id = $1`, tid).Scan(&newPlan)
	if newPlan != 2 {
		t.Fatalf("AssignPlan did not persist plan_id, got %d", newPlan)
	}
}

func TestCancelRevertsToFree(t *testing.T) {
	db := testutil.OpenTestDB(t)
	h := newBillingHandler(db)

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants ORDER BY id LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant:", err)
	}
	var origPlan int
	db.QueryRow(`SELECT plan_id FROM tenant_subscriptions WHERE tenant_id = $1`, tid).Scan(&origPlan)
	defer db.Exec(`UPDATE tenant_subscriptions SET plan_id = $1, updated_at = NOW() WHERE tenant_id = $2`, origPlan, tid)

	// assign a paid plan (starter = 2)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", tid)
	c.Params = gin.Params{{Key: "id", Value: strconv.Itoa(tid)}}
	body, _ := json.Marshal(map[string]int{"plan_id": 2})
	c.Request = httptest.NewRequest(http.MethodPut, "/x", bytes.NewReader(body))
	h.AssignPlan(c)
	if w.Code != http.StatusOK {
		t.Fatalf("AssignPlan expected 200, got %d", w.Code)
	}

	// cancel
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Set("tenant_id", tid)
	c2.Request = httptest.NewRequest(http.MethodPost, "/x", nil)
	h.CancelSubscription(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("CancelSubscription expected 200, got %d", w2.Code)
	}

	// plan must have reverted to Free (id 1)
	var afterPlan int
	db.QueryRow(`SELECT plan_id FROM tenant_subscriptions WHERE tenant_id = $1`, tid).Scan(&afterPlan)
	if afterPlan != 1 {
		t.Fatalf("CancelSubscription did not revert to Free, got plan_id=%d", afterPlan)
	}
}
