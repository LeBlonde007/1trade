# F23 — Console v1.5 screens (frontend depth)

## Spec

The **v1.5 screen backlog** — the post‑MVP frontend catalog in
`docs/other/exascale_v15_screens_prompts.md` (Tiers E–N) — tracked as one feature so the screens are
in the plan, sequenced against the backend feature each one consumes.

This is the **screen‑by‑screen** tracker that `F20` (the console umbrella) does not itemize. Owning
agent: `trading-frontend`. Each screen is live‑only (no mock mode — see CLAUDE.md / v0.2.11); a screen
ships when (a) its backend feature exists, then (b) the page wires to it.

**Pivot alignment:** the **exchange** tiers (E — trading depth; the trading parts of K) stay under the
paused **KW01–KW05** track — designed, may exist as showcase, but not wired/active in Phase 1. The
**platform‑first** tiers (F money‑ops, G compute/inference, H enterprise, I datacenter) are the active
backlog; **J marketing**, **L help/mobile**, and the **N cross‑cutting** polish had **no plan feature**
before this doc.

Legend: ✅ built · 🟡 partial / showcase · ⬜ not started · ⏸ paused (Phase 2).

### Tier E — Trading depth ⏸ (Phase 2 — KW02/KW03; paused)

| ID | Screen | Route | Needs | Status |
|---|---|---|---|---|
| E1 | Order detail view | `/trade` drawer | KW03 matching | ✅ built (showcase drawer) |
| E2 | Watchlist / favorites | `/markets` | KW01 index, market data | ⏸ |
| E3 | Heat map | `/markets` | market data | ⏸ |
| E4 | Depth chart (cumulative) | `/trade` | KW03 book | ⏸ |
| E5 | Position detail + close | `/trade` drawer | KW03 | ✅ built (showcase drawer) |
| E6 | Advanced order types modal | `/trade` | KW03 | ⏸ |
| E7 | Risk / margin dashboard | `/portfolio` | KW03 + margin | ⏸ |

### Tier F — Money operations 🟡 (active — F06/F07/F18)

| ID | Screen | Route | Needs | Status |
|---|---|---|---|---|
| F1 | Withdrawal flow | `/wallet/withdraw` | F18 payouts + banking (M3) | ⬜ |
| F2 | Bank linking (Plaid‑style) | modal | banking (M3) | ⬜ |
| F3 | Credit conversion modal | `/wallet` | **F07 ✅ (backend live)** — UI only | 🟡 (wallet drawer exists; modal variant ⬜) |
| F4 | Transaction detail | ledger row → drawer | **F05 ✅** | ✅ built (drawer) |
| F5 | Recurring purchase setup | `/wallet/recurring` | F06 | ⬜ |
| F6 | Payment methods page | `/wallet/methods` | F06 billing | ⬜ |

### Tier G — Compute & Inference v1.5 🟡 (active — F08/F10–F13)

| ID | Screen | Route | Needs | Status |
|---|---|---|---|---|
| G1 | Compute instance detail | `/compute/[id]` | F12/F13 (GPU‑gated) | 🟡 (page exists, showcase) |
| G2 | SSH keys management | `/settings#ssh` | F12 | ⬜ |
| G3 | Job history | `/compute/jobs` | F12 | ⬜ |
| G4 | Model detail page | `/inference/models/[slug]` | F10 catalog | ⬜ |
| G5 | Fine‑tuning wizard | `/inference/finetune/new` | **no backend feature yet** | ⬜ |
| G6 | Batch processing console | `/inference/batch` | **no backend feature yet** (F08 batch) | ⬜ |
| G7 | API usage analytics | `/inference/analytics` | F08 metering + F05 | ⬜ |

### Tier H — Enterprise depth 🟡 (active — F02 SSO=M4 / F03 / F21)

| ID | Screen | Route | Needs | Status |
|---|---|---|---|---|
| H1 | Sub‑account detail | `/enterprise/teams/[id]` | F03 (sub‑accounts M4) | ⬜ (teams list exists) |
| H2 | Permission matrix | `/enterprise/permissions` | F03 RBAC | ⬜ |
| H3 | SSO / SAML config wizard | `/enterprise/sso` | F02 SAML (M4) | 🟡 (sso stub page exists) |
| H4 | Vendor risk packet generator | `/enterprise/risk-packet` | F21 SOC 2 | ⬜ |
| H5 | MSA / contract renewal | flow | sales (no backend) | ⬜ |

### Tier I — Datacenter partner v1.5 🟡 (active — F16–F19; M3+/M4+)

| ID | Screen | Route | Needs | Status |
|---|---|---|---|---|
| I1 | Settlement detail | `/datacenter/statements/[id]` | F18 payouts | ⬜ |
| I2 | Live capacity dashboard | `/datacenter/live` | F16 supply | ⬜ |
| I3 | SLA dashboard | `/datacenter/sla` | F17/F19 | ⬜ |
| I4 | Hardware lifecycle / assets | `/datacenter/assets` | F17 onboarding | ⬜ |
| I5 | Tax forms / 1099 | `/datacenter/tax` | F18 payouts | ⬜ |

