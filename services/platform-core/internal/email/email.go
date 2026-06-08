// Package email sends transactional email (today: the email-verification link). It speaks SMTP and
// supports three transports so the same code serves local dev and a real provider:
//   - plain (no TLS, no auth)      → Mailpit locally / CI;
//   - implicit TLS (SMTPS, :465)   → providers like PrivateEmail/Namecheap;
//   - STARTTLS (:587)              → most relays (SendGrid/SES/Gmail).
//
// When no SMTP address is configured it is a safe no-op, so the platform still runs without a mail
// server. Credentials are only ever sent over a TLS connection.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Sender delivers transactional emails over SMTP.
type Sender struct {
	addr string // SMTP server host:port; empty disables sending (Send becomes a no-op)
	from string // From address
	user string // SMTP AUTH username; empty → no auth (e.g. Mailpit)
	pass string // SMTP AUTH password
	tls  string // transport: "implicit" (SMTPS/465) | "starttls" (587) | "" / "none" (plain, e.g. Mailpit)
}

// New builds a Sender. A blank addr makes every Send a no-op that returns nil. tlsMode selects the
// transport ("implicit"/"starttls"/""); when blank it is inferred from the port (465→implicit,
// 587→starttls, else plain).
func New(addr, from, user, pass, tlsMode string) Sender {
	if from == "" {
		from = "noreply@exascale.local"
	}
	tlsMode = strings.ToLower(strings.TrimSpace(tlsMode))
	if tlsMode == "" && addr != "" {
		if _, port, err := net.SplitHostPort(addr); err == nil {
			switch port {
			case "465":
				tlsMode = "implicit"
			case "587":
				tlsMode = "starttls"
			}
		}
	}
	return Sender{addr: addr, from: from, user: user, pass: pass, tls: tlsMode}
}

// Enabled reports whether an SMTP server is configured (i.e. mail will actually be sent).
func (s Sender) Enabled() bool { return s.addr != "" }

// SendVerification emails an email-verification link. No-op (nil) when no SMTP server is configured.
func (s Sender) SendVerification(to, link string) error {
	if s.addr == "" {
		return nil
	}
	body := "Welcome to Exascale.\r\n\r\n" +
		"Confirm your email to finish setting up your account:\r\n" + link + "\r\n\r\n" +
		"If you didn't create an account, you can ignore this message.\r\n"
	msg := buildMessage(s.from, to, "Verify your Exascale email", body)
	return s.send(to, msg)
}

// send delivers msg to one recipient over the configured transport, running AUTH when credentials are
// set. Implicit TLS dials a TLS socket up front; STARTTLS upgrades a plain socket; otherwise it speaks
// plain SMTP (Mailpit). PlainAuth refuses to send credentials over a cleartext link, so auth + a
// non-TLS transport will (correctly) error rather than leak the password.
func (s Sender) send(to string, msg []byte) error {
	host, _, err := net.SplitHostPort(s.addr)
	if err != nil {
		return fmt.Errorf("smtp addr %q: %w", s.addr, err)
	}
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, host)
	}

	switch s.tls {
	case "implicit", "ssl", "tls", "smtps":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		d := tls.Dialer{Config: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}}
		conn, err := d.DialContext(ctx, "tcp", s.addr)
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		c, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("smtp client: %w", err)
		}
		defer func() { _ = c.Close() }()
		return deliver(c, auth, s.from, to, msg)
	case "starttls":
		c, err := smtp.Dial(s.addr)
		if err != nil {
			return fmt.Errorf("smtp dial: %w", err)
		}
		defer func() { _ = c.Close() }()
		if err := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
		return deliver(c, auth, s.from, to, msg)
	default:
		return smtp.SendMail(s.addr, auth, s.from, []string{to}, msg)
	}
}

// deliver runs AUTH (when provided) then the MAIL/RCPT/DATA sequence on an established SMTP client.
func deliver(c *smtp.Client, auth smtp.Auth, from, to string, msg []byte) error {
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return c.Quit()
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
