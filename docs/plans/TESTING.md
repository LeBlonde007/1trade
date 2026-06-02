# Testing Exascale

How to verify everything built so far. Two ways to read this:

- **Part 1 — Frontend by persona (dev mode).** Click-through journeys for each of the app's three
  personas. Start here.
- **Part 2 — Backend / CLI / automated.** `curl`, the `exascale` CLI, and `go test` — the proof
  underneath the UI.

Covers tags **v0.1.0 → v0.1.5** (M1) and **v0.2.0 → v0.2.8** (M2). Pair with
[RUN_LOCAL.md](./RUN_LOCAL.md) (bring-up) and `CHANGELOG.md` (what each tag shipped).

---

## 0. Run the frontend in dev mode

The web app is **always live** — there is no mock mode. Bring the stack up first (per RUN_LOCAL.md so
`:8001/:8002/:8085` answer), then:

```bash
cd "Exascale Frontend"
npm run dev                              # → http://localhost:3000 (needs the stack up)
```

Wired screens render real platform data (with loading/empty states). Screens whose backend isn't built
yet (the paused exchange, datacenter/compute supply) still show placeholder data — flagged below.

### The three personas

You pick a persona during onboarding (**`/onboarding/welcome` → `/onboarding/tour`**); it persists in
`localStorage` and tailors the sidebar/nav. The **account-type card on `/signup`** sets this persona
for you on submit (Trader → `trader`; AI Company / Enterprise → `enterprise`); you can still switch it
via the sidebar persona pill.

| Persona | Who | Surface | Live in local mode? |
|---|---|---|---|
| **AI Company** | Maya Chen, VP Eng | inference · wallet · keys · console | ✅ **Yes — the live journey** |
| **Datacenter** | Tom Reyes, Capacity ops | datacenter · compute supply | ⬜ Showcase (supply backend F12+ not built) |
| **Trader** | Jordan Park, Quant | trade · markets · portfolio | ⏸ Showcase (exchange **paused**, license-gated) |

> **What "showcase" means:** the screen renders with realistic simulated data even in local mode,
> because its backend isn't built/enabled yet. Great for design review; **not** a live test. Only the
> **AI Company** journey exercises real services today.

---

# Part 1 — Frontend by persona

## 1A. AI Company — Maya Chen  ✅ the live end-to-end journey

> Run with the stack up (`npm run dev` — always live). This is the path that actually moves real credits.

**Step 1 — Create an account / sign in.** Go to `/signup`.
- The page shows a **3-way account-type selector** (Trader · AI Company · Enterprise) + Work email +
  Password + OAuth buttons. The **account-type sets your runtime persona + onboarding path** on submit
  (Trader → Light KYC; AI Company / Enterprise → straight to the console). It isn't stored on the
  backend; the **OAuth buttons are placeholders**. Pick **AI Company**, enter email + password, accept
  terms → **Open Account**.
- This creates a **real account** (BFF `/api/auth/signup` → platform-core; httpOnly session cookie set)
  and **emails a verification link to Mailpit**, then routes to **`/onboarding/verify`**. Open
  **Mailpit → http://localhost:8025**, open the *"Verify your Exascale email"* message, and click its
  link — it consumes the token and lands you on **`/console`**. (You're already authenticated from
  signup, so you can also just open `/console` directly.)
- **Easiest for testing:** once the account exists, use **`/login`** — it signs in and lands you
  straight on **`/console`**. Logout clears the session and bounces you back to `/login`.

