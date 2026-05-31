# Testing Exascale — verify every shipped feature

A hands-on checklist to confirm everything built so far actually works. Three tiers:

1. **Automated** — `go test` + typecheck. No cluster needed. Fast.
2. **Platform API** — `curl` against the live services. Proves the backend loop.
3. **CLI + Web** — the two product surfaces, end-to-end.

Covers tags **v0.1.0 → v0.1.5** (M1) and **v0.2.0 → v0.2.8** (M2). Pair with
[RUN_LOCAL.md](./RUN_LOCAL.md) (bring-up) and `CHANGELOG.md` (what each tag shipped).

> Legend in the final checklist: ✅ verified this session · 🔁 runnable below · 📄 documented from the
> contract (`docs/contracts/openapi/`).

---

## 0. Prerequisites

Bring the stack up per **RUN_LOCAL.md** (`make up`, or the cluster is already running) and confirm the
forwards are live:

```bash
for p in 8001 8002 8085; do echo ":$p → $(curl -s -o /dev/null -w '%{http_code}' http://localhost:$p/healthz)"; done
# expect: :8001 → 200   :8002 → 200   :8085 → 200
```

Shared test setup — a fresh tenant + a JWT + the ledger service token (reused by the curl sections):

```bash
PC=http://localhost:8001 ; LEDGER=http://localhost:8002 ; GW=http://localhost:8085
EMAIL="test+$(date +%s)@dev.test" ; PW='TestPw!123'

# signup → JWT
TOKEN=$(curl -s -X POST $PC/v1/auth/signup -H 'content-type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\",\"tenant_name\":\"QA\"}" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
TENANT=$(curl -s $PC/v1/auth/me -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["tenant_id"])')
SVC=$(kubectl get secret platform-auth -o jsonpath='{.data.SERVICE_TOKEN}' | base64 -d)
echo "tenant=$TENANT  token=${TOKEN:0:18}…"
```

---

## 1. Automated tests (no cluster)

```bash
# Go services — unit + (DB-less) integration; integration tests self-skip without DATABASE_URL
for s in credit-ledger platform-core inference-gateway; do (cd services/$s && go vet ./... && go test ./...); done
(cd apps/cli && go vet ./... && go test ./...)        # F04 CLI client tests

# Frontend typecheck (Nuxt)
(cd "Exascale Frontend" && npm run typecheck)         # pre-existing chart-lib errors are unrelated
```

**Expect:** `ok` for `internal/domain`, `internal/store`, `internal/api` (ledger), the CLI `internal/client`, etc. Money math, hash chain, conversion floor, Stripe-signature, JWT all have unit tests.

---

## 2. Platform API (curl)

### 2.1 Health / readiness (F01/F05/F02/F08)
```bash
curl -s $PC/healthz -o /dev/null -w 'platform %{http_code}\n'
curl -s $LEDGER/readyz -o /dev/null -w 'ledger  %{http_code}\n'   # 200 = DB reachable
curl -s $GW/healthz -o /dev/null -w 'gateway %{http_code}\n'
```

### 2.2 Auth — F02 (signup/login/me/logout) 🔁✅
```bash
# login with the same creds → a fresh token
curl -s -X POST $PC/v1/auth/login -H 'content-type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\"}" | python3 -m json.tool   # → { token, expires_at }
# identity
curl -s $PC/v1/auth/me -H "Authorization: Bearer $TOKEN" | python3 -m json.tool   # email, tenant_id, roles:[admin], is_paper:true
# wrong password → 401 (and constant-time — no user enumeration)
curl -s -o /dev/null -w '%{http_code}\n' -X POST $PC/v1/auth/login \
  -H 'content-type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"wrong\"}"   # 401
```

### 2.3 Credit ledger — F05 (balances · mint · transactions · hash chain) 🔁✅
```bash
# fresh tenant has no balances
curl -s "$LEDGER/v1/credits/balances?is_paper=true" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# mint credits (service-token only; idempotent on the key)
curl -s -X POST $LEDGER/v1/credits/purchase -H "Authorization: Bearer $SVC" \
  -H "Idempotency-Key: qa-mint-$TENANT" -H 'content-type: application/json' \
  -d "{\"tenant_id\":\"$TENANT\",\"credit_type\":\"ai_index\",\"amount\":\"1000\",\"is_paper\":true}" | python3 -m json.tool
# replay same key → identical tx, balance NOT doubled (idempotency)
curl -s -X POST $LEDGER/v1/credits/purchase -H "Authorization: Bearer $SVC" \
  -H "Idempotency-Key: qa-mint-$TENANT" -H 'content-type: application/json' \
  -d "{\"tenant_id\":\"$TENANT\",\"credit_type\":\"ai_index\",\"amount\":\"1000\",\"is_paper\":true}" \
  | python3 -c 'import sys,json;print("balance_after still:",json.load(sys.stdin)["balance_after"])'   # 1000.000000

curl -s "$LEDGER/v1/credits/transactions?is_paper=true" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# hash chain intact for the tenant (service-token)
curl -s "$LEDGER/v1/credits/audit/chain-verify?tenant_id=$TENANT" -H "Authorization: Bearer $SVC" | python3 -m json.tool
# → { "ok": true, "checked": N }
```
**Invariants proven:** append-only, per-tenant hash chain verifies, fixed-point amounts (`NUMERIC(20,6)`), `(tenant, operation, idempotency_key)` dedupe, `is_paper` isolation.

