package routes

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/config"
	"ai-student-diagnostic/backend/internal/testutil"

	"github.com/gin-gonic/gin"
)

// routes_test.go is the wiring safety net: it builds the real router through
// SetupRouter and asserts the method+path table, so a renamed or dropped route
// fails here rather than in production.
//
// SetupRouter has no seam — it starts the queue consumer and a 30s sweeper — so
// the test is built around that: shutdown is always deferred, cfg keeps
// RedisEnabled=false (which leaves autosaveBuffer nil so shutdown cannot block
// on a flush), and trustedProxies is nil to stay off the log.Fatalf path.

// setupRouter builds the router once per test and registers cleanup.
func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.OpenTestDB(t)

	cfg := &config.Config{
		RedisEnabled:       false, // keeps autosaveBuffer nil => shutdown never blocks
		ComputeChunkSize:   100,
		SubmitGraceSeconds: 30,
		ScaleBandC:         50000,
		UploadDir:          t.TempDir(),
		FrontendURL:        "http://localhost:5173",
	}

	r, shutdown := SetupRouter(db, cfg, nil, nil)
	t.Cleanup(func() {
		if err := shutdown(); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	return r
}

// routeTable returns the registered routes as "METHOD path" strings.
func routeTable(r *gin.Engine) []string {
	routes := r.Routes()
	out := make([]string, 0, len(routes))
	for _, rt := range routes {
		out = append(out, rt.Method+" "+rt.Path)
	}
	sort.Strings(out)
	return out
}

func hasRoute(table []string, method, path string) bool {
	want := method + " " + path
	for _, r := range table {
		if r == want {
			return true
		}
	}
	return false
}

func TestSetupRouterRegistersCoreRoutes(t *testing.T) {
	r := setupRouter(t)
	table := routeTable(r)

	// The groups below are the ones whose absence would break the app hardest.
	required := []struct{ method, path string }{
		// health
		{"GET", "/health"},
		// auth
		{"POST", "/auth/login"},
		{"POST", "/auth/register-admin"},
		{"POST", "/auth/logout"},
		{"POST", "/auth/forgot-password"},
		{"POST", "/auth/reset-password"},
		// profile (live PUT /auth/password)
		{"GET", "/auth/profile"},
		{"PUT", "/auth/profile"},
		{"PUT", "/auth/password"},
		// student
		{"POST", "/student/login"},
		{"GET", "/student/assignments"},
		{"GET", "/student/assignments/:id/questions"},
		{"POST", "/student/assignments/:id/start"},
		{"POST", "/student/assignments/:id/autosave"},
		{"GET", "/student/assignments/:id/state"},
		{"POST", "/student/assignments/:id/submit"},
		{"POST", "/student/assignments/:id/video-chunk"},
		{"POST", "/student/submit/:id"},
		// admin
		{"POST", "/admin/subjects"},
		{"GET", "/admin/subjects"},
		{"POST", "/admin/students"},
		{"GET", "/admin/students"},
		{"GET", "/admin/students/:id"},
		{"POST", "/admin/tests"},
		{"GET", "/admin/tests"},
		{"GET", "/admin/tests/:id"},
		{"PUT", "/admin/tests/:id/reactivate"},
		{"PUT", "/admin/subjects/:id/reactivate"},
		{"POST", "/admin/assignments"},
		{"DELETE", "/admin/assignments/:id"},
		{"GET", "/admin/coaches"},
		{"GET", "/admin/batches"},
		// super-admin
		{"GET", "/super-admin/stats"},
		{"GET", "/super-admin/tenants"},
		{"GET", "/super-admin/plans"},
	}
	for _, req := range required {
		if !hasRoute(table, req.method, req.path) {
			t.Errorf("missing route %s %s", req.method, req.path)
		}
	}
}

func TestSetupRouterRegistersViewerRoutes(t *testing.T) {
	r := setupRouter(t)
	table := routeTable(r)

	// The viewer WS group is registered under /view with admin|coach auth.
	found := false
	for _, rt := range table {
		if strings.Contains(rt, "/view/") {
			found = true
		}
	}
	if !found {
		t.Errorf("no /view/* routes registered; got %v", table)
	}
}

func TestSetupRouterHealthOK(t *testing.T) {
	r := setupRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("/health = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "healthy") {
		t.Fatalf("/health body=%q", w.Body.String())
	}
}

// TestSetupRouterProtectedRoutesRequireAuth proves the auth middleware is
// actually wired in front of the protected groups. Without a token these must
// 401, not 200/500.
func TestSetupRouterProtectedRoutesRequireAuth(t *testing.T) {
	r := setupRouter(t)

	cases := []struct{ method, path string }{
		{http.MethodGet, "/admin/students"},
		{http.MethodGet, "/admin/tests"},
		{http.MethodGet, "/super-admin/tenants"},
		{http.MethodGet, "/auth/profile"},
		{http.MethodGet, "/student/assignments"},
		{http.MethodGet, "/coach/students"},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(c.method, c.path, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401 (auth middleware not wired?)", c.method, c.path, w.Code)
		}
	}
}

// TestSetupRouterNoRouteReturns404 proves the NoRoute handler is wired.
func TestSetupRouterNoRouteReturns404(t *testing.T) {
	r := setupRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/definitely-not-a-route", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown route = %d, want 404", w.Code)
	}
}

// TestSetupRouterCORSPreflight asserts the CORS closure short-circuits OPTIONS.
func TestSetupRouterCORSPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.OpenTestDB(t)
	cfg := &config.Config{RedisEnabled: false, ComputeChunkSize: 100, SubmitGraceSeconds: 30, UploadDir: t.TempDir()}

	r, shutdown := SetupRouter(db, cfg, []string{"http://allowed.example"}, nil)
	t.Cleanup(func() { shutdown() })

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "http://allowed.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS = %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://allowed.example" {
		t.Fatalf("Allow-Origin=%q, want the reflected allowed origin", got)
	}
}

func TestSetupRouterCORSDisallowedOriginNotReflected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.OpenTestDB(t)
	cfg := &config.Config{RedisEnabled: false, ComputeChunkSize: 100, SubmitGraceSeconds: 30, UploadDir: t.TempDir()}

	r, shutdown := SetupRouter(db, cfg, []string{"http://allowed.example"}, nil)
	t.Cleanup(func() { shutdown() })

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin=%q, want empty for a disallowed origin", got)
	}
}
