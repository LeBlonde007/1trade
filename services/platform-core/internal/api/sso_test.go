package api_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/xml"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trade1/platform-core/internal/domain"
)

// testIdP is a real SAML identity provider (crewjam) with its own signing key.
type testIdP struct {
	idp *saml.IdentityProvider
	rg  *teamRig
}

// newTestIdP builds an IdP whose service-provider lookups read our metadata endpoint.
func newTestIdP(t *testing.T, rg *teamRig, host string) *testIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: host},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	md, _ := url.Parse("https://" + host + "/metadata")
	sso, _ := url.Parse("https://" + host + "/sso")
	ti := &testIdP{rg: rg}
	ti.idp = &saml.IdentityProvider{Key: key, Signer: key, Certificate: cert, MetadataURL: *md, SSOURL: *sso,
		ServiceProviderProvider: ti}
	return ti
}

// GetServiceProvider fetches our service-provider metadata (the entity id is its URL on the web app).
func (ti *testIdP) GetServiceProvider(_ *http.Request, id string) (*saml.EntityDescriptor, error) {
	tenant := id[strings.LastIndex(id, "/")+1:]
	resp, err := http.Get(ti.rg.srv.URL + "/v1/auth/sso/metadata/" + tenant)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return samlsp.ParseMetadata(b)
}

// metadataXML is the IdP's metadata document.
func (ti *testIdP) metadataXML() string {
	b, _ := xml.Marshal(ti.idp.Metadata())
	return string(b)
}

// respond plays the IdP: it answers the AuthnRequest in redirectURL with a signed assertion for email.
func (ti *testIdP) respond(t *testing.T, redirectURL, email string) string {
	t.Helper()
	req := httptest.NewRequest("GET", redirectURL, nil)
	ir, err := saml.NewIdpAuthnRequest(ti.idp, req)
	if err != nil {
		t.Fatal(err)
	}
	if err := ir.Validate(); err != nil {
		t.Fatalf("the IdP rejected our AuthnRequest: %v", err)
	}
	sess := &saml.Session{ID: uuid.NewString(), CreateTime: time.Now(), ExpireTime: time.Now().Add(time.Hour),
		NameID: email, NameIDFormat: string(saml.EmailAddressNameIDFormat), UserEmail: email}
	if err := (saml.DefaultAssertionMaker{}).MakeAssertion(ir, sess); err != nil {
		t.Fatal(err)
	}
	if err := ir.MakeResponse(); err != nil {
		t.Fatal(err)
	}
	doc := etree.NewDocument()
	doc.SetRoot(ir.ResponseEl)
	b, _ := doc.WriteToBytes()
	return base64.StdEncoding.EncodeToString(b)
}

// verifyDomain publishes the tenant's TXT proof for dom and verifies it; it returns the status code.
func (rg *teamRig) verifyDomain(t *testing.T, admin, dom string) (int, map[string]any) {
	t.Helper()
	_, cfg := rg.do("GET", "/v1/account/sso", admin, nil, nil)
	for _, p := range cfg["pending_domains"].([]any) {
		m := p.(map[string]any)
		if m["domain"] == dom {
			rg.txt[m["txt_name"].(string)] = append(rg.txt[m["txt_name"].(string)], m["txt_value"].(string))
		}
	}
	return rg.do("POST", "/v1/account/sso/domains/"+dom+"/verify", admin, nil, nil)
}

// start begins SSO for an email and returns the IdP redirect URL and the relay state.
func (rg *teamRig) ssoStart(t *testing.T, email string) (string, string) {
	t.Helper()
	code, out := rg.do("POST", "/v1/auth/sso/start", "", map[string]string{"email": email}, nil)
	if code != 200 {
		t.Fatalf("sso start: %d %v", code, out)
	}
	u, _ := url.Parse(out["redirect_url"].(string))
	return out["redirect_url"].(string), u.Query().Get("RelayState")
}

// acs posts an IdP response to the ACS.
func (rg *teamRig) acs(resp, relay string) (int, map[string]any) {
	return rg.do("POST", "/v1/auth/sso/acs", "", map[string]string{"saml_response": resp, "relay_state": relay}, nil)
}

