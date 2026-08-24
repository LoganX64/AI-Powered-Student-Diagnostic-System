package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"ai-student-diagnostic/backend/internal/repository"

	_ "github.com/lib/pq"
	"github.com/gin-gonic/gin"
)

func qtestDB(t *testing.T) *sql.DB {
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

// freeTenant returns a tenant currently on the Free plan (sqi_access=false,
// video_proctoring_included=false) so we can assert blocking behavior.
func freeTenant(t *testing.T, db *sql.DB) int {
	var tid int
	err := db.QueryRow(`
		SELECT ts.tenant_id FROM tenant_subscriptions ts
		JOIN subscription_plans sp ON sp.id = ts.plan_id
		WHERE sp.slug = 'free' LIMIT 1
	`).Scan(&tid)
	if err != nil {
		t.Skip("no free-plan tenant available:", err)
	}
	return tid
}

// runQuota builds a tiny gin engine that sets tenant_id then applies the given
// quota middleware, finishing at a 200 handler.
func runQuota(tid int, mw gin.HandlerFunc) int {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tid)
		c.Next()
	})
	r.GET("/x", mw, func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	return w.Code
}

func TestQuotaFreeTierBlocksSQI(t *testing.T) {
	db := qtestDB(t)
	defer db.Close()
	tid := freeTenant(t, db)
	qm := NewQuotaMiddleware(repository.NewSubscriptionRepo(db), repository.NewPlanRepo(db))

	code := runQuota(tid, qm.CheckSQIAccess())
	if code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 for SQI on free plan, got %d", code)
	}
}

func TestQuotaFreeTierBlocksVideoProctoring(t *testing.T) {
	db := qtestDB(t)
	defer db.Close()
	tid := freeTenant(t, db)
	qm := NewQuotaMiddleware(repository.NewSubscriptionRepo(db), repository.NewPlanRepo(db))

	code := runQuota(tid, qm.CheckVideoProctoringAccess())
	if code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 for proctoring on free plan, got %d", code)
	}
}

func TestQuotaWithinStudentLimitAllows(t *testing.T) {
	db := qtestDB(t)
	defer db.Close()
	tid := freeTenant(t, db)
	qm := NewQuotaMiddleware(repository.NewSubscriptionRepo(db), repository.NewPlanRepo(db))

	code := runQuota(tid, qm.CheckStudentLimit())
	if code != http.StatusOK {
		t.Fatalf("expected 200 within student limit, got %d", code)
	}
}

func TestQuotaWithinTestLimitAllows(t *testing.T) {
	db := qtestDB(t)
	defer db.Close()
	tid := freeTenant(t, db)
	qm := NewQuotaMiddleware(repository.NewSubscriptionRepo(db), repository.NewPlanRepo(db))

	code := runQuota(tid, qm.CheckTestLimit())
	if code != http.StatusOK {
		t.Fatalf("expected 200 within test limit, got %d", code)
	}
}
