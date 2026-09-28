package api

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"

	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
)

// ssoRoutes registers SAML single sign-on (platform-core.yaml v1.8).
func (s *Server) ssoRoutes() {
	s.mux.HandleFunc("GET /v1/account/sso", s.getSSO)
	s.mux.HandleFunc("PUT /v1/account/sso", s.putSSO)
	s.mux.HandleFunc("DELETE /v1/account/sso", s.deleteSSO)
	s.mux.HandleFunc("POST /v1/account/sso/domains/{domain}/verify", s.verifySSODomain)
	s.mux.HandleFunc("GET /v1/auth/sso/metadata/{tenant_id}", s.ssoMetadata) // public: the IdP reads it
	s.mux.HandleFunc("POST /v1/auth/sso/start", s.ssoStart)                  // public
	s.mux.HandleFunc("POST /v1/auth/sso/acs", s.ssoACS)                      // public: the assertion is the credential
}

// domainTXTPrefix is where a tenant publishes its proof of an email domain:
// TXT _1trade-verify.<domain> = "1trade-verify=<token>".
const domainTXTPrefix = "_1trade-verify."

// SetTXTLookup replaces the DNS TXT resolver (tests).
func (s *Server) SetTXTLookup(f func(ctx context.Context, name string) ([]string, error)) { s.lookupTXT = f }

// ssoURLAllowed reports whether an IdP SSO URL may be sent to a browser: https only (plain http is
// accepted in dev, for a local test IdP). Anything else — javascript:, data:, relative — is refused,
// because the login page navigates to it.
func (s *Server) ssoURLAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Opaque != "" {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && s.cfg.IsDev())
}

// domainRe is a lowercase DNS name with at least one dot.
var domainRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)

// ssoURLs are this tenant's service-provider endpoints, on the web app (the BFF relays to us).
func (s *Server) ssoURLs(tenantID string) (entity, acs url.URL) {
	base := strings.TrimRight(s.cfg.AppBaseURL, "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	e, _ := url.Parse(base + "/api/auth/sso/metadata/" + tenantID)
	a, _ := url.Parse(base + "/api/auth/sso/acs")
	return *e, *a
}

// serviceProvider builds the SAML service provider for a tenant's configuration.
func (s *Server) serviceProvider(c store.SSOConfig) (*saml.ServiceProvider, error) {
	md, err := samlsp.ParseMetadata([]byte(c.IDPMetadata))
	if err != nil {
		return nil, err
	}
	entity, acs := s.ssoURLs(c.TenantID)
	return &saml.ServiceProvider{
		EntityID: entity.String(), MetadataURL: entity, AcsURL: acs, IDPMetadata: md,
		AuthnNameIDFormat: saml.EmailAddressNameIDFormat,
	}, nil
}

// ssoJSON renders a configuration for the admin screen (the metadata itself is not echoed).
func (s *Server) ssoJSON(c store.SSOConfig) map[string]any {
	entity, acs := s.ssoURLs(c.TenantID)
	return map[string]any{
		"configured": true, "idp_entity_id": c.IDPEntityID, "email_domains": c.Domains, "verified_domains": c.Verified,
		"pending_domains": pendingJSON(c.Pending), "default_role": c.DefaultRole,
		"jit": c.JIT, "enforce": c.Enforce, "updated_at": c.UpdatedAt.UTC().Format(time.RFC3339),
		"sp_entity_id": entity.String(), "acs_url": acs.String(), "sp_metadata_url": entity.String(),
	}
}

// pendingJSON renders the TXT records that would prove each pending domain.
func pendingJSON(p []store.SSODomain) []map[string]string {
	out := make([]map[string]string, 0, len(p))
	for _, d := range p {
		out = append(out, map[string]string{"domain": d.Domain, "txt_name": domainTXTPrefix + d.Domain, "txt_value": "1trade-verify=" + d.Token})
	}
	return out
}

// verifySSODomain serves POST /v1/account/sso/domains/{domain}/verify (admin): looks up the TXT
// record and, when it carries this tenant's token, marks the domain verified so it routes sign-ins.
func (s *Server) verifySSODomain(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	d := strings.ToLower(r.PathValue("domain"))
	tok, verified, err := s.st.DomainToken(r.Context(), p.TenantID, d)
	if errors.Is(err, store.ErrNoSSO) {
		writeErr(w, http.StatusNotFound, "not_found", "that domain is not in your SSO configuration")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if !verified {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		recs, _ := s.lookupTXT(ctx, domainTXTPrefix+d)
		if !contains(recs, "1trade-verify="+tok) && !s.cfg.SkipDomainDNS() {
			writeErr(w, http.StatusUnprocessableEntity, "not_verified", "the TXT record "+domainTXTPrefix+d+" does not contain your verification value yet")
			return
		}
		if err := s.st.MarkDomainVerified(r.Context(), p.TenantID, d); errors.Is(err, store.ErrDomainTaken) {
			writeErr(w, http.StatusConflict, "domain_taken", "another tenant has already verified that domain")
			return
		} else if err != nil {
			serverError(w, err)
			return
		}
		_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{TenantID: p.TenantID, ActorID: p.UserID, Action: "sso.domain.verify",
			TargetType: "tenant", TargetID: p.TenantID, After: map[string]any{"domain": d}, IsPaper: p.IsPaper})
	}
	c, err := s.st.GetSSO(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.ssoJSON(c))
}

// getSSO serves GET /v1/account/sso (admin).
func (s *Server) getSSO(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	c, err := s.st.GetSSO(r.Context(), p.TenantID)
	if errors.Is(err, store.ErrNoSSO) {
		entity, acs := s.ssoURLs(p.TenantID)
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "sp_entity_id": entity.String(), "acs_url": acs.String(), "sp_metadata_url": entity.String()})
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.ssoJSON(c))
}

