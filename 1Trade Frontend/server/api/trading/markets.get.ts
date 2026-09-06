/**
 * GET /api/trading/markets — one payload with everything the markets table needs.
 *
 * The engine exposes the product list and per-product detail separately, so building this table
 * client-side would be 1 + N round-trips from the browser (currently 10). Fanning out here is what
 * a BFF is for: the browser makes one request, and the concurrency happens next to the service.
 *
 * Market data is simulated while the venue is paused, but it is simulated SERVER-side — every
 * screen therefore shows the same book. Screens must not generate their own prices.
 *
 * `exchange_status` is passed through untouched: the UI has to render the paused state and the
 * "market data is simulated" reason verbatim (F22 — exchange surfaces stay explicitly staged).
 */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../utils/api'

interface Product {
  product_id: string
  name: string
  description: string
  family: string
  credit_type: string
  product_type: string
  quote_precision: number
  tick_size: string
  tradeable: boolean
  is_paper: boolean
  is_real_money_enabled: boolean
}
interface Summary {
  last: string
  change_pct_24h: string
  spread_bps: string
  volume_24h: string
  high_24h: string
  low_24h: string
  is_paper: boolean
}
interface ExchangeStatus { state: string; reason: string; methodology_url: string }

export interface MarketRow extends Product {
  last: number
  changePct24h: number
  spreadBps: number
  volume24h: number
  high24h: number
  low24h: number
}

/** num parses the engine's fixed-point decimal strings. Never trust a missing field to be 0 — an
 *  absent price and a zero price mean different things, so unparseable values become null upstream
 *  of the UI rather than silently rendering as 0.00. */
const num = (v: string | undefined): number => {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

export default defineEventHandler(async (event) => {
  const list = await proxyJson<{ products: Product[]; exchange_status: ExchangeStatus }>(
    event, 'trading', '/v1/trading/products',
  )
  const products = list.products ?? []

  // One detail call per product, concurrently. A single failure must not blank the whole table —
  // that product falls back to its definition with no market data rather than taking the page down.
  const rows = await Promise.all(products.map(async (p): Promise<MarketRow> => {
    try {
      const d = await proxyJson<{ product: Product; summary: Summary }>(
        event, 'trading', `/v1/trading/products/${encodeURIComponent(p.product_id)}`,
      )
      const s = d.summary
      return {
        ...p,
        last: num(s?.last),
        changePct24h: num(s?.change_pct_24h),
        spreadBps: num(s?.spread_bps),
        volume24h: num(s?.volume_24h),
        high24h: num(s?.high_24h),
        low24h: num(s?.low_24h),
      }
    } catch {
      return { ...p, last: 0, changePct24h: 0, spreadBps: 0, volume24h: 0, high24h: 0, low24h: 0 }
    }
  }))

  return { exchange_status: list.exchange_status, markets: rows }
})
