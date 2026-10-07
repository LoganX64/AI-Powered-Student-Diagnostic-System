package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestSafeErrorResponseNilErrorUnderDebug guards the regression where quota
// rejections pass a nil error (middleware/quota.go:127,143,159,176,192,208).
// Under DEBUG=true the helper used to call err.Error() on nil and panic,
// turning every 402 into a 500.
func TestSafeErrorResponseNilErrorUnderDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DEBUG", "true")

	c, w := newTestContext(t)
	SafeErrorResponse(c, http.StatusPaymentRequired, nil, "student limit reached for your plan")

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusPaymentRequired)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "student limit reached for your plan" {
		t.Fatalf("body error = %v, want the generic message", body["error"])
	}
}

func TestSafeErrorResponseNonNilErrorUnderDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DEBUG", "true")

	c, w := newTestContext(t)
	SafeErrorResponse(c, http.StatusPaymentRequired, errors.New("db exploded"), "generic")

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "db exploded" {
		t.Fatalf("body error = %v, want the raw error in DEBUG mode", body["error"])
	}
}

func TestSafeErrorResponseNonNilErrorWithoutDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DEBUG", "false")

	c, w := newTestContext(t)
	SafeErrorResponse(c, http.StatusPaymentRequired, errors.New("db exploded"), "generic")

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "generic" {
		t.Fatalf("body error = %v, want the generic message outside DEBUG", body["error"])
	}
}

func TestSafeErrorResponseMergesExtraFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DEBUG", "false")

	c, w := newTestContext(t)
	SafeErrorResponse(c, http.StatusPaymentRequired, nil, "locked", gin.H{"deactivated_id": 42})

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "locked" {
		t.Fatalf("body error = %v, want locked", body["error"])
	}
	if body["deactivated_id"] != float64(42) {
		t.Fatalf("deactivated_id = %v, want 42", body["deactivated_id"])
	}
}

func TestInternalErrorNilErrorUnderDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DEBUG", "true")

	c, w := newTestContext(t)
	InternalError(c, nil, "internal error")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "internal error" {
		t.Fatalf("body error = %v, want the generic message", body["error"])
	}
}

func TestInternalErrorNonNilErrorUnderDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DEBUG", "true")

	c, w := newTestContext(t)
	InternalError(c, errors.New("db exploded"), "internal error")

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "db exploded" {
		t.Fatalf("body error = %v, want the raw error in DEBUG mode", body["error"])
	}
}

func newTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	return c, w
}