// putSSO serves PUT /v1/account/sso (admin): the IdP metadata XML (it must carry a signing
// certificate and an SSO URL), the email domains that route to this tenant, and the policy.
func (s *Server) putSSO(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	var b struct {
		IDPMetadataXML string   `json:"idp_metadata_xml"`
		EmailDomains   []string `json:"email_domains"`
		DefaultRole    string   `json:"default_role"`
		JIT            *bool    `json:"jit"`
		Enforce        bool     `json:"enforce"`
	}
	dec := jsonDecoder(w, r, 256<<10) // metadata documents are a few KB
	if err := dec.Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	md, err := samlsp.ParseMetadata([]byte(b.IDPMetadataXML))
	if err != nil || len(md.IDPSSODescriptors) == 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_metadata", "idp_metadata_xml must be the identity provider's SAML metadata")
		return
	}
	if !hasSigningCert(md) || ssoLocation(md) == "" {
		writeErr(w, http.StatusUnprocessableEntity, "bad_metadata", "the metadata needs a signing certificate and an HTTP-Redirect SSO URL")
		return
	}
	if !s.ssoURLAllowed(ssoLocation(md)) {
		writeErr(w, http.StatusUnprocessableEntity, "bad_metadata", "the IdP's SSO URL must be an https:// URL")
		return
	}
	if b.DefaultRole == "" {
		b.DefaultRole = string(domain.RoleViewer)
	}
	if !domain.ValidRole(domain.Role(b.DefaultRole)) || b.DefaultRole == string(domain.RoleAdmin) {
		writeErr(w, http.StatusUnprocessableEntity, "bad_role", "default_role must be a non-admin role")
		return
	}
	if len(b.EmailDomains) == 0 || len(b.EmailDomains) > 10 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_domains", "give 1 to 10 email domains")
		return
	}
	domains := make([]string, 0, len(b.EmailDomains))
	for _, d := range b.EmailDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if !domainRe.MatchString(d) {
			writeErr(w, http.StatusUnprocessableEntity, "bad_domains", "not an email domain: "+d)
			return
		}
		domains = append(domains, d)
	}
	jit := true
	if b.JIT != nil {
		jit = *b.JIT
	}
	c := store.SSOConfig{TenantID: p.TenantID, IDPMetadata: b.IDPMetadataXML, IDPEntityID: md.EntityID,
		DefaultRole: b.DefaultRole, JIT: jit, Enforce: b.Enforce, Domains: domains}
	if err := s.st.PutSSO(r.Context(), c); errors.Is(err, store.ErrDomainTaken) {
		writeErr(w, http.StatusConflict, "domain_taken", "one of those domains already signs in to another tenant")
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{TenantID: p.TenantID, ActorID: p.UserID, Action: "sso.config.update",
		TargetType: "tenant", TargetID: p.TenantID, After: map[string]any{"idp_entity_id": md.EntityID, "email_domains": domains,
			"default_role": b.DefaultRole, "jit": jit, "enforce": b.Enforce}, IsPaper: p.IsPaper})
	c, err = s.st.GetSSO(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.ssoJSON(c))
}

