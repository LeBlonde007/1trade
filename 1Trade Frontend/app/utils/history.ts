/**
 * history — merges the two live sources behind /history into one time-ordered event list.
 *
 *   - Paper fills from the matching engine (`/api/trading/fills`) — real paper trades that settle in the
 *     ledger (real money is paused, F22); every row carries is_paper and is labelled as such on screen.
 *   - Ledger transactions from credit-ledger (`/api/wallet/transactions`). These are real, append-only
 *     and hash-chained; each carries its own chain_hash.
 *
 * Amounts stay fixed-point strings end to end. Totals are summed in integer micro-units (BigInt) so the
 * summary row never picks up float drift — the same NUMERIC(20,6) scale the ledger uses.
 */

/** A paper fill as served by matching-engine /v1/trading/fills. */
export interface EngineFill {
  fill_id: string
  order_id: string
  product_id: string
  side: 'buy' | 'sell'
  price: string
  quantity: string
  notional: string
  fee: string
  liquidity: 'maker' | 'taker'
  executed_at: string
  is_paper: boolean
}

/** A ledger transaction as served by credit-ledger /v1/credits/transactions. */
export interface LedgerTx {
  tx_id: string
  credit_type: string
  operation: string
  amount: string
  balance_after: string
  reference_id?: string | null
  is_paper: boolean
  created_at: string
  chain_hash?: string
}

/** Row kinds shown in the type filter. Ledger operations collapse onto these. */
export type HistoryKind = 'Trade' | 'Purchase' | 'Conversion' | 'Usage' | 'Adjustment' | 'Cash'

/** One row of the history table — either a paper fill or a ledger entry. */
export interface HistoryRow {
  id: string
  source: 'engine' | 'ledger' | 'cash'
  kind: HistoryKind
  ts: string
  market: string
  side: 'buy' | 'sell' | null
  quantity: string
  price: string | null
  total: string | null
  fee: string | null
  isPaper: boolean
  // engine-only
  orderId?: string
  liquidity?: 'maker' | 'taker'
  // ledger-only
  operation?: string
  balanceAfter?: string
  referenceId?: string | null
  chainHash?: string
}

/** ledgerKind maps a ledger operation (credit.yaml Transaction.operation) onto a table kind. */
export function ledgerKind(op: string): HistoryKind {
  switch (op) {
    case 'purchase': return 'Purchase'
    case 'conversion': return 'Conversion'
    case 'consumption': return 'Usage'
    case 'trade': return 'Trade'
    default: return 'Adjustment' // mint · burn · refund
  }
}

/** fromFill turns an engine fill into a history row. */
export function fromFill(f: EngineFill): HistoryRow {
  return {
    id: f.fill_id,
    source: 'engine',
    kind: 'Trade',
    ts: f.executed_at,
    market: f.product_id,
    side: f.side,
    quantity: f.quantity,
    price: f.price,
    total: f.notional,
    fee: f.fee,
    isPaper: f.is_paper,
    orderId: f.order_id,
    liquidity: f.liquidity,
  }
}

/** fromLedger turns a ledger transaction into a history row. Ledger rows carry no price or fee. */
export function fromLedger(t: LedgerTx): HistoryRow {
  return {
    id: t.tx_id,
    source: 'ledger',
    kind: ledgerKind(t.operation),
    ts: t.created_at,
    market: t.credit_type,
    side: null,
    quantity: t.amount,
    price: null,
    total: null,
    fee: null,
    isPaper: t.is_paper,
    operation: t.operation,
    balanceAfter: t.balance_after,
    referenceId: t.reference_id ?? null,
    chainHash: t.chain_hash,
  }
}

/** A paper cash movement as served by credit-ledger /v1/credits/cash/transactions (credit.yaml v1.1). */
export interface LedgerCashTx {
  tx_id: string
  currency: string
  operation: string // paper_grant | trade | fee
  amount: string
  balance_after: string
  reference_id?: string | null
  is_paper: boolean
  created_at: string
  chain_hash?: string
}

/** fromCash turns a paper cash movement into a history row. Cash rows are always paper. */
export function fromCash(t: LedgerCashTx): HistoryRow {
  return {
    id: t.tx_id,
    source: 'cash',
    kind: 'Cash',
    ts: t.created_at,
    market: t.currency,
    side: null,
    quantity: t.amount,
    price: null,
    total: null,
    fee: null,
    isPaper: t.is_paper,
    operation: t.operation,
    balanceAfter: t.balance_after,
    referenceId: t.reference_id ?? null,
    chainHash: t.chain_hash,
  }
}

/**
 * mergeHistory combines both sources, newest first. Timestamps are compared as instants, not strings —
 * the two services may render RFC 3339 differently (offset vs Z, fractional seconds). Ties break on id so
 * the order is stable.
 */
export function mergeHistory(fills: EngineFill[], txs: LedgerTx[], cash: LedgerCashTx[] = []): HistoryRow[] {
  const rows = [...fills.map(fromFill), ...txs.map(fromLedger), ...cash.map(fromCash)]
  const at = (r: HistoryRow) => Date.parse(r.ts) || 0
  return rows.sort((a, b) => at(b) - at(a) || a.id.localeCompare(b.id))
}

const SCALE = 6
const UNIT = 10n ** BigInt(SCALE)

/** toMicros parses a fixed-point decimal string into integer micro-units; malformed input is 0. */
export function toMicros(s: string | null | undefined): bigint {
  const m = /^(-)?(\d+)(?:\.(\d+))?$/.exec((s ?? '').trim())
  if (!m) return 0n
  const frac = (m[3] ?? '').slice(0, SCALE).padEnd(SCALE, '0')
  const v = BigInt(m[2]!) * UNIT + BigInt(frac)
  return m[1] ? -v : v
}

/** fromMicros renders integer micro-units back to a 6-dp fixed-point string. */
export function fromMicros(v: bigint): string {
  const neg = v < 0n
  const abs = neg ? -v : v
  return `${neg ? '-' : ''}${abs / UNIT}.${(abs % UNIT).toString().padStart(SCALE, '0')}`
}

/** sumDecimal adds fixed-point strings exactly (nulls count as zero). */
export function sumDecimal(values: Array<string | null | undefined>): string {
  return fromMicros(values.reduce<bigint>((acc, v) => acc + toMicros(v), 0n))
}

/** CSV_COLUMNS is the export column order — every field a reviewer needs to reconcile a row. */
const CSV_COLUMNS: Array<keyof HistoryRow> = [
  'ts', 'kind', 'source', 'market', 'side', 'quantity', 'price', 'total', 'fee', 'isPaper',
  'orderId', 'liquidity', 'operation', 'balanceAfter', 'referenceId', 'chainHash', 'id',
]

/** csvCell quotes a value per RFC 4180 and neutralises spreadsheet formula injection. */
function csvCell(v: unknown): string {
  let s = v === null || v === undefined ? '' : String(v)
  if (/^[=+\-@\t\r]/.test(s) && !/^-?\d/.test(s)) s = `'${s}`
  return /[",\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

/** toCSV serialises rows with a header line, in CSV_COLUMNS order. */
export function toCSV(rows: HistoryRow[]): string {
  const lines = [CSV_COLUMNS.join(',')]
  for (const r of rows) lines.push(CSV_COLUMNS.map((c) => csvCell(r[c])).join(','))
  return lines.join('\r\n') + '\r\n'
}