### 2.4 Credit conversion — F07 (AI-index ↔ text) 🔁✅
```bash
curl -s $LEDGER/v1/credits/conversion-rates | python3 -m json.tool   # spread + seeded ai_index↔text rates
# convert 100 ai_index → text (rate 0.830579, 1% spread → 82.227321), needs an Idempotency-Key
curl -s -X POST $LEDGER/v1/credits/convert -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: qa-conv-1" -H 'content-type: application/json' \
  -d '{"from":"ai_index","to":"text","amount":"100"}' | python3 -m json.tool
# → { debit: ai_index -100 (→900), credit: text +82.227321 }
# unseeded pair → 422 no_rate ; insufficient → 402
curl -s -o /dev/null -w 'no_rate %{http_code}\n' -X POST $LEDGER/v1/credits/convert -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: qa-conv-x" -H 'content-type: application/json' -d '{"from":"ai_index","to":"video","amount":"1"}'   # 422
```

### 2.5 Inference + metered debit — F08 / F09 / F08↔F05 🔁✅
```bash
# need text credits first (the chat model bills `text`)
curl -s -X POST $LEDGER/v1/credits/purchase -H "Authorization: Bearer $SVC" \
  -H "Idempotency-Key: qa-text-$TENANT" -H 'content-type: application/json' \
  -d "{\"tenant_id\":\"$TENANT\",\"credit_type\":\"text\",\"amount\":\"50\",\"is_paper\":true}" >/dev/null

curl -s $GW/v1/models -H "Authorization: Bearer $TOKEN" | python3 -m json.tool   # curated catalog + Exascale pricing
# run inference (CPU stub locally) — OpenAI-compatible
curl -s -X POST $GW/v1/chat/completions -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"model":"llama-3.1-8b","messages":[{"role":"user","content":"hi in 3 words"}],"max_tokens":32}' \
  | python3 -c 'import sys,json;u=json.load(sys.stdin)["usage"];print("tokens:",u,"→ credits:",u["total_tokens"]/1000*5.0)'

sleep 2   # debit is async (NATS inference.usage.v1 → ledger pull consumer, v0.2.7 fix)
curl -s "$LEDGER/v1/credits/balances?is_paper=true" -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;[print(" ",b["credit_type"],b["balance"]) for b in json.load(sys.stdin)["balances"]]'
# text balance dropped by tokens/1000 × 5.0
```
**Pre-flight 402:** a tenant with **no** `text` credits running the same chat → `402 INSUFFICIENT_CREDIT` (the gateway checks balance before serving).

### 2.6 API keys — F02 🔁✅
```bash
CREATE=$(curl -s -X POST $PC/v1/auth/keys -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"name":"qa key","scopes":["inference"]}')
echo "$CREATE" | python3 -m json.tool          # → id, prefix exk_…, secret (shown ONCE), scopes
KID=$(echo "$CREATE" | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
curl -s $PC/v1/auth/keys -H "Authorization: Bearer $TOKEN" | python3 -m json.tool   # list = metadata only (no secret)
curl -s -o /dev/null -w 'revoke %{http_code}\n' -X DELETE $PC/v1/auth/keys/$KID -H "Authorization: Bearer $TOKEN"   # 200
```
The returned key secret also works as an inference credential: `curl $GW/v1/models -H "Authorization: Bearer <secret>"`.

### 2.7 Billing — F06 (Stripe checkout + budgets) 🔁📄
```bash
# checkout returns a (mock) Stripe URL; the real mint happens on the signed webhook.
curl -s -X POST $PC/v1/billing/checkout -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"amount":"25.00","credit_type":"text","currency":"usd"}' | python3 -m json.tool   # → { purchase_id, checkout_url }
curl -s $PC/v1/billing/purchases -H "Authorization: Bearer $TOKEN" | python3 -m json.tool # purchase history
```
> In dev there's no real Stripe firing the webhook, so the mint won't complete from the UI — use the
> §2.3 service-token mint to add credits. Webhook signature verification is covered by a unit test
> (`internal/domain` Stripe-sig). Monthly **budgets**: `GET/PUT /v1/billing/budget` (see
> `openapi/platform-core.yaml`).

