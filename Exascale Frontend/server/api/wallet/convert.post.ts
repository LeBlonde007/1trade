/**
 * POST /api/wallet/convert — convert between credit types at the published rate. Proxies credit-ledger
 * /v1/credits/convert (atomic burn `from` + mint `to`, 1% house spread; is_paper comes from the
 * session JWT, never the client). The client mints the idempotency key so a retried submit is de-duped
 * by the ledger; we fall back to a server-minted UUID if absent. Amounts are fixed-point decimal strings.
 */
import { defineEventHandler, readBody, createError } from 'h3'
import { proxyJson } from '../../utils/api'

interface ConvertIn {
  from?: string
  to?: string
  amount?: string
  idempotency_key?: string
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
  return proxyJson(event, 'ledger', '/v1/credits/convert', {
    method: 'POST',
    body: { from, to, amount },
    headers: { 'Idempotency-Key': idem },
  })
})
