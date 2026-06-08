package email

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

// fakeSMTP starts a minimal in-process SMTP server that accepts one message and returns its address
// plus a channel that yields the received DATA body. It speaks just enough of RFC 5321 (EHLO/MAIL/
// RCPT/DATA/QUIT) for net/smtp's plain path — no TLS, no auth (the Mailpit shape).
func fakeSMTP(t *testing.T) (string, chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)
		reply := func(s string) { _, _ = w.WriteString(s + "\r\n"); _ = w.Flush() }
		reply("220 fake ESMTP")
		var body strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if strings.TrimRight(line, "\r\n") == "." {
					inData = false
					reply("250 OK")
					got <- body.String()
					continue
				}
				body.WriteString(line)
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				reply("250-fake\r\n250 OK")
			case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
				reply("250 OK")
			case strings.HasPrefix(line, "DATA"):
				reply("354 end with .")
				inData = true
			case strings.HasPrefix(line, "QUIT"):
				reply("221 bye")
				return
			default:
				reply("250 OK")
			}
		}
	}()
	return ln.Addr().String(), got
}

// TestVerifyURL checks link construction, trailing-slash trim, and the empty-base fallback.
func TestVerifyURL(t *testing.T) {
	cases := []struct{ base, token, want string }{
		{"http://localhost:3000", "abc", "http://localhost:3000/onboarding/verify?token=abc"},
		{"http://localhost:3000/", "abc", "http://localhost:3000/onboarding/verify?token=abc"},
		{"", "xyz", "http://localhost:3000/onboarding/verify?token=xyz"},
	}
	for _, c := range cases {
		if got := VerifyURL(c.base, c.token); got != c.want {
			t.Errorf("VerifyURL(%q,%q) = %q, want %q", c.base, c.token, got, c.want)
		}
	}
}

// TestDisabledSenderNoop confirms a Sender with no SMTP address sends nothing and returns nil, so the
// platform runs without a mail server.
func TestDisabledSenderNoop(t *testing.T) {
	s := New("", "", "", "", "")
	if s.Enabled() {
		t.Fatal("Enabled() should be false with no SMTP address")
	}
	if err := s.SendVerification("a@b.com", "http://x/verify?token=1"); err != nil {
		t.Fatalf("disabled SendVerification should be a nil no-op, got %v", err)
	}
}

// TestTLSModeInferredFromPort checks the transport is inferred from the port when not set explicitly:
// 465 → implicit TLS, 587 → STARTTLS, anything else (e.g. Mailpit :1025) → plain.
func TestTLSModeInferredFromPort(t *testing.T) {
	cases := []struct{ addr, want string }{
		{"mail.privateemail.com:465", "implicit"},
		{"smtp.example.com:587", "starttls"},
		{"mailpit.data.svc.cluster.local:1025", ""},
	}
	for _, c := range cases {
		if got := New(c.addr, "from@x", "u", "p", "").tls; got != c.want {
			t.Errorf("New(%q).tls = %q, want %q", c.addr, got, c.want)
		}
	}
	// An explicit mode always wins over the port inference.
	if got := New("mail.x:465", "from@x", "u", "p", "starttls").tls; got != "starttls" {
		t.Errorf("explicit tlsMode should win, got %q", got)
	}
}

// TestPlainDelivery exercises the no-TLS, no-auth path (the Mailpit shape) against an in-process fake
// SMTP server, asserting the MAIL/RCPT/DATA conversation completes and the message body is delivered.
func TestPlainDelivery(t *testing.T) {
	addr, got := fakeSMTP(t)
	s := New(addr, "from@exascale.ai", "", "", "")
	if err := s.SendVerification("user@example.com", "http://app/verify?token=tok123"); err != nil {
		t.Fatalf("SendVerification: %v", err)
	}
	if !strings.Contains(<-got, "tok123") {
		t.Fatal("delivered message did not contain the verification link")
	}
}
