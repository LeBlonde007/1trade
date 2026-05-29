# F22 — Licensing track (parallel, Phase 2 unblock)

> **Runs parallel to all other features.** Owner: `security-compliance` (with the founder).

## Spec

Pausing the *build* of the exchange does not pause the *path* to license clearance. This is the
cheap, high-option-value parallel workstream that keeps Phase 2 unblockable.

Three legs:

1. **Decide jurisdiction.** US (state-by-state, possible MSB registration), Japan (via UBS COO),
   Singapore (MAS-friendly), offshore (Cayman / BVI). Trade-offs documented.
2. **Engage securities/commodities counsel.** Confirm:
   - **Phase 1 prepaid framing stays clear of licensing** (credits are "prepaid service units,"
     redeemable, no secondary transfer between customers).
   - **What exactly triggers licensing** (resale between customers? Index publication? Forward
     contracts?).
3. **Map the license/registration required** for the chosen jurisdiction + timeline + cost.

## Owning agent

`security-compliance`. The founder owns the business decision; security-compliance drives the
process.

## Contracts consumed / produced

- Produces: `docs/regulatory-framing.md` (the customer-facing language guide),
  `docs/licensing-decision.md` (the jurisdiction memo + recommendation).
- Consumes: all customer-facing copy (audited against the framing guide).

## Dependencies

- None within this plan. External: counsel engagement, jurisdiction research.

## Sync points

- M1 — counsel engaged; framing review begins.
- M3 — written sign-off that Phase 1 prepaid framing is license-free.
- M5 — jurisdiction memo + recommendation drafted.
- M6 — founder decision on Phase 2 jurisdiction (proceed or document deferral).

## Acceptance criteria

- [ ] Written counsel sign-off on Phase 1 framing.
- [ ] Jurisdiction recommendation memo with cost + timeline per option.
- [ ] All customer-facing surfaces audited against the framing guide (no banned words: "futures,"
      "speculation," "investment return," "guaranteed yield").
- [ ] Phase 2 switch-on prerequisites documented and tracked.

## Milestone

- M6 (Gate 6): jurisdiction decision.

## Constraint

This track is **about the path, not the build**. Spending here is counsel fees + research time,
not engineering work. If engineering effort starts trickling toward Phase 2 implementation in
violation of the pivot, `tech-lead` and `security-compliance` flag it together.
