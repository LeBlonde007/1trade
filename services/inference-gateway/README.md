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
- ✅ **Complete & deployed in k3d (v0.2.0).** Contract `openapi/inference.yaml` v1.0.0; catalog
  (`GET /v1/models`); auth (API-key → platform-core introspection w/ 30s cache, or first-party JWT);
  credit **pre-flight 402**; `POST /v1/chat/completions` (mock backend, JSON + SSE) with fixed-point
  metering → `inference.usage.v1`; credit-ledger consumes it → idempotent debit.
- **Proven live:** signup → API key → `curl /v1/chat/completions` → text balance `100.000000 →
  99.865000` (debited exactly `5 × 27/1000`); zero-credit tenant → 402.
- ⬜ Next: real vLLM behind `model.Backend` (F09); completions/embeddings/audio/image endpoints
  (same backend+pricing+meter path); token-cost pre-authorization; asymmetric JWT (see below).

## Security follow-ups (logged, non-blocking for sandbox M2)
- **Asymmetric JWT (RS256/EdDSA):** today every service shares the HS256 secret, so any service can
  *mint* tokens (the gateway does, for the pre-flight). Move platform-core to a private signing key +
  public verification keys so services verify but can't mint.
- **Pre-flight fail-open:** a ledger outage lets a request serve uncredited; pair with a
  reservation/hold model so usage can't exceed balance during an outage.

## Invariants
- Never calls compute-platform directly (sync table §6: pod scheduling flows the other way).
- Money/units are fixed-point decimal strings — never floats.
- One `inference.usage.v1` per request; the ledger debit is idempotent on `request_id`.

## Dev
```bash
cd services/inference-gateway && go test ./...
INFERENCE_ADDR=:8085 go run ./cmd/inference-gateway   # then: curl localhost:8085/v1/models
```
