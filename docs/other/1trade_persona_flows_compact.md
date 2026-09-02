# 1Trade — Persona Demo Flows

Three demo scripts. Each line = one on-camera beat: route · action · VO.

## P1 · Jordan Park — Independent Quant (3:30)

New visitor opens an account and trades within two minutes.

1. `/` — Scroll hero → markets grid → AI Index card → audience split. Hover live footer ticker. VO: *"The commodity market for AI compute — one credit = one dollar of work."* Click **Open account**.
2. `/signup` — Email + password with strength meter. **Continue**. *(Should pass through D2 email-verify — see Missing.)*
3. `/onboarding/kyc` — 4 steps: identity → address → tax + investor type → review. Submit → green-check. VO: *"Light-touch KYC — credits are commodities, not securities."* **Continue to trading →**.
4. `/trade` — Dark Bloomberg grid. Pan candle chart · order book · tape · order form · positions. Topbar: EAI-IDX, USD balance, 🔔 **2**. Place $1,000 market buy → fill toast; bell ticks to **3**. VO: *"Real-time book, tape, institutional entry."* Press **⌘K** — *introduce palette*. Type "port" → Enter.
5. `/portfolio` — Hero P&L + benchmark bar. Chart ranges 1D · 1W · 1M · 3M · YTD · ALL. Toggle dashed *AI-INDEX* overlay. Positions table · donut · allocation · activity. VO: *"Attribution vs the index is built in."*
6. **🔔 drawer** — 6 cards (Order filled, Price alert, Budget, API key, Maintenance, Welcome). Chips: All · Unread · Trades · Account · Alerts. **Mark all read**. Esc.
7. `/wallet` — Cash + AI-IDX + sub-credit balances · ledger. VO: *"Conversion is one click."* **Buy credits**.
8. `/wallet/buy` — Amount → method (card / wire / ACH) → confirm. Live receipt. VO: *"Instant on card, T+1 on wire."* Save & exit.
9. *(Optional 20s)* `/benchmark` — TOC · formula · historical chart. VO: *"Every constituent and audit pointer is public."*
10. Return to `/trade`. Hold on ticker. Cut.

## P2 · Maya Chen — Enterprise Buyer / VP Eng (4:00)

Procures compute at scale. Cares about budget, seats, audit, uptime.

1. `/` — Hover **For AI companies**. ⌘K → `/enterprise/onboarding`.
2. `/enterprise/onboarding` — CSM card (email · Slack · phone). 6-step activation: MSA ✅ · credit ✅ · billing ✅ · IdP ◐ · invite team ☐ · first instance ☐. Right rail: MSA · pricing · DPA · order form. VO: *"Every check unlocks usage."* **Invite team**.
3. `/enterprise/teams` — 8 members · roles (Owner · Admin · Trader · Viewer) · per-seat budgets · org spend. **Invite member** modal (role · budget · scopes). VO: *"Per-seat caps keep finance in the loop."*
4. `/compute` — 8 instances across US-EAST, US-WEST, EU-CENTRAL, AP-SOUTH. Status (RUNNING · PROVISIONING · STOPPED) · per-GPU utilization · spend + headroom. VO: *"Every instance, real-time utilization and cost."* **+ New instance**.
5. `/compute/new` — GPU (H100 · H200) → count → region → image → SSH key → budget. Live price calc. VO: *"Spot the market or reserve via the index — same platform."* Launch → back.
6. `/inference` — 10 models tagged text · speech · image · video · embed. Playground: system + user prompt, streaming output, metadata sidebar. Tab to **Code** → Python / cURL / JS. VO: *"Same credit — trading floor or API."*
7. `/settings#api` — 5 keys with scopes. **+ Create key** → scopes + expiry → one-time `sk_live_…` + copy. **Audit log →** *(dead — see Missing).* VO: *"Scoped, expiry-bound, audited."*
8. *(30s)* 🔔 → **Budget alert** ("AI-Research-Team 78% of June"). VO: *"The loop closes."* Cut.

## P3 · Tom Reyes — Datacenter Partner (2:30)

Sells GPU-hours into the venue. Cares about fill rate and payouts.

1. `/` — Scroll to **For datacenters**. Type URL `/datacenter` (unlinked from public nav).
2. `/datacenter` — Partner header (Northstar DC · Reno NV · `pdc_7c2a…`) + CSM contacts. KPIs: capacity sold MTD · fill-rate · payout pending · next settlement. Capacity grid (racks × clusters · LIVE / MAINT / OFFLINE). Orders table by credit family. Settlement statements (CSV). VO: *"What's online, earning, just settled."* **+ Register partner**.
3. `/datacenter/register` — Org → site (region · PUE · SOC 2 / ISO 27001 / Tier III) → capacity (GPU · count · windows) → pricing floor + throttle → banking → review. VO: *"Documentation pack, not a credit card."* Cancel → back.
4. *(Optional 30s)* May 2026 statement → CSV. VO: *"Every fill is a settlement line."*
5. Hold on KPIs. Cut.

## Missing screens — build before shooting

- **D2 Email verify + welcome** — P1 currently jumps signup → KYC.
- **D3 Audit log** — P2 ends a beat on a dead link.
- **D4 Billing dashboard** — Enterprise asks "where are invoices?"; `/wallet` conflates cash with billing.

## Shoot order: 3 → 1 → 2

Shortest first; palette + drawer introduced in P1 are reused in P2.