// TestSAMLSignIn: a signed assertion answering our own request signs a member in once, created
// just-in-time; replays, other requests, tampering, other IdPs and other domains are refused.
func TestSAMLSignIn(t *testing.T) {
	rg := newTeamRig(t)
	admin, ac := rg.signup()
	dom := "acme-" + uuid.NewString()[:8] + ".com"
	ti := newTestIdP(t, rg, "idp."+dom)
	body := map[string]any{"idp_metadata_xml": ti.metadataXML(), "email_domains": []string{strings.ToUpper(dom)}, "default_role": "engineer"}
	code, cfg := rg.do("PUT", "/v1/account/sso", admin, body, nil)
	if code != 200 || cfg["configured"] != true || cfg["email_domains"].([]any)[0] != dom || len(cfg["verified_domains"].([]any)) != 0 {
		t.Fatalf("configure: %d %v", code, cfg)
	}
	// An unverified domain routes nothing, and a missing TXT record does not verify it.
	if code, _ := rg.do("POST", "/v1/auth/sso/start", "", map[string]string{"email": "alice@" + dom}, nil); code != 404 {
		t.Fatalf("start on an unverified domain: %d", code)
	}
	if code, _ := rg.do("POST", "/v1/account/sso/domains/"+dom+"/verify", admin, nil, nil); code != 422 {
		t.Fatalf("verify without the TXT record: %d", code)
	}
	rg.txt["_1trade-verify."+dom] = []string{"1trade-verify=someone-elses-token"}
	if code, _ := rg.do("POST", "/v1/account/sso/domains/"+dom+"/verify", admin, nil, nil); code != 422 {
		t.Fatalf("verify with a wrong TXT value: %d", code)
	}
	if code, out := rg.verifyDomain(t, admin, dom); code != 200 || out["verified_domains"].([]any)[0] != dom {
		t.Fatalf("verify: %d %v", code, out)
	}

	alice := "alice@" + dom
	redirect, relay := rg.ssoStart(t, alice)
	resp := ti.respond(t, redirect, alice)
	code, out := rg.acs(resp, relay)
	if code != 200 {
		t.Fatalf("acs: %d %v", code, out)
	}
	claims, err := domain.VerifyToken(jwtSecret, out["token"].(string))
	if err != nil || claims.TenantID != ac.TenantID || claims.Roles[0] != domain.RoleEngineer {
		t.Fatalf("claims: %+v %v", claims, err)
	}
	if code, _ := rg.acs(resp, relay); code != 401 {
		t.Fatalf("replayed response: %d", code)
	}
	// A response to one request cannot be used for another.
	_, relay2 := rg.ssoStart(t, alice)
	if code, _ := rg.acs(resp, relay2); code != 401 {
		t.Fatalf("response for another request: %d", code)
	}
	// Tampering with the signed assertion breaks it.
	redirect, relay = rg.ssoStart(t, alice)
	raw, _ := base64.StdEncoding.DecodeString(ti.respond(t, redirect, alice))
	forged := strings.ReplaceAll(string(raw), alice, "mallory@"+dom)
	if code, _ := rg.acs(base64.StdEncoding.EncodeToString([]byte(forged)), relay); code != 401 {
		t.Fatalf("tampered assertion: %d", code)
	}
	// Another IdP's signature is not trusted.
	rogue := newTestIdP(t, rg, "idp."+dom)
	redirect, relay = rg.ssoStart(t, alice)
	if code, _ := rg.acs(rogue.respond(t, redirect, alice), relay); code != 401 {
		t.Fatalf("assertion from an untrusted IdP: %d", code)
	}
	// An email outside the tenant's domains is refused.
	redirect, relay = rg.ssoStart(t, alice)
	if code, out := rg.acs(ti.respond(t, redirect, "eve@evil.example"), relay); code != 401 || out["code"] != "sso_domain" {
		t.Fatalf("foreign domain: %d %v", code, out)
	}
	// A request older than ten minutes has expired.
	redirect, relay = rg.ssoStart(t, alice)
	late := ti.respond(t, redirect, alice)
	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(context.Background(), `UPDATE sso_requests SET created_at = now() - interval '11 minutes' WHERE id=$1`, relay)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := rg.acs(late, relay); code != 401 {
		t.Fatalf("expired request: %d", code)
	}
	// A second domain the tenant claimed but never proved does not sign anyone in, even through the
	// tenant's own IdP.
	pending := "pending-" + uuid.NewString()[:8] + ".com"
	body["email_domains"] = []string{dom, pending}
	if code, _ := rg.do("PUT", "/v1/account/sso", admin, body, nil); code != 200 {
		t.Fatal("add a pending domain")
	}
	redirect, relay = rg.ssoStart(t, alice)
	if code, out := rg.acs(ti.respond(t, redirect, "bob@"+pending), relay); code != 401 || out["code"] != "sso_domain" {
		t.Fatalf("assertion for an unverified domain: %d %v", code, out)
	}
	// The second sign-in finds the same member.
	redirect, relay = rg.ssoStart(t, alice)
	_, out = rg.acs(ti.respond(t, redirect, alice), relay)
	c2, _ := domain.VerifyToken(jwtSecret, out["token"].(string))
	if c2 == nil || c2.Subject != claims.Subject {
		t.Fatal("second sign-in created another member")
	}
	// The member cannot sign in with a password (they have none).
	if code, _ := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": alice, "password": "!"}, nil); code != 401 {
		t.Fatalf("password sign-in for an SSO member: %d", code)
	}
	_, audit := rg.do("GET", "/v1/account/audit", admin, nil, nil)
	seen := map[string]bool{}
	for _, e := range audit["entries"].([]any) {
		seen[e.(map[string]any)["action"].(string)] = true
	}
	if !seen["sso.config.update"] || !seen["sso.user.create"] || !seen["sso.login"] {
		t.Fatalf("audit: %v", seen)
	}
	if code, _ := rg.do("POST", "/v1/auth/sso/start", "", map[string]string{"email": "x@unknown-" + uuid.NewString()[:6] + ".com"}, nil); code != 404 {
		t.Fatalf("unknown domain: %d", code)
	}
}

