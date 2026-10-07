package utils

import (
	"strings"
	"testing"
	"time"
)

func TestNewJWTManagerDefaults(t *testing.T) {
	m := NewJWTManager("secret", "", "issuer")
	if m.expiry != 4*time.Hour {
		t.Fatalf("expiry = %v, want 4h", m.expiry)
	}
	if m.issuer != "issuer" {
		t.Fatalf("issuer = %q", m.issuer)
	}
}

func TestNewJWTManagerInvalidExpiryFallsBack(t *testing.T) {
	m := NewJWTManager("secret", "not-a-duration", "issuer")
	if m.expiry != 4*time.Hour {
		t.Fatalf("expiry = %v, want 4h", m.expiry)
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	m := NewJWTManager("topsecret-that-is-long-enough", "1h", "eduquant")
	tok, err := m.GenerateToken(7, "coach", 0, 42)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := m.ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != 7 || claims.Role != "coach" || claims.TenantID != 42 {
		t.Fatalf("claims = %+v", claims)
	}
	if claims.Issuer != "eduquant" {
		t.Fatalf("issuer = %q", claims.Issuer)
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	a := NewJWTManager("secret-a-that-is-long-enough", "1h", "eduquant")
	b := NewJWTManager("secret-b-that-is-long-enough", "1h", "eduquant")
	tok, _ := a.GenerateToken(1, "admin", 0, 1)
	if _, err := b.ValidateToken(tok); err == nil {
		t.Fatal("expected error validating with wrong secret")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	m := NewJWTManager("secret", "-1h", "eduquant")
	tok, _ := m.GenerateToken(1, "admin", 0, 1)
	if _, err := m.ValidateToken(tok); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateTokenIssuerMismatch(t *testing.T) {
	a := NewJWTManager("secret-that-is-long-enough", "1h", "issuer-a")
	b := NewJWTManager("secret-that-is-long-enough", "1h", "issuer-b")
	tok, _ := a.GenerateToken(1, "admin", 0, 1)
	if _, err := b.ValidateToken(tok); err == nil {
		t.Fatal("expected issuer mismatch error")
	}
}

func TestValidateTokenRejectsNonHMAC(t *testing.T) {
	m := NewJWTManager("secret", "1h", "eduquant")
	if _, err := m.ValidateToken("not-a-jwt"); err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestVideoTokenUsesSeparateSecret(t *testing.T) {
	m := NewJWTManagerWithVideoSecret("main-secret-long-enough-xxxx", "video-secret-long-enough-xxxx", "1h", "eduquant")
	tok, err := m.GenerateVideoToken(99, 5, "coach")
	if err != nil {
		t.Fatalf("GenerateVideoToken: %v", err)
	}
	claims, err := m.ValidateVideoToken(tok)
	if err != nil {
		t.Fatalf("ValidateVideoToken: %v", err)
	}
	if claims.AssignmentID != 99 || claims.Action != "video_stream" || claims.TenantID != 5 || claims.Role != "coach" {
		t.Fatalf("claims = %+v", claims)
	}
	// The video token must NOT validate against the main secret.
	if _, err := m.ValidateToken(tok); err == nil {
		t.Fatal("video token must not pass ValidateToken with the main secret")
	}
	// And a main token must not pass video validation.
	mainTok, _ := m.GenerateToken(1, "admin", 0, 1)
	if _, err := m.ValidateVideoToken(mainTok); err == nil {
		t.Fatal("main token must not pass ValidateVideoToken")
	}
}

func TestVideoTokenRejectsNonHMACAlg(t *testing.T) {
	m := NewJWTManagerWithVideoSecret("main-secret-long-enough-xxxx", "video-secret-long-enough-xxxx", "1h", "eduquant")
	if _, err := m.ValidateVideoToken(strings.Repeat("a", 20)); err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestDefaultManagerRoundTrip(t *testing.T) {
	InitJWTConfig("default-secret-long-enough-xxxx", "1h", "eduquant")
	tok, err := GenerateToken(3, "student", 11, 8)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.StudentID != 11 || claims.TenantID != 8 {
		t.Fatalf("claims = %+v", claims)
	}
}