### Tier J — Marketing / public / developer ❌→ now tracked (no backend; marketing track)

| ID | Screen | Route | Status |
|---|---|---|---|
| J1 | About | `/about` | ⬜ |
| J2 | Careers | `/careers` | ⬜ |
| J3 | Status page | `/status` | ✅ built |
| J4 | Changelog | `/changelog` | ⬜ |
| J5 | API documentation | `/docs/api` | ⬜ (M6 docs site) |
| J6 | Trust center | `/trust` | ⬜ (pairs with F21) |
| J7 | Blog / press | `/blog`, `/press` | ⬜ |

### Tier K — Admin (internal ops) ⏸/🟡 (KW04/KW05 paused + F22)

| ID | Screen | Needs | Status |
|---|---|---|---|
| K1 | Market‑making controls | KW04 market maker | ⏸ |
| K2 | Circuit breaker dashboard | KW03/surveillance | ⏸ |
| K3 | KYC review queue | F02/F22 | ⬜ |
| K4 | Trade surveillance | KW05 | ⏸ |
| K5 | AML / sanctions alerts | F22/KW05 | ⬜ |

### Tier L — Help, errors, mobile ❌→ now tracked (no backend)

| ID | Screen | Route | Status |
|---|---|---|---|
| L1 | Help center / KB | `/help` | ⬜ |
| L2 | Glossary | `/help/glossary` | ⬜ |
| L3 | Maintenance mode page | — | ⬜ |
| L4 | Rate‑limited / denied / suspended states | — | 🟡 (`/states` reference exists) |
| L5 | Mobile trading dashboard | responsive | ⬜ |
| L6 | Tour mode | `/onboarding/tour` | ✅ built (click + auto tour) |

### Tier M — Existing screens to round out 🟡 (F20 polish)

| ID | Screen | Status |
|---|---|---|
| M1 | Settings sections beyond Profile + API keys | 🟡 (shell exists; sections stubbed) |
| M2 | `/history` expanded | 🟡 (page exists) |
| M3 | Marketing footer links | ✅ built |
| M4 | `/markets` list page | ✅ built |
| M5 | Logout flow + avatar menu | 🟡 |
| M6 | SSO sign‑in on `/login` | ⬜ (F02 SAML M4) |
| M7 | KYC: actual document upload | ⬜ |
| M8 | `/benchmark` methodology — historical revisions | ⬜ |

### Tier N — Cross‑cutting frontend quality ❌→ now tracked (F20)

| ID | Item | Status |
|---|---|---|
| N1 | Keyboard chord coverage (command palette) | 🟡 (palette exists) |
| N2 | Skeletons on every list/table | ⬜ |
| N3 | Global toast region | ✅ built (`App/Toasts.vue`) |
| N4 | Uniform `<ConfirmDialog>` (2‑step for irreversible) | ⬜ |
| N5 | Empty / loading / error per route | 🟡 (`/states` reference; live‑only empties landing per v0.2.11) |
| N6 | Price‑tick flash animations (consistent) | 🟡 |
| N7 | Locale + currency (USD/EUR/GBP/JPY) | ⬜ |
| N8 | A11y pass (focus, ARIA, keyboard, SR) | ⬜ |

## Suggested shipping order (platform‑first; from the v1.5 doc, exchange items deferred)

1. **M5** logout + avatar menu · **M4** ✅ done · **L6** ✅ done
2. **F4** ✅ done · **F3** conversion modal polish · **G1** instance detail (when F12 lands)
3. **J3** ✅ done · **N3** ✅ done · **N4** ConfirmDialog · **N2** skeletons
4. **H3** SSO wizard + **M6** SSO sign‑in (M4, with F02 SAML)
5. **I1–I5** datacenter (with F16–F19) · **J1/J4/J6** marketing · **L1/L2** help
6. Exchange (Tier E, K‑trading) — Phase 2 under **KW02–KW05** (paused).

## Owning agent

`trading-frontend` (per‑screen), coordinating with the backend feature owner each screen consumes.

## Dependencies

Per‑screen (see "Needs" columns): F02 (SSO/SAML), F03 (orgs/RBAC/sub‑accounts), F05–F07 (money),
F08/F10–F13 (compute/inference), F16–F19 (datacenter), F21 (SOC 2 / trust), F22 (KYC/AML),
KW01–KW05 (exchange — paused). Plus the live‑only frontend baseline (v0.2.11).

## Acceptance criteria

- [ ] Every active‑track screen (Tiers F, G, H, I, J, L, M, N) wired to its real backend (no mock),
      with loading + empty + error states.
- [ ] Exchange tiers (E, trading‑K) remain showcase under the paused KW track until the license lands.
- [ ] Each screen meets the design system (tokens, mono/tabular numbers, sentence case, a11y AA).

## Milestone

M3 → M6 (frontend depth, post first‑customer), gated per‑screen by the backend feature it needs.
Exchange screens: Phase 2.
