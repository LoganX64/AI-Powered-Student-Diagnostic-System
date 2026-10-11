package services

import (
	"strings"
	"testing"
)

// --- Result-released email ---

func TestResultLinkTrimsTrailingSlash(t *testing.T) {
	cases := []struct{ frontend, token, want string }{
		{"https://app.example", "tok", "https://app.example/student/results/tok"},
		{"https://app.example/", "tok", "https://app.example/student/results/tok"},
		{"https://app.example///", "tok", "https://app.example/student/results/tok"},
		{"", "tok", "/student/results/tok"},
	}
	for _, c := range cases {
		m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", c.frontend)
		if got := m.resultLink(c.token); got != c.want {
			t.Errorf("resultLink(frontend=%q)=%q, want %q", c.frontend, got, c.want)
		}
	}
}

func TestComposeResultReleasedStatesScoreAndLink(t *testing.T) {
	m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", "https://app.example/")
	subject, body := m.composeResultReleased("Bob", "Mathematics Midterm Exam 2026", "tok123", 12, 40, 30)

	if subject != "Your result for Mathematics Midterm Exam 2026 is available" {
		t.Fatalf("subject=%q", subject)
	}
	for _, want := range []string{
		"Hello Bob",
		"Mathematics Midterm Exam 2026",
		"12 out of 40",
		"30%",
		"https://app.example/student/results/tok123",
		"15 days",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q:\n%s", want, body)
		}
	}
}

// Scores render without trailing zeros so a stored 12.5 does not read "12.50",
// and a zero score is still shown as a number rather than being blank.
func TestComposeResultReleasedScoreFormatting(t *testing.T) {
	m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", "https://app.example")

	_, body := m.composeResultReleased("Bob", "Test", "tok", 12.5, 40, 31.25)
	if !strings.Contains(body, "12.5 out of 40 (31.25%)") {
		t.Errorf("fractional score not rendered cleanly:\n%s", body)
	}

	_, zero := m.composeResultReleased("Bob", "Test", "tok", 0, 40, 0)
	if !strings.Contains(zero, "0 out of 40 (0%)") {
		t.Errorf("zero score must still render:\n%s", zero)
	}

	// A negative net score is real (over-duration attempts produce one), so it
	// must be reported rather than clamped or dropped.
	_, neg := m.composeResultReleased("Bob", "Test", "tok", -1, 40, -2.5)
	if !strings.Contains(neg, "-1 out of 40 (-2.5%)") {
		t.Errorf("negative score must be reported verbatim:\n%s", neg)
	}
}

// A missing student name or test title must degrade gracefully rather than
// producing "Hello ," or a dangling "result for  is available".
func TestComposeResultReleasedHandlesMissingNames(t *testing.T) {
	m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", "https://app.example")

	subject, body := m.composeResultReleased("", "", "tok", 5, 10, 50)
	if subject != "Your result is available" {
		t.Fatalf("subject=%q", subject)
	}
	if !strings.Contains(body, "Hello there") {
		t.Errorf("expected a neutral greeting:\n%s", body)
	}
	if strings.Contains(body, "result for  ") {
		t.Errorf("blank test title left a dangling phrase:\n%s", body)
	}
}

// The test title is coach-supplied. CRLF in it must not be able to terminate a
// header or forge a header of its own.
func TestComposeResultReleasedSanitizesHeaderInjection(t *testing.T) {
	m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", "https://app.example")
	hostile := "Math\r\nBcc: attacker@evil.example\r\n\r\nignored body"

	subject, body := m.composeResultReleased("Bob", hostile, "tok", 1, 10, 10)

	if strings.ContainsAny(subject, "\r\n") {
		t.Fatalf("subject retained a newline: %q", subject)
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "bcc:") {
			t.Fatalf("injected header survived into the body:\n%s", body)
		}
	}
	if strings.Count(body, "\r") != 0 {
		t.Fatalf("body retained a carriage return:\n%q", body)
	}
}

