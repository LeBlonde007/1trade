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
