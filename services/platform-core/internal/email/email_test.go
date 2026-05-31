package email

import "testing"

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
	s := New("", "")
	if s.Enabled() {
		t.Fatal("Enabled() should be false with no SMTP address")
	}
	if err := s.SendVerification("a@b.com", "http://x/verify?token=1"); err != nil {
		t.Fatalf("disabled SendVerification should be a nil no-op, got %v", err)
	}
}
