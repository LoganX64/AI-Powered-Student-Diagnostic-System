package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func runRoleRequest(roleVal interface{}, roles ...string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Request = req
	if roleVal != nil {
		c.Set("role", roleVal)
	}
	hf := RoleMiddleware(roles...)
	hf(c)
	return w
}

func TestRoleMiddlewareMissingRole(t *testing.T) {
	w := runRoleRequest(nil, "admin")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestRoleMiddlewareNonStringRole(t *testing.T) {
	w := runRoleRequest(12345, "admin")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestRoleMiddlewareAllowed(t *testing.T) {
	w := runRoleRequest("coach", "admin", "coach")
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Fatalf("got %d, want passthrough", w.Code)
	}
}

func TestRoleMiddlewareForbidden(t *testing.T) {
	w := runRoleRequest("student", "admin", "coach")
	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", w.Code)
	}
}
