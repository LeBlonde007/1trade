package config

import "testing"

// TestSkipDomainDNS proves the dev DNS bypass needs both the flag and the dev environment, so setting
// SSO_DEV_SKIP_DNS in staging or prod cannot weaken SSO domain verification.
func TestSkipDomainDNS(t *testing.T) {
	cases := []struct {
		env  string
		flag bool
		want bool
	}{
		{"dev", true, true},
		{"dev", false, false},
		{"staging", true, false},
		{"prod", true, false},
		{"", true, false},
	}
	for _, c := range cases {
		if got := (Config{Env: c.env, SSODevSkipDNS: c.flag}).SkipDomainDNS(); got != c.want {
			t.Errorf("env=%q flag=%v: SkipDomainDNS()=%v, want %v", c.env, c.flag, got, c.want)
		}
	}
}

// TestLoadSSODevSkipDNS proves the flag is read from SSO_DEV_SKIP_DNS and defaults off.
func TestLoadSSODevSkipDNS(t *testing.T) {
	t.Setenv("TRADE1_ENV", "dev")
	if Load().SSODevSkipDNS {
		t.Fatal("flag must default off")
	}
	t.Setenv("SSO_DEV_SKIP_DNS", "1")
	if !Load().SkipDomainDNS() {
		t.Fatal("SSO_DEV_SKIP_DNS=1 in dev must skip the DNS proof")
	}
	t.Setenv("TRADE1_ENV", "prod")
	if Load().SkipDomainDNS() {
		t.Fatal("prod must never skip the DNS proof")
	}
}
