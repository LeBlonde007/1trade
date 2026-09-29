# Test every flow locally

A click-through checklist for each user flow, on the local stack. Every flow here was run end to end
on 2026-09-28 against a freshly migrated database (browser automation plus the helpers below).

- **Bring-up:** [RUN_LOCAL.md](./RUN_LOCAL.md) §1–2 (`make up`, then `npm run dev` in `1Trade Frontend/`).
- **Automated tests:** [TESTING.md](./TESTING.md).

Everything is **paper** (`is_paper`): no real money moves, and the exchange stays in paper mode behind
the licence gate (order entry answers `EXCHANGE_PAUSED`).

## Local-only helpers

Some flows finish outside the web app: a bank clears a debit, finance matches a wire, a datacenter's
agent proves its GPUs, operations set payout terms. Two helpers play those parts on your machine.

| Helper | What it does |
|---|---|
| `node scripts/dev/ops.mjs` | Back office: `ach-clear`, `ach-fail`, `wire-received`, `attest`, `payout-terms`, `payout-cycle`. Run it with no arguments for usage. It reads the service token from the cluster (`kubectl`), or from `$SERVICE_TOKEN`. |
| `go run ./scripts/dev/test-idp` | A SAML identity provider on `http://127.0.0.1:18090`. Sign in as any email. Local only. |

**What the Tiltfile adds locally.** These values exist only in local dev, never in `deploy/k8s/*/base`,
which the public sandbox builds on:

- a Stripe webhook secret, so ACH can clear;
- a test bank, so wires can be issued;
- `SSO_DEV_SKIP_DNS=1`, so an SSO domain verifies without a real DNS record. It does nothing outside
  `TRADE1_ENV=dev`;
- an attestation trust key, generated into `.dev/` (gitignored) on first run.

Keys and IDs live under `.dev/`. Delete that folder to start fresh.

**Where to find IDs.** Purchase IDs are on `/enterprise/billing` → Purchase history (Reference column).
Source IDs are on `/datacenter`.

**Emails** (verification, invites) land in **Mailpit**: `http://localhost:8025` (a link in the Tilt
UI). Login does not wait on email verification locally.

---

## 1. Sign up, sign in, 2FA

1. **Sign up.** Open `/signup` and create an account.
   - Expect: you land in onboarding. The wallet shows 25 paper text credits and $10,000 paper cash.
2. **Verify your email.** Open the email in Mailpit and follow the link to `/onboarding/verify`.
3. **Turn on 2FA.** Open `/settings` → Two-factor authentication.
   - Scan the QR code with any authenticator app (1Password, Google Authenticator…) and enter a code.
   - Expect: 10 recovery codes. Save them.
4. **Sign in with 2FA.** Sign out, then sign in again.
   - Expect: after your password, the app asks for a code.
   - Try a wrong code: "That code is not valid."
   - A recovery code also works, once.

## 2. Team: sub-accounts, invites, roles

1. **Create a sub-account.** Open `/enterprise/teams` and create a sub-account (e.g. "Research").
   - Fund it from the main balance.
   - Expect: the balances move from main to the sub-account.
2. **Invite a member.** Invite `bob@…` as **engineer**, assigned to the sub-account. The invitation
   email is in Mailpit.
3. **Accept.** Open the link in a private window, choose a password, and join.
   - Expect: Bob is signed in as engineer, on the sub-account.
   - Opening the same link again fails (404): links are single-use.
4. **Check roles.** As Bob, try to invite someone.
   - Expect: refused. Only admins invite.
5. **Check the audit log.** Open `/enterprise/audit`. The invite, the acceptance and the sub-account
   funding are all recorded.

## 3. SAML single sign-on

1. **Start the test IdP.** From the repo root: `go run ./scripts/dev/test-idp`.
2. **Connect it.** As an admin, open `/enterprise/sso`.
   - Paste the XML from `http://127.0.0.1:18090/metadata`.
   - Domains: `example.com`. Role: engineer. Save.
3. **Verify the domain.** Section 3 lists `example.com` as pending, with its TXT record.
   - Before verifying, SSO sign-in for `@example.com` is refused. Nothing is routed to an unproved
     domain.
   - Click **Verify**. Expect: "verified".
     - Locally the DNS check is skipped.
     - Anywhere else, the TXT record `_1trade-verify.example.com` must exist.
