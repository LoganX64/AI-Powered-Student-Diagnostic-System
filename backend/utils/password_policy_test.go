package utils

import (
	"strings"
	"testing"
)

func TestValidatePasswordTooShort(t *testing.T) {
	if err := ValidatePassword(""); err == nil {
		t.Fatal("empty password must be rejected")
	}
	if err := ValidatePassword("1234567"); err == nil {
		t.Fatal("7-char password must be rejected")
	}
	if err := ValidatePassword(strings.Repeat("a", 7)); err == nil {
		t.Fatal("7-char password must be rejected")
	}
}

func TestValidatePasswordBoundary(t *testing.T) {
	if err := ValidatePassword(strings.Repeat("a", 8)); err != nil {
		t.Fatalf("8-char password must pass: %v", err)
	}
	if err := ValidatePassword(strings.Repeat("a", 64)); err != nil {
		t.Fatalf("64-char password must pass: %v", err)
	}
}

func TestValidatePasswordMessageMatchesFrontend(t *testing.T) {
	err := ValidatePassword("short")
	if err == nil || err.Error() != "Password must be at least 8 characters" {
		t.Fatalf("error = %v", err)
	}
}
