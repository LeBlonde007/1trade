/**
 * Formatting utilities for the Exascale UI.
 * All number formatting goes through these — never use toFixed/toLocaleString directly.
 * Auto-imported by Nuxt — call as fmt money(), formatCredits(), etc.
 */

/** Format a USD amount, e.g. 10247.83 → "$10,247.83". */
export const formatUSD = (value: number, opts: { showSign?: boolean; compact?: boolean } = {}): string => {
  if (Number.isNaN(value)) return '—'
  const formatter = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: opts.compact && Math.abs(value) >= 1000 ? 0 : 2,
    maximumFractionDigits: 2,
    notation: opts.compact ? 'compact' : 'standard',
  })
  const sign = opts.showSign && value > 0 ? '+' : ''
  return `${sign}${formatter.format(value)}`
}

/** Format a credit count, e.g. 2425000 → "2,425,000". */
export const formatCredits = (value: number): string => {
  if (Number.isNaN(value)) return '—'
  return new Intl.NumberFormat('en-US').format(Math.round(value))
}

/** Format a sub-credit price like 0.001005 → "$0.001005" (six decimals for AI credits). */
export const formatPrice = (value: number, decimals = 6): string => {
  if (Number.isNaN(value)) return '—'
  return `$${value.toFixed(decimals)}`
}

/** Format a percentage delta, e.g. 0.0235 → "+2.35%". */
export const formatPct = (value: number, opts: { showSign?: boolean; decimals?: number } = {}): string => {
  if (Number.isNaN(value)) return '—'
  const decimals = opts.decimals ?? 2
  const sign = opts.showSign !== false && value > 0 ? '+' : ''
  return `${sign}${(value * 100).toFixed(decimals)}%`
}

/** Direction arrow for a numeric delta. ▲ for positive, ▼ for negative, • for zero. */
export const directionGlyph = (value: number): '▲' | '▼' | '•' => {
  if (value > 0) return '▲'
  if (value < 0) return '▼'
  return '•'
}

/** Semantic color token name for a delta. */
export const deltaTokenColor = (value: number): 'var(--pos)' | 'var(--neg)' | 'var(--text-2)' => {
  if (value > 0) return 'var(--pos)'
  if (value < 0) return 'var(--neg)'
  return 'var(--text-2)'
}

/** Compact USD for hero stats: 200000000 → "$200M". */
export const formatBigUSD = (value: number): string => {
  if (Number.isNaN(value)) return '—'
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value)
}

/** Format a timestamp HH:MM:SS.mmm for the trade tape. */
export const formatTapeTime = (date: Date | number): string => {
  const d = typeof date === 'number' ? new Date(date) : date
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  const ms = String(d.getMilliseconds()).padStart(3, '0')
  return `${hh}:${mm}:${ss}.${ms}`
}

/** Format a date like "2026-05-20" (ISO short). */
export const formatDateISO = (date: Date | number): string => {
  const d = typeof date === 'number' ? new Date(date) : date
  return d.toISOString().slice(0, 10)
}

/** Compact a hex string for display, e.g. "ord_abc1234567890" → "ord_abc...890". */
export const truncateId = (id: string, head = 7, tail = 3): string => {
  if (id.length <= head + tail + 3) return id
  return `${id.slice(0, head)}…${id.slice(-tail)}`
}