4. **Sign in with SSO.** In a private window, open `/login` → **Sign in with SSO (firm accounts)** →
   enter `alice@example.com` → type the same email on the test IdP's form.
   - Expect: you are in `/console` as alice, with the engineer role, created on first sign-in.
5. **Negative case.** Turn off "Create members on first sign-in" and sign in as a new email.
   - Expect: "You have no account yet — ask your admin to invite you."

## 4. Buying credits: card, ACH, wire (USD only)

All three start at `/wallet/buy`. Pick a credit type (e.g. H100 GPU-hours) and an amount.

1. **Card.** Pay by card.
   - Expect: locally it settles at once (mock Stripe), and the balance goes up.
2. **ACH.** Pay by **ACH bank debit**.
   - Expect: "credits arrive when the debit clears". The purchase is `pending`/`processing` and the
     balance is unchanged.
   - `node scripts/dev/ops.mjs ach-clear <purchase-id>`
     - Expect: the purchase is `paid` and the credits are booked. Running it again changes nothing.
   - `ach-fail <purchase-id>` on another ACH purchase.
     - Expect: `failed`, and nothing booked.
3. **Wire.** Pay by **Wire transfer**. Wires start at $1,000.
   - Expect: an invoice with the exact USD amount, a reference, and the local test bank's details.
   - `node scripts/dev/ops.mjs wire-received <purchase-id> <amount>`, with the exact amount from the
     invoice (e.g. `2990.00`).
     - Expect: `paid`, and the credits are booked.
   - A wrong amount is refused (409) and nothing is booked.

## 5. Compute: instances, reserved capacity, clusters

Buy some H100 GPU-hours first (§4).

1. **Launch an instance.** Open `/compute/new`.
   - Expect: it appears on `/compute`, and it meters GPU-hours while it runs.
2. **Reserve capacity.** Open `/compute/reserve`.
   - Pick 8 × H100 for 1 month. Expect: the quote shows the discount and the prepaid amount.
   - Reserve. Expect: the prepaid credits leave the wallet at once.
   - New instances now use reserved GPUs first. The side card shows "In use now x / 8".
3. **Clusters.** Open `/compute/clusters`.
   - Create 24 × H100 (3 nodes, InfiniBand). Expect: the nodes list with their `ssh` lines.
   - Sizes are multiples of 8, up to 32 self-serve. 40 GPUs answers "contact sales"; 20 GPUs is refused.
   - Terminate the cluster.

## 6. Datacenter partner: register, attest, get paid

Use a separate account for the partner (it is a different tenant).

1. **Register GPUs.** Open `/datacenter/register` and register 16 × H100 (region, SLA tier).
   - Expect: the source appears on `/datacenter` as "Awaiting…". It cannot take work until it is attested.
2. **Attest.** `node scripts/dev/ops.mjs attest <source-id> <partner-email> <partner-password>`.
   - It passes KYB, bond, the signed GPU identity report and the timed challenge.
   - Expect: "attestation complete: true; source state: active". The dashboard shows 5 / 5 and
     "Taking work".
3. **Set payout terms.** `node scripts/dev/ops.mjs payout-terms <partner-email> <partner-password>`.
4. **Close a cycle.** Once the source has served some GPU time,
   `node scripts/dev/ops.mjs payout-cycle 24`.
   - Expect: a statement under Payouts, with gross, fee, holdback and released amounts. It is marked
     simulated (paper).
   - A statement can be disputed inside its window.

## 7. Inference: catalogue, playground, API

1. **Playground.** Open `/inference` and chat with a model.
   - Expect: a reply, with tokens metered against your text credits.
2. **Catalogue.** Open `/compute/catalog` for the GPU catalogue. The model list is on the inference
   page, or via `curl localhost:8085/v1/models`.
3. **API key.** Create a key in the console (Settings → API keys), then call the OpenAI-compatible API:

   ```bash
   curl localhost:8085/v1/chat/completions -H "Authorization: Bearer $KEY" \
     -H 'content-type: application/json' \
     -d '{"model":"llama-3.1-8b","messages":[{"role":"user","content":"hi"}]}'
   ```

## 8. Model load and unload (F11 hot-swap)

