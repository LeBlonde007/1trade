package api

import (
	"testing"

	"github.com/trade1/platform-core/internal/config"
)

// TestSSOURLAllowed: the browser is only ever sent to an https IdP (plain http in dev only), never a
// javascript:, data: or relative URL.
func TestSSOURLAllowed(t *testing.T) {
	prod, dev := &Server{cfg: config.Config{Env: "prod"}}, &Server{cfg: config.Config{Env: "dev"}}
	for _, c := range []struct {
		url       string
		prod, dev bool
	}{
		{"https://idp.example.com/sso?SAMLRequest=x", true, true},
		{"http://127.0.0.1:18090/sso", false, true},
		{"javascript:alert(1)//", false, false},
		{"data:text/html,<script>alert(1)</script>", false, false},
		{"/sso", false, false},
		{"https:///nohost", false, false},
	} {
		if got := prod.ssoURLAllowed(c.url); got != c.prod {
			t.Errorf("prod %q: %v", c.url, got)
		}
		if got := dev.ssoURLAllowed(c.url); got != c.dev {
			t.Errorf("dev %q: %v", c.url, got)
		}
	}
}
