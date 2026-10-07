package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-student-diagnostic/backend/utils"

	"github.com/gin-gonic/gin"
)

func postJSON(t *testing.T, body string) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	return w, c
}

func TestGetProfileUnauthorized(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	// no user_id in context
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.GetProfile(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestGetProfileNotFound(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Set("user_id", 999999999)
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.GetProfile(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d, want 404", w.Code)
	}
}

func TestGetProfileOK(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.GetProfile(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateProfileUnauthorized(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := postJSON(t, `{"display_name":"X","phone":"1"}`)
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.UpdateProfile(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestUpdateProfileOK(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := postJSON(t, `{"display_name":"Ada","phone":"555"}`)
	c.Set("user_id", f.AdminUserID)
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.UpdateProfile(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	// verify persisted
	p, err := f.ProfileRepo.GetByUserID(f.AdminUserID)
	if err != nil || p.DisplayName == nil || *p.DisplayName != "Ada" {
		t.Fatalf("persisted=%+v err=%v", p, err)
	}
}

func TestUpdatePasswordUnauthorized(t *testing.T) {
	f := newHandlerFixture(t)
	w, c := postJSON(t, `{"current_password":"x","new_password":"newpassword"}`)
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.UpdatePassword(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestUpdatePasswordWeakNewPassword(t *testing.T) {
	f := newHandlerFixture(t)
	hashed, _ := utils.HashPassword("currentpass")
	f.UserRepo.UpdatePassword(f.AdminUserID, hashed)

	w, c := postJSON(t, `{"current_password":"currentpass","new_password":"short"}`)
	c.Set("user_id", f.AdminUserID)
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.UpdatePassword(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestUpdatePasswordWrongCurrentPassword(t *testing.T) {
	f := newHandlerFixture(t)
	hashed, _ := utils.HashPassword("currentpass")
	f.UserRepo.UpdatePassword(f.AdminUserID, hashed)

	w, c := postJSON(t, `{"current_password":"wrong","new_password":"newpassword"}`)
	c.Set("user_id", f.AdminUserID)
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.UpdatePassword(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestUpdatePasswordOK(t *testing.T) {
	f := newHandlerFixture(t)
	hashed, _ := utils.HashPassword("currentpass")
	f.UserRepo.UpdatePassword(f.AdminUserID, hashed)

	w, c := postJSON(t, `{"current_password":"currentpass","new_password":"newpassword"}`)
	c.Set("user_id", f.AdminUserID)
	h := NewProfileHandler(f.ProfileRepo, f.UserRepo)
	h.UpdatePassword(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	// new hash must verify
	newHash, _ := f.UserRepo.GetPasswordHash(f.AdminUserID)
	if err := utils.CheckPassword("newpassword", newHash); err != nil {
		t.Fatalf("new password does not verify: %v", err)
	}
}
