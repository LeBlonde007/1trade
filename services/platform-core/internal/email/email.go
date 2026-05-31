// Package email sends transactional email (today: the email-verification link). It speaks plain SMTP
// so it works with Mailpit locally and any SMTP relay in prod. When no SMTP address is configured it
// is a safe no-op, so the platform still runs without a mail server (e.g. CI).
package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Sender delivers transactional emails over SMTP.
type Sender struct {
	addr string // SMTP server host:port; empty disables sending (Send becomes a no-op)
	from string // From address
}

// New builds a Sender. A blank addr makes every Send a no-op that returns nil.
func New(addr, from string) Sender {
	if from == "" {
		from = "noreply@exascale.local"
	}
	return Sender{addr: addr, from: from}
}

// Enabled reports whether an SMTP server is configured (i.e. mail will actually be sent).
func (s Sender) Enabled() bool { return s.addr != "" }

// SendVerification emails an email-verification link. No-op (nil) when no SMTP server is configured.
// Mailpit (and most dev relays) accept plain SMTP with no auth.
func (s Sender) SendVerification(to, link string) error {
	if s.addr == "" {
		return nil
	}
	body := "Welcome to Exascale.\r\n\r\n" +
		"Confirm your email to finish setting up your account:\r\n" + link + "\r\n\r\n" +
		"If you didn't create an account, you can ignore this message.\r\n"
	msg := buildMessage(s.from, to, "Verify your Exascale email", body)
	return smtp.SendMail(s.addr, nil, s.from, []string{to}, msg)
}

// buildMessage assembles a minimal RFC 5322 plain-text message.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// VerifyURL builds the web app's verification link for a raw token (consumed by /onboarding/verify).
func VerifyURL(baseURL, token string) string {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	return fmt.Sprintf("%s/onboarding/verify?token=%s", base, token)
}
