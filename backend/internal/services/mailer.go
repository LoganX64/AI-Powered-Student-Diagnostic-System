package services

import (
	"fmt"
	"log"
	"mime"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
)

type Mailer struct {
	Host     string
	Port     int
	User     string
	AppPass  string
	From     string
	Frontend string
}

func NewMailer(host string, port int, user, appPass, from, frontend string) *Mailer {
	return &Mailer{Host: host, Port: port, User: user, AppPass: appPass, From: from, Frontend: frontend}
}

// configured reports whether SMTP credentials are present. Both senders use it
// to decide between delivering and logging, so an unconfigured deployment never
// attempts a network call.
func (m *Mailer) configured() bool {
	return m.AppPass != "" && m.User != ""
}

// send delivers one message, or returns an error the caller may ignore.
//
// Headers are built through net/mail and mime rather than interpolated into a
// string. Recipient addresses and the test title are coach- or user-supplied,
// and a raw value containing CRLF would otherwise let a caller append headers of
// its own. The subject is RFC 2047 encoded so a non-ASCII test title survives
// transport rather than being sent raw.
func (m *Mailer) send(to, subject, body string) error {
	from, err := mail.ParseAddress(m.From)
	if err != nil {
		return fmt.Errorf("parse from address: %w", err)
	}
	toAddr, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("parse to address: %w", err)
	}

	encodedSubject := mime.QEncoding.Encode("UTF-8", stripCRLF(subject))

	msg := strings.Join([]string{
		"To: " + toAddr.String(),
		"From: " + from.String(),
		"Subject: " + encodedSubject,
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	if err := smtp.SendMail(addr, smtp.PlainAuth("", m.User, m.AppPass, m.Host), from.Address, []string{toAddr.Address}, []byte(msg)); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}

// stripCRLF removes any CR or LF so a value cannot terminate the header it
// appears in and start a new one. A CRLF pair collapses to a single space
// rather than two, so the sanitised text reads naturally.
func stripCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

// resetLink builds the password-reset URL. Kept separate from the SMTP call so
// the link construction (notably the trailing-slash trim) is testable without a
// mail server.
func (m *Mailer) resetLink(token string) string {
	return fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(m.Frontend, "/"), token)
}

// compose builds the subject and body of the reset email.
func (m *Mailer) compose(token string) (subject, body string) {
	subject = "EduQuant password reset"
	body = fmt.Sprintf("Hello,\n\nA password reset was requested for your account. Click the link below (valid for 30 minutes):\n\n%s\n\nIf you did not request this, you can ignore this email.\n\n- EduQuant", m.resetLink(token))
	return subject, body
}

func (m *Mailer) SendPasswordReset(to, token string) error {
	link := m.resetLink(token)
	subject, body := m.compose(token)

	if !m.configured() {
		log.Printf("[MAIL] SMTP not configured; password-reset link for %s: %s", to, link)
		return nil
	}
	return m.send(to, subject, body)
}

// ResultLinkValidDays is how long a released-result link stays usable. Kept
// beside the email that carries it so the number a student reads matches the
// value the release service actually stamps into result_tokens.expires_at.
const ResultLinkValidDays = 15

// resultLink builds the student results URL for a token.
func (m *Mailer) resultLink(token string) string {
	return fmt.Sprintf("%s/student/results/%s", strings.TrimRight(m.Frontend, "/"), token)
}

// formatScore renders a score without trailing zeros: 12.5 stays "12.5"
// rather than becoming "12.50", matching how the score is stored.
func formatScore(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// composeResultReleased builds the subject and body of the results email.
//
// It states only what the system knows: the score, and that the link expires.
// Nothing about ranking, grading bands, or how the student did against others.
func (m *Mailer) composeResultReleased(studentName, testTitle, token string, marks, maxMarks, percent float64) (subject, body string) {
	name := strings.TrimSpace(studentName)
	if name == "" {
		name = "there"
	}
	// The test title is coach-supplied, so it is untrusted. It is sanitised once
	// here and used for both the subject and the body, so neither can carry a
	// newline out of its field.
	test := stripCRLF(strings.TrimSpace(testTitle))

	subject = "Your result is available"
	if test != "" {
		subject = "Your result for " + test + " is available"
	}

	headline := "Your result is now available."
	if test != "" {
		headline = fmt.Sprintf("Your result for %s is now available.", test)
	}

	body = fmt.Sprintf(
		"Hello %s,\n\n%s\n\nYou scored %s out of %s (%s%%).\n\nView your result:\n%s\n\nThis link is valid for %d days.\n\n- EduQuant",
		name,
		headline,
		formatScore(marks),
		formatScore(maxMarks),
		formatScore(percent),
		m.resultLink(token),
		ResultLinkValidDays,
	)
	return subject, body
}

// SendResultReleased emails a student the link to their released result.
//
// Fire-and-forget: the caller logs the error and continues. A result that has
// been released must not appear un-released because a mail server was down.
// With SMTP unconfigured the link is logged so a coach can copy it in
// development.
func (m *Mailer) SendResultReleased(to, studentName, testTitle, token string, marks, maxMarks, percent float64) error {
	subject, body := m.composeResultReleased(studentName, testTitle, token, marks, maxMarks, percent)

	if !m.configured() {
		log.Printf("[MAIL] SMTP not configured; result link for %s: %s", to, m.resultLink(token))
		return nil
	}
	return m.send(to, subject, body)
}