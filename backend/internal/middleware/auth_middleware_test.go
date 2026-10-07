package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

func initAuthMiddlewareTest() {
	gin.SetMode(gin.TestMode)
	utils.InitJWTConfigWithVideoSecret(
		"main-secret-for-middleware-tests",
		"video-secret-for-middleware-tests",
		"1h", "eduquant",
	)
}

// --- extractToken: X-Role preferred cookie wins, else first available ---

func ctxWithCookies(cookies map[string]string, headers map[string]string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	for name, val := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: val})
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

func TestExtractTokenPrefersXRole(t *testing.T) {
	c := ctxWithCookies(
		map[string]string{"admin_token": "admin-jwt", "coach_token": "coach-jwt"},
		map[string]string{"X-Role": "coach"},
	)
	if got := extractToken(c, []string{"admin", "coach"}); got != "coach-jwt" {
		t.Fatalf("got %q, want coach-jwt", got)
	}
}

func TestExtractTokenFallsBackToFirstAvailable(t *testing.T) {
	c := ctxWithCookies(
		map[string]string{"coach_token": "coach-jwt"},
		nil,
	)
	if got := extractToken(c, []string{"admin", "coach"}); got != "coach-jwt" {
		t.Fatalf("got %q, want coach-jwt", got)
	}
}

func TestExtractTokenXRoleNotInRolesList(t *testing.T) {
	// X-Role asks for student, but that role's cookie isn't in the allowed list;
	// must fall through to the first cookie that IS in the list.
	c := ctxWithCookies(
		map[string]string{"admin_token": "admin-jwt", "student_token": "student-jwt"},
		map[string]string{"X-Role": "student"},
	)
	if got := extractToken(c, []string{"admin", "coach"}); got != "admin-jwt" {
		t.Fatalf("got %q, want admin-jwt", got)
	}
}

func TestExtractTokenNonePresent(t *testing.T) {
	c := ctxWithCookies(nil, nil)
	if got := extractToken(c, []string{"admin"}); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

// --- AuthMiddleware: no-DB failure paths ---

func TestAuthMiddlewareMissingToken(t *testing.T) {
	initAuthMiddlewareTest()
	r := gin.New()
	r.GET("/x", AuthMiddleware(nil, nil, "admin"), func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	initAuthMiddlewareTest()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "admin_token", Value: "garbage"})
	c.Request = req
	hf := AuthMiddleware(nil, nil, "admin")
	hf(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
}

func TestAuthMiddlewareWrongRole(t *testing.T) {
	initAuthMiddlewareTest()
	tok, _ := utils.GenerateToken(1, "student", 5, 1)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "admin_token", Value: tok})
	c.Request = req
	hf := AuthMiddleware(nil, nil, "admin")
	hf(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

// --- VideoTokenMiddleware: no-DB paths (query token + route binding) ---

func runVideoRequest(token, idParam string) *httptest.ResponseRecorder {
	initAuthMiddlewareTest()
	r := gin.New()
	r.GET("/video/:id", VideoTokenMiddleware(), func(c *gin.Context) {
		c.JSON(200, gin.H{"assignment_id": c.GetInt("assignment_id"), "role": c.GetString("role")})
	})
	w := httptest.NewRecorder()
	url := "/video/" + idParam
	if token != "" {
		url += "?token=" + token
	}
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	return w
}

func TestVideoTokenMiddlewareValid(t *testing.T) {
	tok, _ := utils.GenerateVideoToken(7, 3, "coach")
	w := runVideoRequest(tok, "7")
	if w.Code != 200 {
		t.Fatalf("got %d, want 200: %s", w.Code, w.Body.String())
	}
}

func TestVideoTokenMiddlewareMissing(t *testing.T) {
	w := runVideoRequest("", "7")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
}

func TestVideoTokenMiddlewareGarbage(t *testing.T) {
	w := runVideoRequest("not-a-real-token", "7")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
}

func TestVideoTokenMiddlewareAssignmentMismatch(t *testing.T) {
	tok, _ := utils.GenerateVideoToken(7, 3, "coach")
	w := runVideoRequest(tok, "8")
	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", w.Code)
	}
}

func TestVideoTokenMiddlewareMainTokenRejected(t *testing.T) {
	// A full session token must not work as a video token.
	tok, _ := utils.GenerateToken(1, "coach", 0, 1)
	w := runVideoRequest(tok, "7")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}