**Step 2 — Persona.** Your signup account-type already set the runtime persona (so picking *AI Company*
gives you the inference/wallet/keys sidebar and skips trader KYC). To change it later, use the
**sidebar persona pill** or re-run **`/onboarding/tour`**. The onboarding is persona-aware: **Trader**
gets the 4-step Light-KYC identity flow; **AI Company** is verified → straight to the console;
**Datacenter** is pointed at capacity registration (KYC doesn't apply).

**Step 3 — Browse the model catalog.** Open `/inference`. The model picker lists the **live catalog**.
- Expect: `llama-3.1-70b`, `llama-3.1-8b`, `whisper-large-v3` with per-unit **credit prices**.
- Live check: prices come from the gateway `/v1/models` via `/api/catalog`.

**Step 4 — Get credits.** A fresh tenant starts at zero.
- **In dev, the reliable way is the service-token mint** (Part 2 §2.3) — run it once for your tenant.
- Or via UI: `/wallet/buy` → pick an amount → checkout. In dev this returns a dev (Stripe-test) URL and
  the mint only completes on the (absent) webhook, so prefer the mint.
- After crediting, `/wallet` shows the real balance.

**Step 5 — Run inference (the core loop).** `/inference` → type a prompt → **Send**.
- Expect: a completion (CPU-stub reply) + token usage. The turn shows the **real credit cost**
  (`tokens / 1000 × catalog price`); the **session meter** sums credits spent and shows your live
  `text` balance. _(v0.2.7)_
- ~1s later, the wallet balance drops by that cost (debit is async via NATS).

**Step 6 — Wallet: convert + movements.** `/wallet`.
- Open the **Convert** drawer → from **AI Credits** → to **Text**, amount `100` → **Convert**.
- Expect: a success toast; balances update; the **Recent movements** table shows the two **real ledger
  legs** (`−100 ai_index`, `+82.227321 text`). _(v0.2.5 + v0.2.6)_
- The drawer shows the live rate + house spread; an unsupported pair shows "no live rate".

**Step 7 — Settings: API keys.** `/settings` → **API Keys** → **Create new key** → name + scopes → Generate.
- Expect: the **one-time secret** is revealed (copy it now — never shown again); the key appears in the
  list (metadata only); **Revoke** flips it to "Revoked". _(v0.2.8)_
- That secret is a real inference credential: it authenticates `/v1/chat/completions`.

**Step 8 — Console (one-screen ops).** `/console`.
- Expect: catalog, run-inference, wallet, budget, purchases, and the audit trail — all live, on the
  dark institutional design system.

**Quick pass/fail:** signup → catalog → (mint) → run → cost+meter → convert → movements → key. If each
shows real data and balances move, the AI-Company surface is green.

## 1B. Datacenter — Tom Reyes  ⬜ showcase

Screens: `/datacenter`, `/datacenter/register`, `/compute`, `/compute/new`.

**Walkthrough (mock or local — same):** open `/datacenter` (capacity overview) → `/datacenter/register`
(onboard a GPU cluster: tier, count, region) → `/compute` (rentable GPU inventory).

> **Status:** the supply/compute control plane (**F12+**) isn't built yet, so these are **mock
> showcases** even in local mode — realistic data, no live mutation. Review them for design/flow; they
> are not a live backend test.

## 1C. Trader — Jordan Park  ⏸ showcase (exchange paused)

Screens: `/trade`, `/markets`, `/markets/[slug]`, `/portfolio`, `/history`.

**Walkthrough (mock):** `/markets` (product list) → `/trade` (order book + chart + tape, animated) →
`/portfolio` (positions + P&L) → `/history`.

> **Status:** the exchange/trading layer is **paused** (license-gated) per the GTM pivot. These screens
> are the headline-quality **mock showcase** (Brownian-motion prices, power-law book, live-ticking
> tape) — **not** wired to a matching engine. Use them for the "feels like a real market" review, not
> as a live test.

---

# Part 2 — Backend / CLI / automated

The UI calls these through its BFF; here you hit them directly. Bring the stack up (RUN_LOCAL) and
confirm `:8001/:8002/:8085 → 200` on `/healthz`.

**Shared setup** (a fresh tenant + JWT + the ledger service token, reused below):

```bash
PC=http://localhost:8001 ; LEDGER=http://localhost:8002 ; GW=http://localhost:8085
EMAIL="test+$(date +%s)@dev.test" ; PW='TestPw!123'
TOKEN=$(curl -s -X POST $PC/v1/auth/signup -H 'content-type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\",\"tenant_name\":\"QA\"}" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
TENANT=$(curl -s $PC/v1/auth/me -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["tenant_id"])')
SVC=$(kubectl get secret platform-auth -o jsonpath='{.data.SERVICE_TOKEN}' | base64 -d)
echo "tenant=$TENANT token=${TOKEN:0:18}…"
```

### 2.1 Auth — F02
```bash
curl -s -X POST $PC/v1/auth/login -H 'content-type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\"}" | python3 -m json.tool     # → { token, expires_at }
curl -s $PC/v1/auth/me -H "Authorization: Bearer $TOKEN" | python3 -m json.tool # roles:[admin], is_paper:true
curl -s -o /dev/null -w '%{http_code}\n' -X POST $PC/v1/auth/login \
  -H 'content-type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"wrong\"}"   # 401
```

### 2.2 Credit ledger — F05 (chain · idempotency)
```bash
curl -s "$LEDGER/v1/credits/balances?is_paper=true" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

curl -s -X POST $LEDGER/v1/credits/purchase -H "Authorization: Bearer $SVC" \
  -H "Idempotency-Key: qa-mint-$TENANT" -H 'content-type: application/json' \
  -d "{\"tenant_id\":\"$TENANT\",\"credit_type\":\"ai_index\",\"amount\":\"1000\",\"is_paper\":true}" | python3 -m json.tool
# replay the same Idempotency-Key → balance NOT doubled
curl -s -X POST $LEDGER/v1/credits/purchase -H "Authorization: Bearer $SVC" \
  -H "Idempotency-Key: qa-mint-$TENANT" -H 'content-type: application/json' \
  -d "{\"tenant_id\":\"$TENANT\",\"credit_type\":\"ai_index\",\"amount\":\"1000\",\"is_paper\":true}" \
  | python3 -c 'import sys,json;print("still:",json.load(sys.stdin)["balance_after"])'      # 1000.000000

curl -s "$LEDGER/v1/credits/audit/chain-verify?tenant_id=$TENANT" -H "Authorization: Bearer $SVC" | python3 -m json.tool
# → { "ok": true, "checked": N }
```
**§2.3 service-token mint** (used by the UI journey, Step 4): the `purchase` call above — set
`credit_type` to `text` (for inference) or `ai_index` (for conversion).

### 2.4 Conversion — F07
```bash
curl -s $LEDGER/v1/credits/conversion-rates | python3 -m json.tool
curl -s -X POST $LEDGER/v1/credits/convert -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: qa-conv-1" -H 'content-type: application/json' \
  -d '{"from":"ai_index","to":"text","amount":"100"}' | python3 -m json.tool   # debit -100 / credit +82.227321
```

### 2.5 Inference + metered debit — F08 / F09 / F08↔F05
```bash
curl -s -X POST $LEDGER/v1/credits/purchase -H "Authorization: Bearer $SVC" \
  -H "Idempotency-Key: qa-text-$TENANT" -H 'content-type: application/json' \
  -d "{\"tenant_id\":\"$TENANT\",\"credit_type\":\"text\",\"amount\":\"50\",\"is_paper\":true}" >/dev/null
curl -s $GW/v1/models -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
curl -s -X POST $GW/v1/chat/completions -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"model":"llama-3.1-8b","messages":[{"role":"user","content":"hi"}],"max_tokens":32}' \
  | python3 -c 'import sys,json;print("usage:",json.load(sys.stdin)["usage"])'
sleep 2   # async debit (v0.2.7 pull consumer)
curl -s "$LEDGER/v1/credits/balances?is_paper=true" -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;[print(" ",b["credit_type"],b["balance"]) for b in json.load(sys.stdin)["balances"]]'
```
Pre-flight 402: a tenant with **no** `text` credits → the same chat returns `402 INSUFFICIENT_CREDIT`.

### 2.6 API keys — F02
```bash
CREATE=$(curl -s -X POST $PC/v1/auth/keys -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"name":"qa key","scopes":["inference"]}'); echo "$CREATE" | python3 -m json.tool   # secret shown once
KID=$(echo "$CREATE" | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
curl -s $PC/v1/auth/keys -H "Authorization: Bearer $TOKEN" | python3 -m json.tool          # metadata only
curl -s -o /dev/null -w 'revoke %{http_code}\n' -X DELETE $PC/v1/auth/keys/$KID -H "Authorization: Bearer $TOKEN"
```

### 2.7 Billing (F06) · RBAC + audit (F03) — contract
- Billing: `POST /v1/billing/checkout` → `{ checkout_url }` (mock in dev); `GET /v1/billing/purchases`;
  budgets `GET/PUT /v1/billing/budget`. Webhook signature is unit-tested. See `openapi/platform-core.yaml`.
- RBAC: privileged endpoints require `admin` → non-admin gets **403**; admin actions emit
  `admin.action.v1` into the queryable audit log (`/v1/orgs…`, `/v1/audit…`).

### 2.8 CLI — F04
```bash
make cli                                  # → bin/exascale
EM="cli+$(date +%s)@dev.test"
./bin/exascale signup --email "$EM" --password 'pw' --name 'CliQA'   # v0.1.5 --password
./bin/exascale whoami
#   (mint credits for this tenant via §2.2, then:)
./bin/exascale catalog
./bin/exascale credits convert --from ai_index --to text --amount 100
./bin/exascale infer chat -m llama-3.1-8b "Define a GPU in one line"
./bin/exascale keys create --name production
```

### 2.9 Automated tests (no cluster)
```bash
for s in credit-ledger platform-core inference-gateway; do (cd services/$s && go vet ./... && go test ./...); done
(cd apps/cli && go vet ./... && go test ./...)
(cd "Exascale Frontend" && npm run typecheck)     # pre-existing chart-lib errors are unrelated
```

---

## Feature → verification checklist

| Feature | Tag(s) | Verify | State |
|---|---|---|---|
| F01 data plane + cluster | v0.1.0 | `kubectl get pods -A`; §0 health | ✅ |
| F02 auth (signup/login/me/logout) | v0.1.1 | UI 1A·1-2 / §2.1 / §2.8 | ✅ |
| F02 API keys | v0.2.0, v0.2.8 | UI 1A·7 / §2.6 | ✅ |
| F02 email verify, budgets | v0.1.3 | `openapi/platform-core.yaml` | 📄 |
| F03 accounts/orgs/RBAC + audit | v0.1.2 | §2.7 (403 + audit) / console | 📄 |
| F04 exascale CLI (+ `--password`) | v0.1.4, v0.1.5 | §2.8 | ✅ |
| F05 credit ledger (chain, idempotency) | v0.1.0+ | UI 1A·6 / §2.2 | ✅ |
| F06 billing (Stripe → mint) | v0.2.1 | UI 1A·4 / §2.7 | 🔁📄 |
| F07 credit conversion | v0.2.4 | UI 1A·6 / §2.4 | ✅ |
| F08 inference gateway | v0.2.0 | UI 1A·5 / §2.5 | ✅ |
| F08↔F05 metered debit | v0.2.0, v0.2.7 | UI 1A·5 (balance drops) / §2.5 | ✅ |
| F09 vLLM runtime (CPU stub) | v0.2.2 | §2.5 (real GPU: `GPU=1` + GPU node) | ✅ |
| F20 console + live wiring | v0.2.3–v0.2.8 | Part 1A | ✅ |

**Not built yet (don't test):** F12 compute control plane (Datacenter/Trader surfaces are showcase),
the exchange/trading layer (paused), real observability dashboards, real Stripe/email (M3).

> Legend: ✅ verified live · 🔁 runnable above · 📄 documented from the contract.
