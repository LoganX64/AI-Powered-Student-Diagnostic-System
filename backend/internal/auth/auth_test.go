package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-student-diagnostic/backend/internal/services"

	"github.com/gin-gonic/gin"
)

func newAuthHandlerForValidation() *AuthHandler {
	return &AuthHandler{
		AuthService: &services.AuthService{},
	}
}

func bindRequest(body string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestRegisterAdminInvalidPayload(t *testing.T) {
	h := newAuthHandlerForValidation()
	c := bindRequest(`{"email":"a@b.com"}`)
	h.RegisterAdmin(c)
	if c.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", c.Writer.Status())
	}
}

func TestRegisterAdminWeakPassword(t *testing.T) {
	h := newAuthHandlerForValidation()
	c := bindRequest(`{"email":"a@b.com","password":"short","org_name":"X"}`)
	h.RegisterAdmin(c)
	if c.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", c.Writer.Status())
	}
}

func TestRegisterCoachInvalidPayload(t *testing.T) {
	h := newAuthHandlerForValidation()
	c := bindRequest(`{"email":"a@b.com","password":"12345678","name":"n"}`)
	h.RegisterCoach(c)
	if c.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", c.Writer.Status())
	}
}

func TestRegisterCoachWeakPassword(t *testing.T) {
	h := newAuthHandlerForValidation()
	c := bindRequest(`{"email":"a@b.com","password":"1234567","name":"n","subject_ids":[1]}`)
	h.RegisterCoach(c)
	if c.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", c.Writer.Status())
	}
}

func TestUserLoginMissingFields(t *testing.T) {
	h := newAuthHandlerForValidation()
	c := bindRequest(`{"email":"a@b.com"}`)
	h.UserLogin(c)
	if c.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", c.Writer.Status())
	}
}

func TestForgotPasswordInvalidEmail(t *testing.T) {
	h := newAuthHandlerForValidation()
	c := bindRequest(`{"email":"not-an-email"}`)
	h.ForgotPassword(c)
	if c.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", c.Writer.Status())
	}
}

func TestResetPasswordWeakPassword(t *testing.T) {
	h := newAuthHandlerForValidation()
	c := bindRequest(`{"token":"abc","new_password":"short"}`)
	h.ResetPassword(c)
	if c.Writer.Status() != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", c.Writer.Status())
	}
}

func TestResetPasswordRepoNotConfigured(t *testing.T) {
	h := newAuthHandlerForValidation()
	// ResetRepo nil → clean 500 rather than nil-pointer panic
	c := bindRequest(`{"token":"abc","new_password":"long-enough-password"}`)
	h.ResetPassword(c)
	if c.Writer.Status() != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", c.Writer.Status())
	}
}

func TestLogoutClearsSingleRoleCookie(t *testing.T) {
	h := newAuthHandlerForValidation()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/logout", nil)
	c.Request.Header.Set("X-Role", "admin")
	h.Logout(c)
	headers := w.Header().Values("Set-Cookie")
	if len(headers) != 1 || !strings.Contains(headers[0], "admin_token=") {
		t.Fatalf("got %v", headers)
	}
}

func TestLogoutClearsAllWhenNoRole(t *testing.T) {
	h := newAuthHandlerForValidation()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/logout", nil)
	h.Logout(c)
	if n := len(w.Header().Values("Set-Cookie")); n != 4 {
		t.Fatalf("got %d Set-Cookie, want 4", n)
	}
}
