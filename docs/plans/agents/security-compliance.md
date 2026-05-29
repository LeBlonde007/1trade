# Agent plan — `security-compliance`

> Active. Reviewer + owner of the **licensing track** (the cheap parallel workstream that makes
> Phase 2 unblockable).

## 1. Scope under the GTM pivot

The reviewer charter is unchanged: audit-trail completeness, paper/real isolation, KYC/AML
enforcement, regulatory framing, SOC 2 controls, secrets, PII. **Added under the pivot:**
explicit ownership of the licensing track — keep the prepaid-credit framing clean (so no license
needed for Phase 1), and drive the path to license clearance so Phase 2 is unblockable.

## 2. Features owned

| Feature | Status |
|---|---|
| [F21 — SOC 2 Type I](../features/F21-soc2-type1.md) | active (co-owner with `infra-sre`), M4–M6 |
| [F22 — Licensing track](../features/F22-licensing-track.md) | active (sole owner), continuous, parallel |
| Cross-cutting: PR review on credits/orders/auth/PII | active, continuous |

## 3. Milestone-by-milestone

### Milestone 1 — Baseline
- Engage Vanta (or Drata) for SOC 2 control automation.
- Engage securities/commodities counsel.
- Confirm prepaid-credit framing keeps Phase 1 license-free:
  - Credits are "prepaid service units."
  - No secondary transfer between customers in Phase 1.
  - No "futures," "speculation," or "investment return" language in customer-facing surfaces.
- KYC: light KYC for paper trading (existing flow stays warm); for Phase 1 platform customers,
  follow `platform-core`'s enterprise onboarding (corporate documents, beneficial ownership for
  $X+ purchases).
- Review every PR that touches: credits, orders (keep-warm), auth, billing, customer data.

### Milestone 2 — Audit hooks
- Verify hash chain in `credit-ledger` is functioning end-to-end.
- Verify Stripe webhook idempotency keys are enforced.
- Verify `is_paper` carried in events + DB rows + API requests.
- Document SOC 2 control mapping to services.

### Milestone 3 — Real-money platform path
- Real-money inference/compute starts this milestone. Verify:
  - Full KYC for any account that buys real-money credits.
  - AML screening (Persona / Parallel Markets — integration scoped this milestone).
  - Audit log queryable for every admin action.
- Conservative framing review on all customer-facing copy.

### Milestone 4 — SOC 2 kickoff + enterprise
- SOC 2 Type I audit kickoff with audit firm.
- SAML SSO + SCIM rolled out (Phase 1 enterprise) — verify access controls.
- Sub-account RBAC review: cross-tenant data leak attempts blocked.
- Insider risk: internal accounts cannot trade against customer paper-trading accounts (keep-warm
  surface, but verified now).

### Milestone 5 — Licensing decision support
- Map exact license/registration requirement and timeline per jurisdiction (US per-state, Japan via UBS,
  Singapore MAS, offshore).
- Confirm the credits-as-prepaid framing review with counsel — written sign-off.
- Brief founder on jurisdiction options + recommended path.

### Milestone 6 — SOC 2 Type I report + license decision
- SOC 2 Type I report achieved.
- Founder decision on Phase 2 jurisdiction (or explicit deferral).
- Trade surveillance review (the design from `surveillance` agent kept warm): cleared for Phase 2 use.

## 4. Contracts owned / consumed

### Owned
- No service contracts (security-compliance doesn't expose APIs).
- The regulatory framing copy guide (`docs/regulatory-framing.md`).
- The KYC/AML policy doc (`docs/kyc-aml-policy.md`).
- The SOC 2 control matrix (`docs/soc2-controls.md`).

### Consumed
- Every other contract — reads everything; flags drift, gaps, language issues.
- `credit-types.md` — to verify nothing accidentally turns into a tradeable security label.

## 5. Local dev

- No service to run.
- `make audit-review` runs:
  - `gitleaks` (secrets scan).
  - `trivy` (image scan against the latest built images).
  - A linter over `apps/web/` for forbidden customer-facing words ("futures", "speculation",
    "investment return", "guaranteed return", "yield").
  - `events/` + DB schemas: presence of `is_paper`.
  - `credit-ledger` chain verify.

## 6. Tools restriction

Read / Grep / Glob / Bash only (per agent definition). No Write/Edit — `security-compliance`
reviews and recommends; the owning agent makes the fix.

## 7. Output format

Reviews are produced as prioritized findings:

```
[BLOCKER] services/credit-ledger/internal/api/purchase.go:142
  Purchase endpoint does not check Stripe webhook signature.
  Required by: SOC 2 CC6.1 + regulatory.
  Recommended fix: verify Stripe-Signature header per platform-core's auth helper.

[SHOULD-FIX] apps/web/pages/trade/index.vue:18
  Marketing copy uses "futures contracts" in the trading paused ribbon.
  Recommended fix: "secondary market for unused prepaid services — currently paused."

[NOTE] services/inference-gateway/internal/api/chat.go:89
  Usage event payload includes user prompt (PII). Consider truncating to length only.
```

Hand findings back to the owning agent through the orchestrator.

## 8. Definition of done

- Every PR touching credits, orders, auth, or PII reviewed before merge.
- SOC 2 Type I report achieved by M6.
- Licensing track has a documented path + jurisdiction recommendation by M5.
- The four immutable commitments verified intact at every gate.
- Regulatory framing clean on customer-facing surfaces (no banned words, conservative language).
- KYC/AML enforced: no path to real-money without full KYC; AML screening live.
