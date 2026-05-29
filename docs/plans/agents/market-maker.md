# Agent plan — `market-maker`

> **Paused for Phase 1.** Designed-only. Activates at Phase 2 switch-on.

## 1. Scope under the pivot

The original agent definition stays. No service ships in Phase 1 — only the spec. With the
exchange paused, there is no continuous quoting to do.

## 2. Features owned

| Feature | Status |
|---|---|
| [KW04 — Market-maker spec](../features/KW04-market-maker-spec.md) | spec-only, M6 finalized |
| Real market-maker implementation | Phase 2 (post-license) |

## 3. Milestone-by-milestone (Phase 1)

### Milestones 1–5 — Periodic spec maintenance
- `services/market-maker/SPEC.md` documents:
  - Pricing logic (base 1% spread, skew on inventory imbalance).
  - Risk controls (max inventory per product, daily P&L stop-loss, vol-triggered withdrawal).
  - Fair-value anchoring against the private reference index.
  - Audit log shape for every quote update.
- Reviewed alongside `matching-engine` SPEC.md — they need to agree on order submission interface.

### Milestone 6 — Spec finalized
- SPEC.md complete; reviewed by `tech-lead` and `security-compliance`.
- Test plan written (no tests run yet — no engine to test against).

### Phase 2 — Implement
- Go service: continuous bid/ask quotes for every tradeable product.
- Submits orders to `matching-engine` like any participant — no privileged path.
- Risk limits enforced; quotes auto-withdraw on volatility / anomalous book.
- Manual override hook.
- Quote audit log lands in the same trail as other order activity.

## 4. Contracts owned / consumed

### Owned
- None (market-maker submits to others' APIs).

### Consumed
- `openapi/trading.yaml` (submits orders).
- `openapi/index.yaml` (fair-value anchor from the private/public index).
- `events/trades.executed.v1.yaml` (sees the result of its own quotes).

## 5. Local dev

- Not running in Phase 1.
- The mock adapter in `matching-engine` simulates two-sided depth; no separate MM service is needed
  for the demo.

## 6. Dockerfile + deploy

- Specced at `deploy/docker/Dockerfile.go-service` template; activates at Phase 2.
- K8s Deployment in Phase 2: 1 replica with hot standby; risk-control config from ConfigMap +
  manual-override CRD.

## 7. Conventions

- **Just another order source** to `matching-engine` — no special routing.
- **Respects `is_paper`** — separate quoting paths for paper vs. real-money venues.
- **Internal accounts cannot trade against customer paper accounts** — even during paper trading
  (insider risk). Verified by `security-compliance`.
- **Fixed-point math** in pricing.

## 8. Hard boundaries

- Don't modify the matching engine or ledger.
- Don't set the index.
- Don't ship in Phase 1.

## 9. Definition of done

### Phase 1
- SPEC.md complete and reviewed.

### Phase 2
- Maintains continuous two-sided quotes within target spread.
- Risk limits enforced and tested.
- Withdraws cleanly under volatility.
- All quotes audited.
