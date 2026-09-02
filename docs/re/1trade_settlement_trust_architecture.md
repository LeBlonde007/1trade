# 1Trade — Settlement & Trust Architecture

> **Scope:** how a GPU credit becomes trustworthy enough that real money trades for it — the layer between "we minted a credit" and "compute was delivered and the datacenter got paid."
> **Sits alongside:** the Credit Architecture & Trading Model work. That doc defines *what* trades (the instruments); this doc defines *how settlement and integrity work* so the instruments are believable.
> **Status:** design proposal. Open questions for Tai in §10.

---

## 1. The core problem

A GPU credit is only worth money if a buyer believes three things:

1. The GPU behind it is **real** — a genuine NVIDIA device of the claimed class, not a spoof or a weaker card.
2. The capacity is **actually there and not oversold** — dedicated to back this credit, not triple-booked.
3. When they redeem, **compute is delivered** — and when they don't, **the datacenter still gets paid fairly**.

Everything below exists to manufacture that belief. The honest framing: we are not building "verifiable compute" in the cryptographic sense (proving an arbitrary customer workload executed correctly). That problem is largely unsolved and has sunk well-funded attempts. We are building something narrower and tractable — proof of **genuineness, presence, performance, and delivery** — which is all a credit market actually requires.

---

## 2. The instrument: a dated, class-standardized contract

The single most important design decision, because it resolves the central tension: a GPU-hour is **perishable**. An unused H100-hour from last Tuesday is gone forever — it cannot be stored. So a credit cannot be an evergreen, perpetual claim; the underlying is consumed (or wasted) in real time.

The fix is the same one commodity markets use for perishable goods — make the credit a **dated contract with a delivery window**, like a future:

```
1 credit = 1 × [H100-80GB]-hour, deliverable in [window: e.g. June 2026]
```

| Field | Example | Why |
|---|---|---|
| GPU class | H100-80GB | Standardized — fungible within a class. NOT by serial number. |
| Unit | 1 GPU-hour | The atomic tradeable quantity |
| Delivery window | June 2026 | Bounds the perishability; defines settlement date |
| Settlement | physical *or* cash | Redeem for compute, or net against the index |
| Backing pool ref | (internal) | Which reserved capacity stands behind it |

