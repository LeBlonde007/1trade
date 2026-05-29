# platform-core (F02 auth · F03 accounts/RBAC · F06 billing · gateway · CLI)

Connective platform tissue: identity for the whole fleet. Owner: `platform-core`. Contract:
`docs/contracts/openapi/platform-core.yaml` (auth, accounts, RBAC, API keys, **JwtClaims**).

## Status (F02, in progress)
- ✅ **Domain core** (`internal/domain/`, pure + unit-tested): bcrypt password hashing; HS256 JWT
  issuance/verification whose claims (`tenant_id`, `org_id`, `sub_account_id`, `roles`, `is_paper`,
  `sub`, `iat`, `exp`) match the contract **and** what consumers like credit-ledger verify; API-key
  generation (shown once) + sha256 hash + constant-time verify; the RBAC `Role` enum.
- ⬜ Store (tenants/orgs/users/api_keys + migrations), API (signup/login→JWT/me/logout/keys, OAuth
  scaffold), F02↔F05 integration (shared `PLATFORM_JWT_SECRET`), deploy.

## Invariants
- The JWT here is the fleet's auth contract — keep its claim keys exactly as `platform-core.yaml`
  `JwtClaims` (a drift breaks every service that authorizes by `tenant_id`/`is_paper`).
- Secrets (JWT signing key, OAuth client secrets) come from env/Vault — never code.
- Admin actions are audited (F03). Passwords: bcrypt. API keys: high-entropy random + sha256.

## Dev
```bash
cd services/platform-core && go test ./...   # domain unit tests (no DB)
```
