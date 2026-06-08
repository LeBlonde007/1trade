// Package email sends transactional email (today: the email-verification link). It supports two
// transports behind one Sender so the same code serves local dev and a real provider:
//
//   - SMTP — plain (Mailpit/CI), implicit TLS (:465, PrivateEmail), or STARTTLS (:587, SES/Gmail);
//   - HTTP Email API (Mailtrap-style JSON over HTTPS/443) — the right choice when the host blocks the
//     SMTP egress ports (most cloud VPS block 25, and often 465/587), since 443 is never blocked.
//
// The API transport wins when an API URL is configured; otherwise SMTP; with neither it is a safe
// no-op so the platform still runs without a mail server. Credentials only ever leave over TLS/HTTPS.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Config selects the transport and credentials for a Sender. Set APIURL for the HTTP Email API
// (Mailtrap etc.); otherwise the SMTP fields are used.
type Config struct {
	From     string // From address
	Addr     string // SMTP host:port (used when APIURL is empty)
	User     string // SMTP AUTH username; empty → no auth (e.g. Mailpit)
	Pass     string // SMTP AUTH password
	TLS      string // SMTP transport: "implicit" (465) | "starttls" (587) | "" (plain; inferred from port)
	APIURL   string // HTTP Email-API send URL (e.g. https://send.api.mailtrap.io/api/send) — preferred when set
	APIToken string // bearer token for the HTTP Email API
}

// Sender delivers transactional emails over SMTP or an HTTP Email API.
type Sender struct {
	from     string
	addr     string
	user     string
	pass     string
	tls      string
	apiURL   string
	apiToken string
	http     *http.Client
}

// New builds a Sender from config. With no Addr and no APIURL every Send is a no-op that returns nil.
// When TLS is blank it is inferred from the SMTP port (465→implicit, 587→starttls, else plain).
func New(c Config) Sender {
	from := c.From
	if from == "" {
		from = "noreply@exascale.local"
	}
	tlsMode := strings.ToLower(strings.TrimSpace(c.TLS))
	if tlsMode == "" && c.Addr != "" {
		if _, port, err := net.SplitHostPort(c.Addr); err == nil {
			switch port {
			case "465":
				tlsMode = "implicit"
			case "587":
				tlsMode = "starttls"
			}
		}
	}
	return Sender{
		from: from, addr: c.Addr, user: c.User, pass: c.Pass, tls: tlsMode,
		apiURL: strings.TrimSpace(c.APIURL), apiToken: c.APIToken,
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// Enabled reports whether a transport is configured (i.e. mail will actually be sent).
func (s Sender) Enabled() bool { return s.apiURL != "" || s.addr != "" }

// SendVerification emails an email-verification link over whichever transport is configured (HTTP API
// preferred, then SMTP). No-op (nil) when none is configured.
func (s Sender) SendVerification(to, link string) error {
	if !s.Enabled() {
		return nil
	}
	const subject = "Verify your Exascale email"
	body := "Welcome to Exascale.\r\n\r\n" +
		"Confirm your email to finish setting up your account:\r\n" + link + "\r\n\r\n" +
		"If you didn't create an account, you can ignore this message.\r\n"
	if s.apiURL != "" {
		return s.sendAPI(to, subject, body)
	}
	return s.send(to, buildMessage(s.from, to, subject, body))
}

// mailtrapPayload is the JSON body of the Mailtrap-style Email API (also matches several other HTTP
// providers): a structured from/to plus subject and a text body.
type mailtrapPayload struct {
	From     addr   `json:"from"`
	To       []addr `json:"to"`
	Subject  string `json:"subject"`
	Text     string `json:"text"`
	Category string `json:"category,omitempty"`
}

// addr is one {email,name} entry in the Email-API payload.
type addr struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// sendAPI delivers a message via the HTTP Email API (HTTPS/443) — used when SMTP egress is blocked. A
// non-2xx is surfaced with the provider's response body (truncated) so misconfig is visible in logs.
func (s Sender) sendAPI(to, subject, body string) error {
	payload, _ := json.Marshal(mailtrapPayload{
		From:     addr{Email: s.from, Name: "Exascale"},
		To:       []addr{{Email: to}},
		Subject:  subject,
		Text:     body,
		Category: "verification",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("email api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("email api call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("email api returned %d: %s", resp.StatusCode, bytes.TrimSpace(excerpt))
	}
	return nil
}

// send delivers msg to one recipient over the configured SMTP transport, running AUTH when credentials
// are set. Implicit TLS dials a TLS socket up front; STARTTLS upgrades a plain socket; otherwise it
// speaks plain SMTP (Mailpit). PlainAuth refuses to send credentials over a cleartext link.
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
