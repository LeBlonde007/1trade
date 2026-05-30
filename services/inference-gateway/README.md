# inference-gateway (F08 — the Exascale inference API)

The customer-facing inference API: OpenAI-wire-compatible chat/completions, embeddings, audio, and
images. Owner: `inference-ml`. Contract: `docs/contracts/openapi/inference.yaml`. **The fastest
revenue path** — it turns auth (F02) + the credit ledger (F05) into a billable product.

## Request flow (target)
```
Customer (Authorization: Bearer exk_… | tenant JWT)
  → resolve key → tenant_id / sub_account_id / scopes / is_paper   (platform-core introspection)
  → pre-flight: enough of the target sub-credit?  (credit-ledger balances)  → 402 if short
  → serve via the model backend (mock now; real vLLM behind the same interface in F09)
  → on completion: emit inference.usage.v1 exactly once (request_id = debit idempotency key)
  → credit-ledger consumes the event → atomic, idempotent debit  ("debit visible in wallet")
```
`is_paper` is derived from the authenticated tenant, never from the request body.

## Status (F08, in progress)
- ✅ **Contract** authored: `openapi/inference.yaml` v1.0.0 (validates).
- ✅ **Scaffold**: service skeleton — config, `/healthz`, `/readyz`, `GET /v1/models` from the
  curated catalog (Llama-70B/8B + Whisper). Builds, vets, runs.
- ⬜ Auth (API-key/JWT → principal), credit pre-flight (402), inference handlers (mock backend) +
  SSE, usage metering → `inference.usage.v1`, the ledger consumer (idempotent debit), tests, deploy.

## Invariants
- Never calls compute-platform directly (sync table §6: pod scheduling flows the other way).
- Money/units are fixed-point decimal strings — never floats.
- One `inference.usage.v1` per request; the ledger debit is idempotent on `request_id`.

## Dev
```bash
cd services/inference-gateway && go test ./...
INFERENCE_ADDR=:8085 go run ./cmd/inference-gateway   # then: curl localhost:8085/v1/models
```