The cluster's gateway serves the mock backend. To watch models load and swap on one GPU, run a second
gateway on your machine with the example pool. It has 1 GPU and two models that don't fit together,
running on the CPU stub runtime.

```bash
# from the repo root
export PLATFORM_JWT_SECRET=$(kubectl get secret platform-auth -o jsonpath='{.data.PLATFORM_JWT_SECRET}' | base64 -d)
export SERVICE_TOKEN=$(kubectl get secret platform-auth -o jsonpath='{.data.SERVICE_TOKEN}' | base64 -d)
(cd services/inference-gateway && go build -o ../../bin/igw ./cmd/inference-gateway)
INFERENCE_ADDR=127.0.0.1:18085 PLATFORM_CORE_URL=http://localhost:8001 CREDIT_LEDGER_URL=http://localhost:8002 \
  INFERENCE_POOL="$(cat scripts/dev/inference-pool.json)" ./bin/igw
```

Then call `localhost:18085/v1/chat/completions` with your API key (§7):

1. `llama-3.1-8b`.
   - Expect: 200 after about 0.2 s (a cold load); then fast.
2. `qwen2.5-1.5b` straight away.
   - Expect: **503 with `Retry-After`**. The loaded model is younger than `min_residency` (5 s here),
     so it is not evicted.
3. `qwen2.5-1.5b` again after 5 s.
   - Expect: 200. llama was evicted and qwen loaded. The gateway log shows `model loaded` with
     `cold_start_ms`.

Usage events are not published from this host gateway (it has no NATS), so metering is not tested
here. Use the cluster's gateway (§7) for that.

## 9. Exchange: paper trading

Sign up as a **Trader** (the sidebar then shows Portfolio, Trade, Markets, History, Index, Wallet) and open **Markets** (`/markets`).

- Expect: a **PAPER** banner ("Paper trading is open … real money is paused") and a **Trade →** link
  on each market.

1. **Trade.** Open `/trade?product=H100-SPOT`.
   - Expect: a real book. The paper liquidity account quotes three levels a side around about 2.99.
   - "Available" shows your $10,000 of paper cash.
2. **Market buy.** Buy 5 at market.
   - Expect: "Filled 5 at 3.0x".
   - The fill appears on the tape and in Fills. Positions shows long 5.
   - Paper cash drops by the cost plus the 1% taker fee (check `/wallet`).
3. **Limit order.** Place a limit buy 2 at 1.00.
   - Expect: "Resting on the book at 1.00". Orders shows it as open.
   - The cash for it is held: $2.02 is locked.
   - Cancel it. Expect: cancelled, and the hold is released.
4. **Close.** In Positions, click **Close**.
   - Expect: a market sell flattens it. Realized P&L shows the round trip, fees included.
5. **Chart.** Bars with trades are coloured; bars with no trade show the reference price in grey.
   The engine labels every bar.
6. **Detail and history.** `/markets/h100-spot` shows the same book, tape and 24h stats.
   `/portfolio` and `/history` show your paper fills.
7. **Restart.** Restart the engine (e.g. `kubectl rollout restart deploy/matching-engine`).
   - Expect: your orders, fills and positions are all still there. They are rebuilt from the journal.

Real money stays paused. An account whose token is not paper gets **503 EXCHANGE_PAUSED** on every
order. Paper trading uses paper cash and credits only. `/benchmark` shows the public index
methodology; the index values are still provisional.

---

## If something looks wrong

| Symptom | Cause |
|---|---|
| A page shows no data and the BFF log shows `ECONNREFUSED 127.0.0.1:80xx` | A Tilt port-forward is down. Run `tilt up`, or `kubectl port-forward` (RUN_LOCAL.md §1). |
| ACH never clears | Tilt was started before the Tiltfile change. Restart `tilt up` so platform-core gets the webhook secret. |
| "wire transfers are not available" | Same: platform-core is missing the test bank values. |
| `attest` ends "complete: false" | compute-control does not trust your key. Restart `tilt up`: it reads `node scripts/dev/ops.mjs pubkey`. |
| SSO Verify says the TXT record is missing | `SSO_DEV_SKIP_DNS` isn't set, or `TRADE1_ENV` isn't `dev`. |
| SSO sign-in returns "no account" | Create members on first sign-in is off. Invite the user, or turn it on. |
