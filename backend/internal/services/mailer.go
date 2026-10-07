package services

import (
	"fmt"
	"log"
	"net/smtp"
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

	if m.AppPass == "" || m.User == "" {
		log.Printf("[MAIL] SMTP not configured; password-reset link for %s: %s", to, link)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	msg := []byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\n%s", to, m.From, subject, body))
	auth := smtp.PlainAuth("", m.User, m.AppPass, m.Host)
	if err := smtp.SendMail(addr, auth, m.From, []string{to}, msg); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}
