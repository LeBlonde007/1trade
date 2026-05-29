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