// The email must not leak coach-facing diagnostics. sqi_score is never passed in
// and must never appear, nor any comparison with other students.
func TestComposeResultReleasedOmitsCoachingSignals(t *testing.T) {
	m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", "https://app.example")
	_, body := m.composeResultReleased("Bob", "Test", "tok", 12, 40, 30)

	for _, banned := range []string{"sqi", "SQI", "rank", "Rank", "percentile", "grade", "Grade", "pass/fail"} {
		if strings.Contains(body, banned) {
			t.Errorf("email leaked %q:\n%s", banned, body)
		}
	}
}

// The stated validity window must come from the same constant the release
// service uses for expires_at, or the student is told a different number than
// the token actually gets.
func TestResultLinkValidDaysMatchesEmailText(t *testing.T) {
	m := NewMailer("smtp.example", 587, "u", "p", "from@example.com", "https://app.example")
	_, body := m.composeResultReleased("Bob", "Test", "tok", 1, 10, 10)

	if ResultLinkValidDays != 15 {
		t.Fatalf("ResultLinkValidDays=%d, want 15", ResultLinkValidDays)
	}
	if !strings.Contains(body, "15 days") {
		t.Errorf("email does not state the 15-day window:\n%s", body)
	}
}

// TestSendResultReleasedUnconfiguredDoesNotDial mirrors the password-reset
// offline guarantee: with no SMTP credentials the mailer logs the link and
// returns nil without networking.
func TestSendResultReleasedUnconfiguredDoesNotDial(t *testing.T) {
	m := NewMailer("smtp.example", 587, "", "", "from@example.com", "https://app.example")
	if err := m.SendResultReleased("to@example.com", "Bob", "Test", "tok", 1, 10, 10); err != nil {
		t.Fatalf("unconfigured mailer must return nil, got %v", err)
	}
}

// A configured-but-unreachable host must surface an error so the caller can log
// it, rather than reporting a successful release notification that never sent.
func TestSendResultReleasedUnreachableHostSurfacesError(t *testing.T) {
	// Port 1 on localhost refuses connections quickly.
	m := NewMailer("127.0.0.1", 1, "user", "pass", "from@example.com", "https://app.example")
	if err := m.SendResultReleased("to@example.com", "Bob", "Test", "tok", 1, 10, 10); err == nil {
		t.Fatal("expected an error when SMTP is unreachable")
	}
}

// A malformed recipient must be rejected rather than placed in a header raw.
func TestSendResultReleasedRejectsMalformedRecipient(t *testing.T) {
	m := NewMailer("smtp.example", 587, "user", "pass", "from@example.com", "https://app.example")
	if err := m.SendResultReleased("not-an-address", "Bob", "Test", "tok", 1, 10, 10); err == nil {
		t.Fatal("expected an error for a malformed recipient address")
	}
}

// TestSendPasswordResetRejectsMalformedRecipient covers the same guard on the
// password-reset path, where the recipient comes straight from user input.
func TestSendPasswordResetRejectsMalformedRecipient(t *testing.T) {
	m := NewMailer("smtp.example", 587, "user", "pass", "from@example.com", "https://app.example")
	if err := m.SendPasswordReset("not-an-address", "tok"); err == nil {
		t.Fatal("expected an error for a malformed recipient address")
	}
}

// TestStripCRLF pins the header-injection guard directly.
func TestStripCRLF(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"a\r\nb", "a b"},
		{"a\rb", "a b"},
		{"a\nb", "a b"},
		// Two separate CRLF pairs collapse to two spaces; the point is that no
		// newline survives, not that the spacing is minimal.
		{"a\r\n\r\nBcc: x", "a  Bcc: x"},
		{"\r\n\r\n", "  "},
	}
	for _, c := range cases {
		if got := stripCRLF(c.in); got != c.want {
			t.Errorf("stripCRLF(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}