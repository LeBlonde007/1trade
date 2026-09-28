// Command test-idp is a SAML identity provider for LOCAL testing of 1Trade single sign-on (F02). It
// is not a real IdP: anyone can sign in as any email address. Never expose it beyond localhost.
//
//	go run ./scripts/dev/test-idp            # from the repo root; serves http://127.0.0.1:18090
//
// Paste http://127.0.0.1:18090/metadata's XML into /enterprise/sso. The signing key is kept in
// .dev/test-idp.pem (gitignored), so the metadata stays valid across restarts.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
)

// spMetadata fetches a service provider's metadata from its entity id, which for 1Trade is the
// tenant's metadata URL — so no SP has to be registered with this IdP up front.
type spMetadata struct{ client *http.Client }

// GetServiceProvider resolves the SP that sent an AuthnRequest. Only http(s) localhost entity ids are
// fetched: this tool must never be turned into a request forwarder.
func (s spMetadata) GetServiceProvider(r *http.Request, id string) (*saml.EntityDescriptor, error) {
	u, err := url.Parse(id)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("entity id %q is not an http(s) URL", id)
	}
	if h := u.Hostname(); h != "localhost" && h != "127.0.0.1" {
		return nil, fmt.Errorf("entity id host %q is not local", h)
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, id, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return samlsp.ParseMetadata(b)
}

// form asks which email to sign in as; it re-posts the AuthnRequest so the IdP can answer it.
var form = template.Must(template.New("f").Parse(`<!doctype html><meta charset="utf-8">
<title>Test IdP</title><body style="font:15px system-ui;max-width:420px;margin:60px auto">
<h2>Test identity provider</h2><p>Local testing only — sign in as any email.</p>
<form method="post" action="{{.Action}}">
<input type="hidden" name="SAMLRequest" value="{{.Req}}"><input type="hidden" name="RelayState" value="{{.Relay}}">
<p><input name="email" type="email" required autofocus placeholder="alice@example.com" style="width:100%;padding:8px"></p>
<p><button style="padding:8px 16px">Sign in</button></p></form>`))

// emailSessions signs in whoever types an email into the form.
type emailSessions struct{ action string }

// GetSession returns a session for the posted email, or shows the form and returns nil.
func (e emailSessions) GetSession(w http.ResponseWriter, r *http.Request, req *saml.IdpAuthnRequest) *saml.Session {
	if email := strings.TrimSpace(r.PostFormValue("email")); email != "" {
		return &saml.Session{ID: base64.RawURLEncoding.EncodeToString([]byte(email)), CreateTime: time.Now(),
			ExpireTime: time.Now().Add(time.Hour), NameID: email, NameIDFormat: string(saml.EmailAddressNameIDFormat), UserEmail: email}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := form.Execute(w, map[string]string{"Action": e.action,
		"Req": base64.StdEncoding.EncodeToString(req.RequestBuffer), "Relay": req.RelayState}); err != nil {
		log.Printf("form: %v", err)
	}
	return nil
}

// loadKey reads the IdP's signing key and certificate from path, creating them on first run.
func loadKey(path string) (*rsa.PrivateKey, *x509.Certificate, error) {
	if b, err := os.ReadFile(path); err == nil {
		var key *rsa.PrivateKey
		var cert *x509.Certificate
		for blk, rest := pem.Decode(b); blk != nil; blk, rest = pem.Decode(rest) {
			switch blk.Type {
			case "RSA PRIVATE KEY":
				key, err = x509.ParsePKCS1PrivateKey(blk.Bytes)
			case "CERTIFICATE":
				cert, err = x509.ParseCertificate(blk.Bytes)
			}
			if err != nil {
				return nil, nil, err
			}
		}
		if key == nil || cert == nil {
			return nil, nil, errors.New(path + ": missing key or certificate — delete it to regenerate")
		}
		return key, cert, nil
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "1trade-test-idp"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(5, 0, 0)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, _ := x509.ParseCertificate(der)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, err
	}
	out := append(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	return key, cert, os.WriteFile(path, out, 0o600)
}

// main serves the IdP's metadata and single sign-on endpoints on localhost.
func main() {
	addr := flag.String("addr", "127.0.0.1:18090", "listen address (keep it on localhost)")
	keyFile := flag.String("key", ".dev/test-idp.pem", "signing key + certificate (created on first run)")
	flag.Parse()
	key, cert, err := loadKey(*keyFile)
	if err != nil {
		log.Fatal(err)
	}
	base := "http://" + *addr
	md, _ := url.Parse(base + "/metadata")
	sso, _ := url.Parse(base + "/sso")
	idp := &saml.IdentityProvider{Key: key, Signer: key, Certificate: cert, MetadataURL: *md, SSOURL: *sso,
		ServiceProviderProvider: spMetadata{client: &http.Client{Timeout: 5 * time.Second}},
		SessionProvider:         emailSessions{action: sso.String()}}
	http.HandleFunc("/metadata", func(w http.ResponseWriter, _ *http.Request) {
		b, err := xml.MarshalIndent(idp.Metadata(), "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(b)
	})
	http.HandleFunc("/sso", idp.ServeSSO)
	log.Printf("test IdP on %s — metadata: %s", base, md)
	srv := &http.Server{Addr: *addr, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
