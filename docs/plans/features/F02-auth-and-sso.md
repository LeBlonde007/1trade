# F02 — Auth & SSO

> Ship in **Milestone 1** (base) → **Milestone 4** (enterprise). Owner: `platform-core`.

## Spec

Three identity surfaces:

1. **Standard signup** (M1): email + password or OAuth (Google, GitHub, Microsoft).
2. **API keys** (M1): scoped, hashed at rest, rotatable.
3. **Enterprise SSO** (M4): SAML 2.0, SCIM provisioning, IP allowlisting, 2FA (TOTP).

JWTs carry `tenant_id`, `org_id`, `sub`, `roles`, `is_paper`, `exp`. Every service authenticates
inbound requests through the same auth helper.

## Owning agent

`platform-core`.

## Contracts consumed / produced

### Produces
- `openapi/platform-core.yaml` — auth endpoints (`/auth/login`, `/auth/oauth/*`, `/auth/saml/*`,
  `/auth/keys/*`).
- JWT shape (documented inside `platform-core.yaml`).
- SCIM endpoints.

### Consumes
- Vault for OAuth client secrets, SAML signing keys.

## Dependencies

- F01 (infra) — needed for K8s, Postgres, Vault.

## Sync points

- M1 day 5 — JWT shape published; other services adopt the auth helper.
- M1 end — email/OAuth login works end-to-end (with `make test-e2e` covering OAuth flow).
- M4 start — SAML/SCIM/2FA shape published; downstream agents (`trading-frontend` enterprise
  login page) adapt.

## Acceptance criteria

- [ ] Email + password signup with verification.
- [ ] OAuth login with Google, GitHub, Microsoft.
- [ ] API key create/list/revoke.
- [ ] (M4) SAML SSO with Okta, Entra ID, Google Workspace.
- [ ] (M4) SCIM 2.0 provisioning.
- [ ] (M4) 2FA TOTP.
- [ ] (M4) IP allowlisting.
- [ ] All auth events logged (signup, login, logout, key issued/revoked, SSO config changed).
- [ ] `security-compliance` review: no path to real-money without full KYC; KYC trigger lives
      here in M3+.

## Milestone

- M1 (Gate 1): standard signup + OAuth + API keys.
- M4 (Gate 4): enterprise SSO + SCIM + 2FA.

---

## Status — two-factor authentication (2026-09-28)

**Built** (`platform-core.yaml` v1.7, migration `0007_mfa.sql`):
- **TOTP (RFC 6238):** SHA-1, 6 digits, 30 s steps, one step of drift allowed either side. Secrets
  are sealed with AES-256-GCM (`MFA_ENC_KEY`, or a key derived from the JWT secret with domain
  separation).
- **Enrolment:** `setup` issues a pending secret. `enable` confirms a code from it and returns ten
  single-use recovery codes, shown once and stored as sha256.
- **Login:** with 2FA on, the password earns only a **5-minute challenge**. It has no `tenant_id`, so
  no service accepts it as a session. The BFF keeps it in an httpOnly, SameSite=strict cookie scoped
  to `/api/auth/login`, so page scripts never see it. `/login/2fa` redeems it with a code or a
  recovery code.
- **Hardening:**
  - every code works once: its time step is claimed atomically in SQL, so a race of 8 concurrent
    submissions yields exactly 1 session;
  - recovery codes are single use;
  - 5 wrong codes lock the second step for 15 minutes;
  - turning 2FA off needs a current code or a recovery code, not just a session.

  Enable and disable are audited.
- **Console:** the Settings → Account & Security card is live (set up with the key and an otpauth
  link, confirm, recovery codes shown once, turn off). The login 2FA step is wired, with a
  recovery-code option. The placeholder card and the fake "remember this device" and countdown are
  gone.

**Verified:**
- **Tests:**
  - the RFC 6238 test vectors;
  - drift window, replay, and short codes;
  - sealed-secret tampering;
  - a challenge is not a session and a session is not a challenge;
  - end to end: enable, login challenge, the enrolment code reused (refused), the next code once,
    recovery once, disable needing a factor;
  - lockout;
  - the concurrent race.
- **Mutation-checked:** 11 of 11 caught.
- **Live:** enrolled with a real TOTP; after the password only the challenge cookie exists; a wrong
  recovery code is refused; a real one signs in. Screenshots: `docs/screenshots/settings-2fa.png`,
  `login-2fa.png`.

**Acceptance:** ✅ 2FA TOTP · ⬜ tenant-wide "require 2FA" policy · ⬜ WebAuthn.

## Status — SAML single sign-on (2026-09-28)

**Built** (`platform-core.yaml` v1.8, migration `0008_sso.sql`, crewjam/saml for XML-DSig):
- **Per-tenant IdP.** Admins save the IdP metadata (it must carry a signing certificate and an
  HTTP-Redirect SSO URL), 1–10 email domains (each routes to exactly one tenant), a non-admin role
  for just-in-time members, and optional **enforcement**. Under enforcement, password sign-in is
  refused for non-admins; admins keep it as the break-glass path. Changes are audited.
- **Flow.** The login "Sign in with SSO" option takes the work email and redirects to the IdP. The
  AuthnRequest id is recorded. The IdP form-posts to the web app's ACS, which relays it to
  platform-core. The assertion must:
  - be signed by the tenant's IdP;
  - answer a request we made (the id is consumed once, within 10 minutes);
  - be for our ACS and audience;
  - be currently valid;
  - name an email in the tenant's domains.

  The member is then found, or created (JIT) with the default role, a verified email and no usable
  password, and signed in. The BFF sets the session and redirects to `/console`; failures land on
  `/login?sso_error=…` with a readable reason.
- **Never merges accounts.** An email that belongs to another tenant is refused. There are no
  outbound calls: the artifact binding and metadata-by-URL are not accepted (no SSRF surface).
  Encrypted assertions are not supported, because there is no SP key.
- **Console:** `/enterprise/sso` is live. It shows the SP entity ID, ACS and metadata URL for the IdP
  admin, and takes the metadata, domains and policy.

**Verified:**
- **Real crewjam IdP in tests:**
  - JIT sign-in with role and tenant;
  - replay, a response to another request, a tampered assertion, an untrusted IdP key, a foreign
    domain and an expired request are all refused;
  - the second sign-in reuses the member; an SSO member has no password;
  - audit entries; an unknown domain is 404.
- **Policy:** another tenant's account refused, JIT off refused, enforcement (a viewer is refused,
  removal restores password sign-in), domain exclusivity, metadata / role / domain validation, and
  admins only.
- **Mutation-checked:** 9 of 9 caught.
- **Live browser against a running SAML IdP:** login → IdP → signed post → console as a JIT
  engineer. Screenshot: `docs/screenshots/sso-admin.png`.

**Acceptance:** ✅ SAML SSO (any SAML 2.0 IdP — Okta, Entra ID, Google Workspace — via metadata) ·
✅ 2FA TOTP · ⬜ SCIM 2.0 · ⬜ IP allowlisting · ⬜ domain ownership verification (DNS TXT) before a
domain can route · ⬜ signed AuthnRequests and encrypted assertions (needs an SP key pair).
