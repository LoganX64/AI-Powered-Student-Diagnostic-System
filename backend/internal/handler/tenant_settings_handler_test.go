package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func ginCreate(w *httptest.ResponseRecorder) (*gin.Context, *gin.Engine) {
	c, _ := gin.CreateTestContext(w)
	return c, nil
}

// postBody is the JSON payload for a mutating request body.
func postBody(s string) *bytes.Reader { return bytes.NewReader([]byte(s)) }

func TestTenantSettingsGetUnauthorized(t *testing.T) {
	f := newHandlerFixture(t)
	w := httptest.NewRecorder()
	c, _ := ginCreate(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	// tenant_id deliberately absent
	h := NewTenantSettingsHandler(f.TenantRepo)
	h.GetSettings(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}

func TestTenantSettingsGetOK(t *testing.T) {
	f := newHandlerFixture(t)
	c, w := f.ctx(t, "admin")
	h := NewTenantSettingsHandler(f.TenantRepo)
	h.GetSettings(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestTenantSettingsUpdateRoundtrip(t *testing.T) {
	f := newHandlerFixture(t)
	h := NewTenantSettingsHandler(f.TenantRepo)

	w, c := postJSON(t, `{"key":"theme","value":"dark"}`)
	c.Set("tenant_id", f.TenantID)
	h.UpdateSettings(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	got, err := f.TenantRepo.GetSettings(f.TenantID)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if _, ok := got["theme"]; !ok {
		t.Fatalf("theme not persisted: %v", got)
	}
}

func TestTenantSettingsUpdateInvalidPayload(t *testing.T) {
	f := newHandlerFixture(t)
	h := NewTenantSettingsHandler(f.TenantRepo)

	w, c := postJSON(t, `{}`)
	c.Set("tenant_id", f.TenantID)
	h.UpdateSettings(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", w.Code)
	}
}

func TestTenantSettingsUpdateName(t *testing.T) {
	f := newHandlerFixture(t)
	h := NewTenantSettingsHandler(f.TenantRepo)

	w, c := postJSON(t, `{"name":"Renamed Org"}`)
	c.Set("tenant_id", f.TenantID)
	h.UpdateTenantName(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	tenant, err := f.TenantRepo.GetByID(f.TenantID)
	if err != nil || tenant.Name != "Renamed Org" {
		t.Fatalf("tenant=%+v err=%v", tenant, err)
	}
}

func TestTenantSettingsUpdateNameUnauthorized(t *testing.T) {
	f := newHandlerFixture(t)
	h := NewTenantSettingsHandler(f.TenantRepo)

	w, c := postJSON(t, `{"name":"X"}`)
	h.UpdateTenantName(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401", w.Code)
	}
}
