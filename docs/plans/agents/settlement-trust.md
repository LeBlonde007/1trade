# Agent plan — `settlement-trust`

> Partial under the GTM pivot. **DC onboarding + attestation + payout are active** (they make the
> platform work). Tradeable-contract and trade-escrow logic are designed-but-dormant for Phase 2.

## 1. Scope under the pivot

The settlement-trust layer makes credits credible enough to back real consumption (Phase 1) and,
later, real trading (Phase 2). Under the pivot:

- **Active now:** mint/burn against attested reserves, escrow + streamed payout to DCs, backing
  ratio + proof-of-reserves publication, partner DC onboarding (manual v1), the attestation
  stack.
- **Deferred to Phase 2:** the trade-escrow flow (buyer cash → custodial escrow → DC on metered
  delivery) is **designed and stubbed** because today, customers pay 1Trade directly and
  1Trade pays partner DCs (it's a redeem-only flow). When the exchange goes live, the same
  primitives compose into the trade-escrow flow.

## 2. Features owned

| Feature | Status |
|---|---|
| [F17 — DC partner onboarding (manual v1 → productized v1.5)](../features/F17-dc-partner-onboarding.md) | active, M4 (first) → M5 (second) |
| [F18 — Partner payouts (escrow + streamed)](../features/F18-partner-payouts.md) | active, M4 (first cycle) → M5 (PoR public) |
| [F19 — GPU attestation stack](../features/F19-gpu-attestation.md) | active, M4 |
| [F16 — Supply-source abstraction](../features/F16-supply-source-abstraction.md) (co-owner with `compute-platform`) | active, M3–M4 |

## 3. Milestone-by-milestone

### Milestone 3 — Foundation: owned DC backing
- `services/supply-service/` scaffolded.
- Schema: `dc_partners`, `partner_capacity`, `partner_payouts`, `attestation_records`,
  `reserves_snapshots`.
- Own DC modeled as a `dc_partner` with `state=active` and trivial attestation (we own it).
- Mint/burn integration with `credit-ledger`: minting GPU credits against attested capacity,
  burning on consumption.
- Backing ratio invariant enforced at mint: outstanding GPU credits per class ≤ attested reserved
  capacity for that class. Refuse mint if it would over-issue.

### Milestone 4 — First partner DC
- Partner onboarding flow (manual): ops contact, location, capacity, SLA, pricing floor.
- **KYB + facility certs** uploaded and reviewed.
- **NVIDIA hardware attestation**: partner runs an attestation agent that reports NVIDIA-signed
  GPU identity claims.
- **Randomized nonce-seeded challenge-response**: a small workload run on the partner's GPUs
  to prove they actually perform at the claimed class.
- **Continuous DCGM telemetry** streamed back: utilization, throttling, ECC errors.
- **Staked bond** held in escrow; forfeitable on proven fraud.
- First partner activated; soft launch (small allocation, monitored).
- First **partner payout cycle**: monthly aggregation of `supply_source=<partner>` consumption,
  `gpu_hours_consumed * agreed_rate`, minus 1Trade fee, 10–20% holdback through dispute window,
  wired to partner.

### Milestone 5 — Public proof of reserves
- **`GET /v1/supply/proof-of-reserves`** — per credit class, the breakdown:
  attested capacity, outstanding credits, backing ratio, last attestation timestamp.
- Web view in the platform console (consumer-facing transparency).
- Second partner DC onboarded; cross-partner backing math working.
- Dispute window flow: hold → release on no dispute / escalate on dispute.

### Milestone 6 — Hardening
- Bond forfeiture playbook (legal + technical).
- Attestation refresh discipline (heartbeats, re-attest on agent restart, alert on drift).
- Documented partner SLA enforcement.

### Phase 2 — Trade-escrow switch-on (post-license)
- The buyer-cash → custodial-escrow → streamed-to-DC flow goes live. The primitives (escrow,
  streaming payout, holdback, bond) are the same ones built in Phase 1 — just wired to a trade
  event instead of a direct consumption debit.
- Dated, class-standardized credits (`H100-80GB-hour deliverable in <window>`) — the data model
  for this already exists from Phase 1, kept stub-active.

## 4. Contracts owned / consumed

### Owned
- `docs/contracts/openapi/supply.yaml` (partner-facing + internal).
- `docs/contracts/events/partner.capacity.v1.yaml` (co-owned with `compute-platform`; the
  capacity-registration event).
- `docs/contracts/events/payout.cycle.v1.yaml` (payout records).
- Attestation event schemas in `events/attestation.*.yaml`.

### Consumed
- `credit-types.md` — GPU credit tiers.
- `openapi/credit.yaml` — for mint/burn.
- `events/compute.usage.v1.yaml` — for payout calculation.

## 5. Local dev

- `services/supply-service/` runs on `:8005`.
- `make seed` creates mock partner DC (`mock-partner-1`, tier=H100-80GB, count=8, state=active)
  with a mock attestation record.
- `make test-payout` runs a synthetic month's worth of consumption and asserts the payout
  calculation matches the agreed formula.

## 6. Dockerfile

`services/supply-service/Dockerfile` → `deploy/docker/Dockerfile.go-service`.

The partner-DC agent (a separate small binary that runs at partner sites) has its own
Dockerfile: `services/supply-service/agent/Dockerfile`. It's small, distroless, and only needs
egress to `prod-real.1trade.io:443` over mTLS.

## 7. Deploy

- `services/supply-service/`: K8s Deployment, 3 replicas.
- Partner agents: deployed at partner DCs; we provide the binary + install script + mTLS cert
  bootstrap.
- Vault: holds escrow account credentials, partner-signing-key material, bond-account references.
- Network policy: only Kong → supply-service, supply-service → credit-ledger, supply-service →
  external (for wire transfers, attestation services).

## 8. Conventions

- **v1 backing = 1Trade's own DC** (trivially attestable — bootstrap trust on hardware you control).
  External DCs onboarded through the full stack + bond afterward.
- **Build the trade-escrow data model FOR REAL in v1** even though it's stubbed — so the Phase 2
  switch-on is a config flip, not a redesign.
- **Don't overclaim attestation**: we prove genuineness, presence, performance-at-class, and
  delivery. **NOT** "verifiable arbitrary-computation correctness." Customer-facing copy in
  `trading-frontend` must respect this.

## 9. Hard boundaries

- Don't schedule GPUs (that's `compute-platform`).
- Don't hold balances (that's `credit-ledger`).
- Don't set credit conversion rates (that's a contract, not policy).

## 10. Definition of done

- Credits are dated + ≥100% backed; proof-of-reserves published and accurate.
- Attestation layers implemented and verified on own DC + first partner DC.
- Escrow streams correctly with holdback; first partner payout cycle wired without dispute.
- Mint/burn keeps the backing ratio honest (test: try to over-issue; system refuses).
- Trade-escrow path designed and stubbed; Phase 2 switch-on is a flag flip.
