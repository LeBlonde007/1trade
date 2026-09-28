# F03 — Accounts, orgs, sub-accounts, RBAC

> Ship in **Milestone 1** (base) → **Milestone 4** (sub-accounts + audit). Owner: `platform-core`.

## Spec

- **Tenant** is the billing unit. A tenant can be a single user (AI-startup engineer) or an org
  (F500 / frontier lab).
- **Org** wraps a tenant for multi-user accounts; users belong to one or more orgs.
- **Sub-accounts** (M4): an org can carve out sub-accounts per team with per-team credit budgets.
- **RBAC roles**: admin, billing, engineer, viewer. (`trader` role pre-defined for Phase 2; unused
  in Phase 1.)
- **Audit log**: every admin action logged with `(tenant_id, actor_id, action, before, after, ts)`.

## Owning agent

`platform-core`.

## Contracts consumed / produced

### Produces
- `openapi/platform-core.yaml` — tenants, orgs, users, roles, sub-accounts.
- `schemas/types.sql` co-owned with `tech-lead` (tenant_id shape, role enum).
- `events/admin.action.v1.yaml` — admin audit events.

### Consumes
- Auth (F02).
- `openapi/credit.yaml` — sub-account budget reads.

## Dependencies

- F01, F02.

## Sync points

- M1 — tenant + org + role shape published; downstream services adopt for authz.
- M4 — sub-account hierarchy published; `inference-ml`, `compute-platform`, `credit-ledger`
  adapt to sub-account-scoped quotas + budgets + consumption views.
- M4 — audit-log queryable surface published; `security-compliance` validates SOC 2 control
  coverage.

## Acceptance criteria

- [ ] Create tenant, org, user, invite flow.
- [ ] Assign roles per user per org.
- [ ] (M4) Create sub-account; assign budget; restrict members.
- [ ] (M4) Org-wide consumption dashboard data API.
- [ ] (M4) Audit log queryable via API + CLI (`1trade audit log`).
- [ ] All sensitive actions logged.
- [ ] Sub-account RBAC blocks cross-sub-account data access (penetration-tested by `security-compliance`).

## Milestone

- M1 (Gate 1): base tenants/orgs/RBAC.
- M4 (Gate 4): sub-accounts + audit + org dashboards.

---

## Status — team management (2026-09-28)

**Built** (`platform-core.yaml` v1.6, `credit.yaml` v1.4, platform-core migration `0006_team.sql`,
ledger migration `0006_transfer.sql`):
- **Members:** the whole team can list members. Admins remove them, but never themselves (409 `self`)
  and never the last admin (409 `last_admin`). The last-admin check runs under a lock over the
  tenant's users, so a stale admin token cannot remove the last real admin.
- **Invitations:**
  - admins invite an email with roles (default viewer) and optionally a sub-account;
  - a random token goes out by email, and only its sha256 is stored; it is **single use** and lasts 7
    days, with one pending invite per email and one account per email;
  - accepting it (`/invite`) creates a verified user in the inviting tenant and signs them in;
  - admins can revoke.
- **Sub-accounts:** teams with their **own ledger balances**. A member placed in one carries
  `sub_account_id` in their JWT, so inference and compute usage is billed to it. Admin or billing
  funds or drains it from the main balance through the ledger's new `transfer`:
  - two hash-chained legs in one DB transaction, so credits are never created or lost;
  - idempotent;
  - service token only, with no dev shortcut;
  - the idempotency key is scoped to the sub-account and direction.
- **Audit:** every change is audited: `invite.create/accept/revoke`, `user.remove`,
  `user.sub_account.set`, `sub_account.create/fund/return`.
- **Console:** `/enterprise/teams` is live (members with their sub-account and remove, invite and
  revoke, sub-accounts with create / fund / return). `/invite` accepts an invitation.

**Verified:**
- **Ledger:** fund then return with exact balances; a replay moves nothing; an overdraft moves
  nothing; same-balance, zero, bad-id, missing-key and non-service calls refused; hash chain intact.
- **Platform:**
  - invite, lookup, accept (roles and sub-account in the JWT); a second use is 404;
  - a weak password is 422, a duplicate invite 409, inviting an existing account 409;
  - revoke; remove, after which the member cannot sign in; self-removal refused;
  - the last admin is protected even from a stale token;
  - engineers cannot manage the team;
  - no cross-tenant sub-account assignment, invitation, removal or funding;
  - transfers with scoped keys, validation, a short balance, and the role gate.
- **Mutation-checked:** 14 of 14 caught.
- **Live stack** (real ledger): funding moves main 25 → Research 25. Bob accepts the emailed link,
  signs in as an engineer in Research, and reusing the link gets 404. Screenshots:
  `docs/screenshots/team-members.png`, `team-invite.png`.

**Acceptance:** ✅ invite flow · ✅ roles per user · ✅ sub-accounts with budgets (funded balances) and
restricted members · ✅ sensitive actions logged · ⬜ org-wide consumption dashboard API · ⬜ CLI
`1trade audit log` · ⬜ penetration test by `security-compliance`.

Known limit: a removed member's existing token stays valid until it expires (JWTs are verified by
each service, and nothing revokes them), although sign-in fails at once.
