#!/usr/bin/env bash
# Backup/restore drill (F01) — prove the credit ledger survives a Postgres dump + restore intact,
# *including the append-only hash chain* (the financial heart; tampering would change the digest).
#
# It: (1) pg_dumps the `exascale` DB, (2) restores it into a scratch DB, (3) asserts every table's
# row count matches the live DB, and (4) asserts the ledger's chain-hash digest is byte-identical.
# Postgres-only — needs just the data plane (no app services). Idempotent; safe to re-run.
#
#   scripts/backup-restore-drill.sh          # run the full drill against the running cluster
#   KEEP=1 scripts/backup-restore-drill.sh   # keep the scratch DB + dump for inspection
#
# Env: PG_NAMESPACE(data) PG_DB(exascale) RESTORE_DB(exascale_restore) PG_USER(exascale) OUT_DIR(backups)
set -euo pipefail

NS="${PG_NAMESPACE:-data}"
DB="${PG_DB:-exascale}"
RESTORE_DB="${RESTORE_DB:-exascale_restore}"
PGUSER="${PG_USER:-exascale}"
OUT_DIR="${OUT_DIR:-backups}"
DUMP="$OUT_DIR/${DB}-$(date +%Y%m%d-%H%M%S).sql"

say()  { printf '\n\033[1m== %s ==\033[0m\n' "$*"; }
ok()   { printf '\033[32m✓ %s\033[0m\n' "$*"; }
fail() { printf '\033[31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

command -v kubectl >/dev/null || fail "kubectl not found"
POD="$(kubectl get pod -n "$NS" -l app=postgres -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
[ -n "$POD" ] || fail "no postgres pod in namespace '$NS' — is the data plane up? (make data-plane)"
ok "postgres pod: $NS/$POD"

# psql_val runs one SQL statement against $1 and returns the raw scalar (-tA: tuples-only, unaligned).
psql_val() { kubectl exec -n "$NS" -i "$POD" -- psql -U "$PGUSER" -d "$1" -tAc "$2"; }
# exact_counts emits "table:rowcount" for every public table in DB $1 (exact count, not the estimate).
exact_counts() {
  local db="$1" t c
  for t in $(psql_val "$db" "SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename"); do
    c="$(psql_val "$db" "SELECT count(*) FROM \"$t\"")"
    printf '%s:%s\n' "$t" "$c"
  done
}

# --- 1. Backup ---------------------------------------------------------------
say "1. Backup — pg_dump $DB"
mkdir -p "$OUT_DIR"
kubectl exec -n "$NS" "$POD" -- pg_dump -U "$PGUSER" --no-owner --no-privileges "$DB" > "$DUMP"
SIZE="$(wc -c < "$DUMP")"
[ "$SIZE" -gt 0 ] || fail "dump is empty"
ok "dumped $SIZE bytes → $DUMP"

# --- 2. Fingerprint the live ledger -----------------------------------------
say "2. Fingerprint the live ledger"
TX_BEFORE="$(psql_val "$DB" "SELECT count(*) FROM credit_transactions")"
CHAIN_BEFORE="$(psql_val "$DB" "SELECT coalesce(md5(string_agg(chain_hash, ',' ORDER BY tx_id)),'empty') FROM credit_transactions")"
echo "  credit_transactions: $TX_BEFORE rows"
echo "  chain-hash digest:   $CHAIN_BEFORE"

# --- 3. Restore into a scratch DB -------------------------------------------
say "3. Restore into scratch DB $RESTORE_DB"
psql_val postgres "DROP DATABASE IF EXISTS $RESTORE_DB" >/dev/null
psql_val postgres "CREATE DATABASE $RESTORE_DB" >/dev/null
kubectl exec -n "$NS" -i "$POD" -- psql -U "$PGUSER" -d "$RESTORE_DB" -v ON_ERROR_STOP=1 -q < "$DUMP" >/dev/null \
  || fail "restore failed (psql ON_ERROR_STOP)"
ok "restored from $DUMP"

# --- 4. Verify the restored ledger ------------------------------------------
say "4. Verify the restored ledger"
TX_AFTER="$(psql_val "$RESTORE_DB" "SELECT count(*) FROM credit_transactions")"
CHAIN_AFTER="$(psql_val "$RESTORE_DB" "SELECT coalesce(md5(string_agg(chain_hash, ',' ORDER BY tx_id)),'empty') FROM credit_transactions")"
echo "  credit_transactions: $TX_AFTER rows"
echo "  chain-hash digest:   $CHAIN_AFTER"

# --- 5. Compare every table's row count -------------------------------------
say "5. Compare every table's row count (live vs restored)"
if diff <(exact_counts "$DB") <(exact_counts "$RESTORE_DB"); then ok "all table counts match"; else fail "row counts differ between live and restored"; fi

# --- 6. Verdict --------------------------------------------------------------
say "Verdict"
[ "$TX_BEFORE" = "$TX_AFTER" ]       || fail "transaction count drifted: $TX_BEFORE → $TX_AFTER"
[ "$CHAIN_BEFORE" = "$CHAIN_AFTER" ] || fail "LEDGER HASH-CHAIN digest changed — restore is NOT byte-identical"
ok "Backup/restore drill PASSED — ledger hash chain intact ($TX_AFTER txns · digest $CHAIN_AFTER)"

# --- cleanup -----------------------------------------------------------------
if [ "${KEEP:-0}" != "1" ]; then
  psql_val postgres "DROP DATABASE IF EXISTS $RESTORE_DB" >/dev/null && echo "  (scratch DB dropped — KEEP=1 to retain it + the dump)"
fi
