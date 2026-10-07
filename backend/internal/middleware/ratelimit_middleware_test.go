package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInMemoryRateLimiterBlocksAfterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewRateLimiter(nil, 3))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 1; i <= 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "1.2.3.4:9999"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, w.Code)
		}
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "1.2.3.4:9999"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: got %d, want 429", w.Code)
	}
}

func TestInMemoryRateLimiterPerIPIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewRateLimiter(nil, 1))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req1.RemoteAddr = "1.1.1.1:1"
	r.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2.RemoteAddr = "2.2.2.2:1"
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second IP should not be limited: got %d", w2.Code)
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req3.RemoteAddr = "1.1.1.1:1"
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("same IP second hit: got %d, want 429", w3.Code)
	}
}

func TestDefaultLimitWhenZero(t *testing.T) {
	if DefaultLimit <= 0 {
		t.Fatal("DefaultLimit must be positive")
	}
	// NewRateLimiter with limit=0 should fall back to DefaultLimit, i.e. only
	// block after DefaultLimit requests per window, not immediately.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewRateLimiter(nil, 0))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	for i := 0; i < DefaultLimit; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "9.9.9.9:9"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d got %d, want 200", i+1, w.Code)
		}
	}
	fmt.Println("DefaultLimit fallback works")
}