// TestSAMLPolicy: accounts elsewhere are never merged, JIT can be off, enforcement blocks passwords for
// non-admins, domains are exclusive, and only admins configure with valid input.
func TestSAMLPolicy(t *testing.T) {
	rg := newTeamRig(t)
	admin, _ := rg.signup()
	dom := "corp-" + uuid.NewString()[:8] + ".com"
	ti := newTestIdP(t, rg, "idp."+dom)
	// Someone with this domain already has an account in another tenant.
	other := "bob@" + dom
	rg.do("POST", "/v1/auth/signup", "", map[string]string{"email": other, "password": "pw-123456"}, nil)
	f := false
	put := func(tok string, b map[string]any) (int, map[string]any) {
		return rg.do("PUT", "/v1/account/sso", tok, b, nil)
	}
	base := func() map[string]any {
		return map[string]any{"idp_metadata_xml": ti.metadataXML(), "email_domains": []string{dom}, "jit": &f}
	}
	if code, _ := put(admin, base()); code != 200 {
		t.Fatalf("configure: %d", code)
	}
	if code, _ := rg.verifyDomain(t, admin, dom); code != 200 {
		t.Fatalf("verify: %d", code)
	}
	redirect, relay := rg.ssoStart(t, other)
	if code, out := rg.acs(ti.respond(t, redirect, other), relay); code != 403 || out["code"] != "sso_account_conflict" {
		t.Fatalf("account in another tenant: %d %v", code, out)
	}
	carol := "carol@" + dom
	redirect, relay = rg.ssoStart(t, carol)
	if code, out := rg.acs(ti.respond(t, redirect, carol), relay); code != 403 || out["code"] != "sso_no_account" {
		t.Fatalf("jit off: %d %v", code, out)
	}
	// Enforcement: an invited viewer can no longer use a password; the admin still can.
	dave := "dave+" + uuid.NewString()[:6] + "@acme.ai"
	rg.accept(rg.invite(admin, dave, []string{"viewer"}, nil))
	b := base()
	b["enforce"] = true
	if code, _ := put(admin, b); code != 200 {
		t.Fatalf("enforce: %d", code)
	}
	if code, out := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": dave, "password": "correct-horse"}, nil); code != 403 || out["code"] != "sso_required" {
		t.Fatalf("password sign-in under enforcement: %d %v", code, out)
	}
	// Another tenant may claim the domain, but cannot verify it (and so never receives its sign-ins).
	admin2, _ := rg.signup()
	if code, _ := put(admin2, base()); code != 200 {
		t.Fatalf("a pending claim by another tenant: %d", code)
	}
	if code, out := rg.verifyDomain(t, admin2, dom); code != 409 || out["code"] != "domain_taken" {
		t.Fatalf("second tenant verifying a verified domain: %d %v", code, out)
	}
	for name, mut := range map[string]func(map[string]any){
		"bad metadata": func(m map[string]any) { m["idp_metadata_xml"] = "<x/>" },
		"admin role":   func(m map[string]any) { m["default_role"] = "admin" },
		"bad domain":   func(m map[string]any) { m["email_domains"] = []string{"not a domain"} },
		"no domain":    func(m map[string]any) { m["email_domains"] = []string{} },
		"javascript sso url": func(m map[string]any) {
			m["idp_metadata_xml"] = strings.ReplaceAll(m["idp_metadata_xml"].(string), "https://idp."+dom+"/sso", "javascript:alert(document.domain)//")
		},
	} {
		m := base()
		mut(m)
		if code, _ := put(admin, m); code != 422 {
			t.Errorf("%s: %d, want 422", name, code)
		}
	}
	viewerTok, _ := rg.accept(rg.invite(admin, "v+"+uuid.NewString()[:6]+"@acme.ai", []string{"viewer"}, nil))
	if code, _ := put(viewerTok, base()); code != 403 {
		t.Fatalf("viewer configures SSO: %d", code)
	}
	if code, _ := rg.do("DELETE", "/v1/account/sso", admin, nil, nil); code != 204 {
		t.Fatalf("delete: %d", code)
	}
	if code, _ := rg.do("POST", "/v1/auth/login", "", map[string]string{"email": dave, "password": "correct-horse"}, nil); code != 200 {
		t.Fatalf("password sign-in after SSO removed: %d", code)
	}
}
