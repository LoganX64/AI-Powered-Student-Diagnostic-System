package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"

	_ "github.com/lib/pq"
	"github.com/gin-gonic/gin"
)

func btestDB(t *testing.T) *sql.DB {
	url := os.Getenv("DB_URL")
	if url == "" {
		url = "postgres://postgres:9908@localhost:5432/sqi_db?sslmode=disable"
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return db
}

func newBillingHandler(db *sql.DB) *BillingHandler {
	return NewBillingHandler(
		repository.NewPlanRepo(db),
		repository.NewSubscriptionRepo(db),
		nil, nil, nil,
		nil,
	)
}

func TestBillingListPlans(t *testing.T) {
	db := btestDB(t)
	defer db.Close()
	h := newBillingHandler(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.ListPlans(c)
	if w.Code != http.StatusOK {
		t.Fatalf("ListPlans expected 200, got %d", w.Code)
	}
}

func TestBillingGetSubscription(t *testing.T) {
	db := btestDB(t)
	defer db.Close()
	h := newBillingHandler(db)

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tid); err != nil {
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
	db := btestDB(t)
	defer db.Close()
	h := newBillingHandler(db)

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant:", err)
	}
	// capture original plan to restore later
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
	// confirm it applied
	var newPlan int
	db.QueryRow(`SELECT plan_id FROM tenant_subscriptions WHERE tenant_id = $1`, tid).Scan(&newPlan)
	if newPlan != 2 {
		t.Fatalf("AssignPlan did not persist plan_id, got %d", newPlan)
	}
}

// TestCancelRevertsToFree verifies Gap 1: cancelling a subscription reverts the
// tenant to the Free plan so quota checks downgrade immediately.
func TestCancelRevertsToFree(t *testing.T) {
	db := btestDB(t)
	defer db.Close()
	h := newBillingHandler(db)

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tid); err != nil {
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
