/**
 * GET /api/feedback — return all tour-feedback records.
 *
 * Query params:
 *   ?persona=trader       filter by persona
 *   ?session=tour_xyz123  filter by sessionId
 *   ?limit=100            cap (default 1000, max 5000)
 *
 * Read-only mirror of server/data/feedback.json. Used by an internal
 * audit/feedback dashboard. Returns most-recent-first.
 */
// @ts-expect-error -- nitro/nuxt provides node types at runtime
import { promises as fs } from 'node:fs'
// @ts-expect-error -- nitro/nuxt provides node types at runtime
import path from 'node:path'
import { defineEventHandler, getQuery } from 'h3'

// @ts-expect-error -- nitro/nuxt provides node globals at runtime
const FILE = path.join(process.cwd(), 'server', 'data', 'feedback.json')

export default defineEventHandler(async (event) => {
  const q = getQuery(event)
  const persona = typeof q.persona === 'string' ? q.persona : null
  const session = typeof q.session === 'string' ? q.session : null
  const limitRaw = Number(q.limit ?? 1000)
  const limit = Math.max(1, Math.min(5000, Number.isFinite(limitRaw) ? limitRaw : 1000))

  let records: Array<Record<string, unknown>> = []
  try {
    const raw = await fs.readFile(FILE, 'utf8')
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) records = parsed
  } catch (err: unknown) {
    if ((err as { code?: string }).code !== 'ENOENT') throw err
  }

  if (persona) records = records.filter(r => r.persona === persona)
  if (session) records = records.filter(r => r.sessionId === session)

  records.sort((a, b) => String(b.receivedAt ?? b.ts ?? '').localeCompare(String(a.receivedAt ?? a.ts ?? '')))

  return {
    total: records.length,
    records: records.slice(0, limit),
  }
})