**Standardize by class, not by machine.** Your instinct — "credits redeemable only where GPUs are dedicated to back them" — is the right *trust* instinct, but taken literally (credit tied to DC-A's specific reserved boxes) it fragments liquidity into a dozen thin, non-interchangeable markets and kills the exchange. So: the **dedication lives at the reserve layer** (real GPUs are reserved to back outstanding credits), but the **trader holds a standard instrument**. On redemption, the platform routes to *any* qualifying attested node in the backing pool. Backed by dedicated capacity — your requirement holds — yet liquid and fungible.

Until its window, the contract trades freely. At the window it either **physically settles** (holder redeems for real compute) or **cash-settles** against the published index. Most holders are traders who never touch a GPU — they cash-settle.

---

## 3. Backing-ratio policy

| Mode | Rule | When |
|---|---|---|
| **Full backing (≥100%)** | Outstanding credits in a window ≤ attested reserved capacity for that window | **v1 default** |
| Fractional backing | Outstanding credits > reserved capacity (bet on non-simultaneous redemption) | Deliberate later decision — carries bank-style run-risk. Not v1. |

**Proof of reserves.** Publish, per delivery window and GPU class: attested reserved capacity vs. outstanding credits. This is the honest, auditable backbone — the equivalent of an exchange showing its warehouse receipts. It is also the single best trust signal you can give institutional buyers.

> v1 keeps it strictly ≥100% backed. Fractional backing is a financial product decision to make *later, on purpose*, with disclosure and risk controls — never a default the system drifts into.

---

## 4. The attestation stack — proving the GPU is real

No single check is sufficient; each layer catches what the previous misses. Defense in depth.

| Layer | What it proves | What it catches | Mechanism |
|---|---|---|---|
| **1. KYB + facility certs** | The DC is a real legal entity with real allocation | Shell entities, obvious fraud | Legal onboarding, Tier rating, SOC 2 / ISO, proof of NVIDIA allocation |
| **2. Hardware attestation** | Device is a genuine NVIDIA GPU of model X, firmware intact | Spoofed `nvidia-smi`, A100-as-H100, faked VMs | NVIDIA device root-of-trust signed attestation report (H100/H200+) |
| **3. Randomized challenge-response** | GPU performs at its claimed class *right now*, not overcommitted | Throttling, oversubscription, shared/weaker cards | Nonce-seeded sealed benchmark; correct result required *within the throughput/latency envelope* of the class. Nonce defeats precompute. |
| **4. Continuous telemetry** | It stays genuine and healthy over time | Degradation, drift, quiet oversubscription | Agent streams utilization, power draw, ECC errors, clocks, NVLink topology. Power inconsistent with claimed load → audit. |
| **5. Staked performance bond** | Lying is economically irrational | *Rational* fraud (where the math otherwise favors cheating) | DC posts collateral, slashed on proven fraud or SLA breach |

The logic of the stack: **attestation proves what the hardware is, the challenge proves it's performing, telemetry proves it stays performing, and the bond makes fraud cost more than it pays.**

---

## 5. Escrow & payment flow

Neither extreme works. Pay the DC in full up front → you've paid for undelivered compute and carry fraud risk. Pay only at the very end → you starve the DC's cash flow and they won't supply. The answer is **escrowed, streamed on metered delivery, with a holdback.**

```
Buyer cash ──► CUSTODIAL ESCROW (not the DC)
                    │
                    │  released in installments as credits are
                    │  redeemed AND compute is metered (daily/weekly)
                    ▼
              DATACENTER  ◄── minus platform fee, minus holdback
                    │
   Holdback (10–20%) retained through dispute window, then released
                    │
   BOND sits behind everything — forfeitable on proven fraud
```

**A DC is owed two distinct things — keep them separate:**

| Owed for | Paid how |
|---|---|
| **Delivery** — credits redeemed, compute actually consumed | Streamed from escrow per metered hour (the bulk of payment) |
| **Availability** — keeping dedicated GPUs on standby to back outstanding-but-unredeemed credits | A **reservation fee** for idle standby capacity |

Without the reservation fee, no DC will dedicate hardware to back credits that traders are sitting on — the standby capacity earns nothing while it waits. This fee is small relative to delivery revenue but it's what makes dedicated backing economically possible. (Who bears it — buyer premium, platform spread, or DC opportunity cost — is an open question, §10.)

---

## 6. The full lifecycle: mint → trade → redeem → settle → burn

| Stage | What happens | Integrity control |
|---|---|---|
| **1. Mint** | DC commits dedicated capacity to a delivery window → platform mints N standardized credits, fully backed | Capacity must pass the §4 attestation stack first; bond posted |
| **2. Trade** | Credits sold to buyers, then bought/sold on the market; price floats. Most holders never touch a GPU. | Order book, market maker (simulated in v1), surveillance |
| **3. Redeem** | A holder who wants compute redeems → platform allocates *any* attested qualifying node in the backing pool → meters delivery | Live attestation + telemetry during delivery |
| **4. Settle** | Metered delivery releases escrowed payment to the DC (minus fee, minus holdback). Cash-settled positions net against the index at the window. | Holdback released after dispute window |
| **5. Burn** | Redeemed and expired credits are destroyed | Keeps the backing ratio honest — supply can't exceed reserves |

---

## 7. Disputes & slashing

1. SLA breach (node down, underperforms, fails a live challenge mid-delivery) → buyer files a claim within the dispute window.
2. **Holdback** covers the immediate remedy (refund / re-route to another node).
3. Repeated or proven fraud → **bond slashed** → DC delisted from the backing pool.
4. Slashing events feed the DC's public reliability score, which feeds the reservation fee they can command. Good actors earn cheaper standing; bad actors price themselves out.

---

## 8. Bootstrapping trust: own datacenter first

This dovetails with the existing strategy ("start with GPU @ datacenter," "own datacenter as credible underlying"):

- **v1 backing = 1Trade's own datacenter.** You control the hardware, so attestation and telemetry are trivial to stand up and fully trusted. You bootstrap the market against capacity you can prove cold.
- **v1.5+ = vetted external DCs** onboarded through the full §4 stack + bond. The reservation-fee and slashing mechanics matter most here, where you don't control the hardware.
- This sequencing means the trust architecture is *tested on your own hardware first*, before you ever depend on a third party's honesty.

---

## 9. What we deliberately do NOT promise

Stating the boundary protects credibility:

- We do **not** prove that an arbitrary customer computation executed correctly end-to-end (the verifiable-compute / Gensyn problem). It is expensive, largely unsolved, and **not required** for a credit market.
- We **do** prove: the hardware is genuine, present, performing at class, and that delivered hours were actually delivered.
- Claiming the former invites scrutiny we can't satisfy and gains nothing. The latter is sufficient and defensible.

---

## 10. Open questions for Tai

1. **Delivery-window granularity** — monthly contracts to start, or weekly/quarterly too? (Affects liquidity vs. precision.)
2. **Default settlement** — physical or cash? (Lean: cash default, physical on explicit redemption.)
3. **Region as a contract dimension** — does "H100 in Tokyo" trade separately from "H100 in EU," or is region abstracted in v1? (Latency/compliance vs. fungibility.)
4. **Escrow custodian** — regulated third-party custodian vs. in-house? Ties directly to the jurisdiction question (US/Japan/Singapore) still open from the trading-layer work.
5. **Who bears the reservation fee** — buyer premium, platform spread, or DC opportunity cost?
6. **Backing buffer** — strict 100%, or a safety buffer (e.g. 110%) to absorb attestation failures without breaking redemption?
7. **Bond sizing** — fixed, or scaled to the DC's outstanding minted credits?

---

## 11. v1 scope vs. later

| Capability | v1 (paper trading) | Real-money testing | Later |
|---|---|---|---|
| Lifecycle (mint→burn) | Full data model, simulated | Real, on own DC | — |
| Attestation stack | Mock attestation; build the *interfaces* real | Layers 1–4 live on own DC | Full stack on external DCs |
| Escrow / streamed payment | Simulated ledger | Real escrow, own DC | Third-party custodian |
| Bond / slashing | Modeled | — | Live for external DCs |
| Proof of reserves | Published (against simulated reserves) | Published (against real own-DC reserves) | Across all DCs |

Build the **data model and interfaces for real in v1** even while the trading is paper. That way the switch to real-money testing on your own datacenter is a configuration change, not a rewrite — exactly the "paper trading customer-facing, real-money testing in parallel" posture from the trading-layer decisions.
