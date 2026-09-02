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
