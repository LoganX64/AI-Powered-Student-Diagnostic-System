package utils

import (
	"fmt"
	"strings"
	"testing"
)

func TestAuthCookieName(t *testing.T) {
	if got := AuthCookieName("admin"); got != "admin_token" {
		t.Fatalf("AuthCookieName(admin) = %q", got)
	}
	if got := AuthCookieName("student"); got != "student_token" {
		t.Fatalf("AuthCookieName(student) = %q", got)
	}
}

func TestSetAuthCookieFlags(t *testing.T) {
	InitJWTConfig("cookie-test-secret-long-enough", "1h", "eduquant")
	c, w := newTestContext(t)
	SetAuthCookie(c, "coach", "tok123")
	header := w.Header().Get("Set-Cookie")
	if !strings.Contains(header, "coach_token=tok123") {
		t.Fatalf("cookie = %q", header)
	}
	if !strings.Contains(header, "HttpOnly") {
		t.Fatalf("cookie missing HttpOnly: %q", header)
	}
	if !strings.Contains(header, "Path=/") {
		t.Fatalf("cookie missing Path=/: %q", header)
	}
	if !strings.Contains(header, "SameSite=Lax") {
		t.Fatalf("cookie missing SameSite=Lax: %q", header)
	}
	if !strings.Contains(header, fmt.Sprintf("Max-Age=%d", 3600)) {
		t.Fatalf("cookie Max-Age mismatch: %q", header)
	}
}

func TestClearAuthCookiesClearsAllRoles(t *testing.T) {
	c, w := newTestContext(t)
	ClearAuthCookies(c)
	header := w.Header().Values("Set-Cookie")
	if len(header) != 4 {
		t.Fatalf("want 4 Set-Cookie headers, got %d: %v", len(header), header)
	}
	for _, role := range []string{"admin", "coach", "student", "super_admin"} {
		found := false
		for _, h := range header {
			if strings.HasPrefix(h, role+"_token=;") {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing clear cookie for %s: %v", role, header)
		}
	}
}
