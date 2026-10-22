package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

type sessionResponseBody struct {
	Authenticated bool     `json:"authenticated"`
	Role          *string  `json:"role"`
	Roles         []string `json:"roles"`
}

func initSessionTestJWT() {
	utils.InitJWTConfig("session-test-secret-long-enough", "1h", "eduquant")
}

func runSessionRequest(t *testing.T, h *SessionHandler, cookies map[string]string) (*httptest.ResponseRecorder, sessionResponseBody) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	c.Request = req

	h.GetSession(c)

	var body sessionResponseBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode session body: %v (%s)", err, w.Body.String())
	}
	return w, body
}

func mustToken(t *testing.T, userID int, role string, studentID, tenantID int) string {
	t.Helper()
	tok, err := utils.GenerateToken(userID, role, studentID, tenantID)
	if err != nil {
		t.Fatalf("generate %s token: %v", role, err)
	}
	return tok
}

func TestSessionHandlerAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/session", nil)

	// No cookies means the handler never touches the repos.
	NewSessionHandler(nil, nil).GetSession(c)

	if r.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", r.Code)
	}
	if got := r.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store", got)
	}
	var body sessionResponseBody
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Authenticated {
		t.Fatal("authenticated=true, want false")
	}
	if body.Role != nil {
		t.Fatalf("role=%q, want null", *body.Role)
	}
	if len(body.Roles) != 0 {
		t.Fatalf("roles=%v, want empty", body.Roles)
	}
}

func TestSessionHandlerAuthenticatedRoles(t *testing.T) {
	initSessionTestJWT()
	f := newHandlerFixture(t)
	h := NewSessionHandler(f.UserRepo, f.StudentRepo)

	adminToken := mustToken(t, f.AdminUserID, "admin", 0, f.TenantID)
	studentToken := mustToken(t, 0, "student", f.StudentID, f.TenantID)

	t.Run("admin", func(t *testing.T) {
		w, body := runSessionRequest(t, h, map[string]string{
			utils.AuthCookieName("admin"): adminToken,
		})
		if w.Code != http.StatusOK || !body.Authenticated {
			t.Fatalf("status=%d body=%+v", w.Code, body)
		}
		if body.Role == nil || *body.Role != "admin" {
			t.Fatalf("role=%v, want admin", body.Role)
		}
		if len(body.Roles) != 1 || body.Roles[0] != "admin" {
			t.Fatalf("roles=%v, want [admin]", body.Roles)
		}
	})

	t.Run("student", func(t *testing.T) {
		_, body := runSessionRequest(t, h, map[string]string{
			utils.AuthCookieName("student"): studentToken,
		})
		if !body.Authenticated || body.Role == nil || *body.Role != "student" {
			t.Fatalf("body=%+v, want authenticated student", body)
		}
	})

	t.Run("cookie role mismatch", func(t *testing.T) {
		_, body := runSessionRequest(t, h, map[string]string{
			utils.AuthCookieName("coach"): adminToken,
		})
		if body.Authenticated || len(body.Roles) != 0 {
			t.Fatalf("body=%+v, want anonymous", body)
		}
	})

	t.Run("deleted user", func(t *testing.T) {
		missingToken := mustToken(t, 99999999, "admin", 0, f.TenantID)
		_, body := runSessionRequest(t, h, map[string]string{
			utils.AuthCookieName("admin"): missingToken,
		})
		if body.Authenticated {
			t.Fatalf("body=%+v, want anonymous for missing user", body)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, body := runSessionRequest(t, h, map[string]string{
			utils.AuthCookieName("admin"): "not-a-jwt",
		})
		if body.Authenticated {
			t.Fatalf("body=%+v, want anonymous for invalid token", body)
		}
	})
}

func TestSessionHandlerReturnsNewestRoleFirst(t *testing.T) {
	initSessionTestJWT()
	f := newHandlerFixture(t)
	h := NewSessionHandler(f.UserRepo, f.StudentRepo)

	adminToken := mustToken(t, f.AdminUserID, "admin", 0, f.TenantID)
	// JWTs carry second-precision IssuedAt, so cross a second boundary to make
	// the coach session strictly newer instead of relying on map/iteration order.
	time.Sleep(1100 * time.Millisecond)
	coachToken := mustToken(t, f.CoachUserID, "coach", 0, f.TenantID)

	_, body := runSessionRequest(t, h, map[string]string{
		utils.AuthCookieName("admin"): adminToken,
		utils.AuthCookieName("coach"): coachToken,
	})

	if !body.Authenticated || body.Role == nil {
		t.Fatalf("body=%+v, want authenticated", body)
	}
	if *body.Role != "coach" {
		t.Fatalf("role=%q, want newest coach", *body.Role)
	}
	if len(body.Roles) != 2 || body.Roles[0] != "coach" || body.Roles[1] != "admin" {
		t.Fatalf("roles=%v, want [coach admin]", body.Roles)
	}
}
