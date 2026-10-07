package services

import (
	"strings"
	"testing"
)

// --- Pure helpers extracted in Phase 6 Step 1 ---

// TestResetLinkTrimsTrailingSlash: a frontend URL configured with a trailing
// slash must not produce a double slash in the reset link.
func TestResetLinkTrimsTrailingSlash(t *testing.T) {
	cases := []struct {
		frontend, token, want string
	}{
		{"https://app.example", "tok", "https://app.example/reset-password?token=tok"},
		{"https://app.example/", "tok", "https://app.example/reset-password?token=tok"},
		{"https://app.example///", "tok", "https://app.example/reset-password?token=tok"},
		{"", "tok", "/reset-password?token=tok"},
	}
	for _, c := range cases {
		m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", c.frontend)
		if got := m.resetLink(c.token); got != c.want {
			t.Errorf("resetLink(frontend=%q)=%q, want %q", c.frontend, got, c.want)
		}
	}
}

func TestComposeIncludesLinkAndCaveat(t *testing.T) {
	m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", "https://app.example/")
	subject, body := m.compose("tok123")

	if subject != "EduQuant password reset" {
		t.Fatalf("subject=%q", subject)
	}
	if !strings.Contains(body, "https://app.example/reset-password?token=tok123") {
		t.Fatalf("body must contain the reset link: %q", body)
	}
	if !strings.Contains(body, "30 minutes") {
		t.Fatalf("body must state the expiry window: %q", body)
	}
	if !strings.Contains(body, "ignore this email") {
		t.Fatalf("body must include the do-nothing caveat: %q", body)
	}
}

// TestSendPasswordResetUnconfiguredDoesNotDial is the offline guarantee: with no
// SMTP credentials the mailer logs the link and returns nil without networking.
func TestSendPasswordResetUnconfiguredDoesNotDial(t *testing.T) {
	m := NewMailer("smtp.example", 587, "", "", "from@example.com", "https://app.example")
	if err := m.SendPasswordReset("to@example.com", "tok"); err != nil {
		t.Fatalf("unconfigured mailer must return nil, got %v", err)
	}
}

// TestSendPasswordResetUnreachableHostSurfacesError confirms that when SMTP is
// configured, a bad host produces an error rather than a silent success.
func TestSendPasswordResetUnreachableHostSurfacesError(t *testing.T) {
	// Port 1 on localhost refuses connections quickly.
	m := NewMailer("127.0.0.1", 1, "user", "pass", "from@example.com", "https://app.example")
	if err := m.SendPasswordReset("to@example.com", "tok"); err == nil {
		t.Fatal("expected an error when SMTP is unreachable")
	}
}