// deleteSSO serves DELETE /v1/account/sso (admin): back to password sign-in.
func (s *Server) deleteSSO(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	if err := s.st.DeleteSSO(r.Context(), p.TenantID); err != nil {
		serverError(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{TenantID: p.TenantID, ActorID: p.UserID, Action: "sso.config.delete",
		TargetType: "tenant", TargetID: p.TenantID, IsPaper: p.IsPaper})
	w.WriteHeader(http.StatusNoContent)
}

// hasSigningCert reports whether IdP metadata carries a certificate usable for signatures.
func hasSigningCert(md *saml.EntityDescriptor) bool {
	for _, d := range md.IDPSSODescriptors {
		for _, k := range d.KeyDescriptors {
			if (k.Use == "" || k.Use == "signing") && len(k.KeyInfo.X509Data.X509Certificates) > 0 {
				return true
			}
		}
	}
	return false
}

// ssoLocation is the IdP's HTTP-Redirect SSO URL.
func ssoLocation(md *saml.EntityDescriptor) string {
	for _, d := range md.IDPSSODescriptors {
		for _, e := range d.SingleSignOnServices {
			if e.Binding == saml.HTTPRedirectBinding {
				return e.Location
			}
		}
	}
	return ""
}

// ssoMetadata serves GET /v1/auth/sso/metadata/{tenant_id}: this tenant's service-provider metadata,
// for the IdP administrator.
func (s *Server) ssoMetadata(w http.ResponseWriter, r *http.Request) {
	c, err := s.st.GetSSO(r.Context(), r.PathValue("tenant_id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "single sign-on is not configured")
		return
	}
	sp, err := s.serviceProvider(c)
	if err != nil {
		serverError(w, err)
		return
	}
	out, err := xml.MarshalIndent(sp.Metadata(), "", "  ")
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	_, _ = w.Write(out)
}

// ssoStart serves POST /v1/auth/sso/start {email}: the IdP URL to send the browser to, with an
// AuthnRequest whose id we record (and later accept exactly once).
func (s *Server) ssoStart(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Email string `json:"email"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	at := strings.LastIndex(b.Email, "@")
	if at < 1 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_email", "a work email is required")
		return
	}
	c, err := s.st.SSOForDomain(r.Context(), strings.ToLower(strings.TrimSpace(b.Email[at+1:])))
	if errors.Is(err, store.ErrNoSSO) {
		writeErr(w, http.StatusNotFound, "sso_not_configured", "single sign-on is not set up for that email domain")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	sp, err := s.serviceProvider(c)
	if err != nil {
		serverError(w, err)
		return
	}
	req, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(saml.HTTPRedirectBinding), saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		serverError(w, err)
		return
	}
	if err := s.st.SaveSSORequest(r.Context(), req.ID, c.TenantID); err != nil {
		serverError(w, err)
		return
	}
	u, err := req.Redirect(req.ID, sp) // RelayState carries the request id back to the ACS
	if err != nil {
		serverError(w, err)
		return
	}
	if !s.ssoURLAllowed(u.String()) { // re-checked here: the browser navigates to it
		writeErr(w, http.StatusUnprocessableEntity, "bad_metadata", "the identity provider's sign-in URL is not https")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"redirect_url": u.String()})
}

// ssoACS serves POST /v1/auth/sso/acs {saml_response, relay_state} (relayed by the web app from the
// IdP's form post): the signed assertion must answer a request we made (once), be for our ACS and
// audience, be current, and name an email in one of the tenant's domains. It then signs the member in,
// creating them if just-in-time creation is on. The IdP is responsible for its own MFA.
func (s *Server) ssoACS(w http.ResponseWriter, r *http.Request) {
	var b struct {
		SAMLResponse string `json:"saml_response"`
		RelayState   string `json:"relay_state"`
	}
	dec := jsonDecoder(w, r, 512<<10)
	if err := dec.Decode(&b); err != nil || b.SAMLResponse == "" || b.RelayState == "" || len(b.RelayState) > 128 {
		writeErr(w, http.StatusBadRequest, "bad_request", "saml_response and relay_state are required")
		return
	}
	fail := func(code, msg string, err error) {
		slog.Warn("sso sign-in refused", "code", code, "err", err)
		writeErr(w, http.StatusUnauthorized, code, msg)
	}
	tenantID, err := s.st.ClaimSSORequest(r.Context(), b.RelayState)
	if err != nil {
		fail("sso_request", "this sign-on attempt is unknown, used or expired — start again", err)
		return
	}
	c, err := s.st.GetSSO(r.Context(), tenantID)
	if err != nil {
		fail("sso_not_configured", "single sign-on is not configured", err)
		return
	}
	sp, err := s.serviceProvider(c)
	if err != nil {
		serverError(w, err)
		return
	}
	raw, err := base64.StdEncoding.DecodeString(b.SAMLResponse)
	if err != nil {
		fail("sso_invalid", "the identity provider's response is not valid", err)
		return
	}
	assertion, err := sp.ParseXMLResponse(raw, []string{b.RelayState}, sp.AcsURL)
	if err != nil {
		var ire *saml.InvalidResponseError
		if errors.As(err, &ire) {
			err = ire.PrivateErr
		}
		fail("sso_invalid", "the identity provider's response is not valid", err)
		return
	}
	email := assertedEmail(assertion)
	at := strings.LastIndex(email, "@")
	if at < 1 || !contains(c.Verified, strings.ToLower(email[at+1:])) {
		fail("sso_domain", "that email is not in a domain this tenant signs in with", errors.New(email))
		return
	}
	u, created, err := s.st.SSOUser(r.Context(), c, email)
	switch {
	case errors.Is(err, store.ErrSSOOtherTenant):
		writeErr(w, http.StatusForbidden, "sso_account_conflict", "that email already has an account in another 1Trade tenant")
		return
	case errors.Is(err, store.ErrSSONoAccount):
		writeErr(w, http.StatusForbidden, "sso_no_account", "ask your admin to invite you first")
		return
	case err != nil:
		serverError(w, err)
		return
	}
	action := "sso.login"
	if created {
		action = "sso.user.create"
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{TenantID: u.TenantID, ActorID: u.UserID, Action: action,
		TargetType: "user", TargetID: u.UserID, After: map[string]any{"email": email, "idp": c.IDPEntityID}, IsPaper: u.IsPaper})
	s.issue(w, http.StatusOK, u.UserID, domain.Claims{
		TenantID: u.TenantID, OrgID: u.OrgID, SubAccountID: u.SubAccountID, Roles: u.Roles, IsPaper: u.IsPaper,
	})
}

// assertedEmail is the email an assertion names: an email-format NameID, else a standard email attribute.
func assertedEmail(a *saml.Assertion) string {
	if a.Subject != nil && a.Subject.NameID != nil && strings.Contains(a.Subject.NameID.Value, "@") {
		return strings.TrimSpace(a.Subject.NameID.Value)
	}
	for _, st := range a.AttributeStatements {
		for _, at := range st.Attributes {
			switch at.Name {
			case "email", "mail", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress", "urn:oid:0.9.2342.19200300.100.1.3":
				if len(at.Values) > 0 {
					return strings.TrimSpace(at.Values[0].Value)
				}
			}
		}
	}
	return ""
}

// contains reports whether list holds v.
func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
