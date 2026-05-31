/**
 * POST /api/wallet/convert — convert between credit types at the published rate. Live: credit-ledger
 * /v1/credits/convert (atomic burn `from` + mint `to`, 1% house spread; is_paper comes from the
 * session JWT, never the client). Mock: computes the floored target locally so the drawer is
 * exercisable offline. The client mints the idempotency key so a retried submit is de-duped by the
 * ledger; we fall back to a server-minted UUID if absent. Amounts are fixed-point decimal strings.
 */
import { defineEventHandler, readBody, createError } from 'h3'
import { isMock, proxyJson } from '../../utils/api'

interface ConvertIn {
  from?: string
  to?: string
  amount?: string
  /** rate used only by the mock path to fake a realistic result. */
  rate?: string
  idempotency_key?: string
}

/** Mock leg shape — mirrors the ledger Transaction DTO fields the drawer reads. */
function leg(creditType: string, amount: string, balanceAfter: string, op: string) {
  return {
    tx_id: `mock_${crypto.randomUUID()}`,
    credit_type: creditType,
    operation: op,
    amount,
    balance_after: balanceAfter,
    is_paper: true,
    created_at: new Date().toISOString(),
  }
}

export default defineEventHandler(async (event) => {
  const body = await readBody<ConvertIn>(event)
  const from = (body.from || '').trim()
  const to = (body.to || '').trim()
  const amount = (body.amount || '').trim()
  if (!from || !to || from === to || !amount) {
    throw createError({ statusCode: 422, statusMessage: 'from, to (distinct) and amount are required' })
  }
  const idem = body.idempotency_key || crypto.randomUUID()

  if (isMock(event)) {
    // floor(amount × rate × (1 − 1% spread)) — keep the showcase honest about the house spread.
    const rate = Number(body.rate || '0.830579')
    const target = Math.floor(Number(amount) * rate * 0.99 * 1e6) / 1e6
    return {
      debit: leg(from, `-${Number(amount).toFixed(6)}`, '0.000000', 'conversion'),
      credit: leg(to, `${target.toFixed(6)}`, target.toFixed(6), 'conversion'),
    }
  }

  return proxyJson(event, 'ledger', '/v1/credits/convert', {
    method: 'POST',
    body: { from, to, amount },
    headers: { 'Idempotency-Key': idem },
  })
})
