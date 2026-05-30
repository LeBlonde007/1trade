# platform-core (F02 auth · F03 accounts/RBAC · F06 billing · gateway · CLI)

Connective platform tissue: identity for the whole fleet. Owner: `platform-core`. Contract:
`docs/contracts/openapi/platform-core.yaml` (auth, accounts, RBAC, API keys, **JwtClaims**).

## Status (F02 core — complete, deployed to k3d)
- ✅ **Domain core** (`internal/domain/`, pure + unit-tested): bcrypt password hashing; HS256 JWT
  issuance/verification whose claims (`tenant_id`, `org_id`, `sub_account_id`, `roles`, `is_paper`,
  `sub`, `iat`, `exp`) match the contract **and** what consumers like credit-ledger verify; API-key
  generation (shown once) + sha256 hash + constant-time verify; the RBAC `Role` enum.
- ✅ **Store** (`internal/store/`, integration-tested vs Postgres): tenants/orgs/users/api_keys +
  migration `0001_init.sql`; atomic signup (tenant+admin user), login lookup, API-key lifecycle.
- ✅ **API** (`internal/api/`, httptest vs Postgres): `signup`(auto-login)/`login`→JWT/`me`/`logout`/
  keys CRUD; OAuth scaffold (501 until configured); generic errors (no account-existence / internal
  leak); secret shown once; tenant-scoped revoke.
- ✅ **F02↔F05 integration** (verified live in k3d): one signup → JWT → credit-ledger verifies it
  and resolves `tenant_id`/`is_paper` from the token. Shared `PLATFORM_JWT_SECRET` via the
  `platform-auth` Secret. Absent/tampered tokens → 401.
- ✅ **Deploy**: distroless Dockerfile, k8s base (migrate initContainer, probes, non-root), Tilt.
- ⬜ M4 follow-ons: OAuth providers, SAML/SCIM, 2FA, full accounts/org surface (F03).

## Invariants
- The JWT here is the fleet's auth contract — keep its claim keys exactly as `platform-core.yaml`
  `JwtClaims` (a drift breaks every service that authorizes by `tenant_id`/`is_paper`).
- Secrets (JWT signing key, OAuth client secrets) come from env/Vault — never code.
- Admin actions are audited (F03). Passwords: bcrypt. API keys: high-entropy random + sha256.

## Dev
```bash
cd services/platform-core && go test ./...   # domain unit tests (no DB)
```