### 2.8 Accounts / RBAC / audit — F03 📄
RBAC: privileged endpoints require an `admin` role → non-admin gets **403**; admin actions emit
`admin.action.v1` and land in the queryable **audit log**. Org create + role assignment + the audit
query live in `openapi/platform-core.yaml` (`/v1/orgs…`, `/v1/audit…`). Exercise them with the same
`Authorization: Bearer $TOKEN` (your signup tenant is `admin`).

---

## 3. CLI — F04 (the primary engineer interface) 🔁✅

```bash
make cli                                  # → bin/exascale (defaults to localhost:8001/8085/8002)
EM="cli+$(date +%s)@dev.test"
./bin/exascale signup --email "$EM" --password 'pw' --name 'CliQA'   # v0.1.5: --password flag
./bin/exascale whoami                                                # identity
# grant credits (service-token, as in §2.3) for this tenant, then:
./bin/exascale catalog                                              # models + price
./bin/exascale credits balance
./bin/exascale credits convert --from ai_index --to text --amount 100
./bin/exascale infer chat -m llama-3.1-8b "Define a GPU in one line" # reply + [N in · N out · N tokens]
./bin/exascale keys create --name production                        # secret shown once
./bin/exascale keys list
./bin/exascale config get                                           # ~/.exascale/config.json
```
**Expect:** the full money loop headless — signup → convert → infer → keys. (`gpu`/`cluster`/`train`
land with F12/F13.)

---

## 4. Web console — F20

Run the frontend two ways (same screens, zero code difference):

```bash
# Mock mode — no backend; realistic canned data. Good for UI/design review.
cd "Exascale Frontend" && npm run dev                                  # http://localhost:3000

# Local mode — wired to the live services (needs the forwards from §0).
cd "Exascale Frontend" && EXASCALE_API_MODE=local npm run dev
```

In **local mode**, sign up / log in, then walk the loop:

| Screen | What to check | Tag |
|---|---|---|
| **Login / signup** | real auth; session persists; logout clears it | v0.2.3 |
| **Wallet** | balances overlay real ledger data | v0.2.3 |
| **Wallet → Convert drawer** | live rate + house spread; submit burns/mints; toast; balance updates | v0.2.5 |
| **Wallet → Recent movements** | shows real ledger transactions (a convert appears as two legs) | v0.2.6 |
| **Inference playground** | each run shows real **credit cost**; session meter + live `text` balance | v0.2.7 |
| **Wallet → Buy credits** | Stripe checkout (mock URL in dev) | v0.2.1 |
| **Settings → API Keys** | create reveals one-time secret; list; revoke → "Revoked" | v0.2.8 |
| **/console** | one-screen ops view: catalog, run, wallet, budget, purchases, audit | v0.2.3 |

**Mock-mode tip:** every screen renders fully with canned data, so design/UX can be reviewed without
the cluster. The seam is `EXASCALE_API_MODE` only — no component changes.

---

## 5. Feature → verification checklist

| Feature | Tag(s) | How to verify | State |
|---|---|---|---|
| F01 data plane + cluster | v0.1.0 | `kubectl get pods -A`; §0 health | ✅ |
| F02 auth (signup/login/me/logout) | v0.1.1 | §2.2 / §3 | ✅ |
| F02 API keys | v0.2.0/F20-6 | §2.6 / §4 settings | ✅ |
| F02 email verify, budgets (gap-closers) | v0.1.3 | `openapi/platform-core.yaml` verify/budget | 📄 |
| F03 accounts/orgs/RBAC + audit | v0.1.2 | §2.8 (403 on non-admin; audit query) | 📄 |
| F04 exascale CLI | v0.1.4 | §3 | ✅ |
| F04 signup `--password` | v0.1.5 | `./bin/exascale signup --password …` | ✅ |
| F05 credit ledger (chain, idempotency) | v0.1.0+ | §2.3 | ✅ |
| F06 billing (Stripe → mint) | v0.2.1 | §2.7 (+ unit test for webhook sig) | 🔁📄 |
| F07 credit conversion | v0.2.4 | §2.4 / §4 convert drawer | ✅ |
| F08 inference gateway | v0.2.0 | §2.5 | ✅ |
| F08↔F05 metered debit | v0.2.0/v0.2.7 | §2.5 (balance drops ~1s after a run) | ✅ |
| F09 vLLM runtime (CPU stub) | v0.2.2 | §2.5 (real GPU needs `GPU=1` + a GPU node) | ✅ |
| F20 console + live wiring | v0.2.3–v0.2.8 | §4 | ✅ |

**Not built yet (nothing to test):** F12 compute control plane (GPU-gated), the exchange/trading
layer (paused), real observability dashboards, real Stripe/email providers (M3 provisioning).
```
