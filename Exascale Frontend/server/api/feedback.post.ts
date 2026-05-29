/**
 * POST /api/feedback — append a tour-feedback record to server/data/feedback.json.
 *
 * Body shape:
 *   {
 *     sessionId:  string
 *     persona:    'trader' | 'lab' | 'datacenter' | 'enterprise'
 *     stepId:     string
 *     screen:     string   // route the user was on, e.g. '/trade'
 *     section:    string   // logical section name, e.g. 'Order form'
 *     rating:     number | null
 *     feedback:   string
 *     ts:         string   // ISO timestamp
 *     userAgent?: string
 *   }
 *
 * File is read, parsed, the new record is appended, then re-written atomically
 * (write to .tmp + rename). Concurrent writers in dev are vanishingly rare, but
 * the .tmp pattern keeps a partial write from corrupting the store.
 */
// @ts-expect-error -- nitro/nuxt provides node types at runtime
import { promises as fs } from 'node:fs'
// @ts-expect-error -- nitro/nuxt provides node types at runtime
import path from 'node:path'
import { defineEventHandler, readBody, createError } from 'h3'

interface FeedbackIn {
  sessionId: unknown
  persona:   unknown
  stepId:    unknown
  screen:    unknown
  section:   unknown
  feedback:  unknown
  ts?:       unknown
  userAgent?: unknown
}

interface FeedbackOut {
  id:        string
  sessionId: string
  persona:   string
  stepId:    string
  screen:    string
  section:   string
  feedback:  string
  ts:        string
  userAgent: string
  receivedAt: string
}

const VALID_PERSONAS = new Set(['trader', 'lab', 'datacenter', 'enterprise'])
// @ts-expect-error -- nitro/nuxt provides node globals at runtime
const FILE = path.join(process.cwd(), 'server', 'data', 'feedback.json')

function asString(v: unknown, max = 4000): string {
  if (typeof v !== 'string') return ''
  return v.slice(0, max)
}

async function readStore(): Promise<FeedbackOut[]> {
  try {
    const raw = await fs.readFile(FILE, 'utf8')
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch (err: unknown) {
    if ((err as { code?: string }).code === 'ENOENT') return []
    throw err
  }
}

async function writeStoreAtomic(records: FeedbackOut[]) {
  const tmp = FILE + '.tmp'
  await fs.mkdir(path.dirname(FILE), { recursive: true })
  await fs.writeFile(tmp, JSON.stringify(records, null, 2), 'utf8')
  await fs.rename(tmp, FILE)
}

export default defineEventHandler(async (event) => {
  const body = await readBody<FeedbackIn>(event)
  if (!body || typeof body !== 'object') {
    throw createError({ statusCode: 400, statusMessage: 'Body required' })
  }

  const persona = asString(body.persona, 32)
  if (!VALID_PERSONAS.has(persona)) {
    throw createError({ statusCode: 400, statusMessage: 'Invalid persona' })
  }
  const sessionId = asString(body.sessionId, 64)
  const stepId    = asString(body.stepId,    128)
  const screen    = asString(body.screen,    256)
  const section   = asString(body.section,   128)
  const feedback  = asString(body.feedback,  4000)

  if (!sessionId || !stepId || !screen || !section) {
    throw createError({ statusCode: 400, statusMessage: 'sessionId, stepId, screen, section all required' })
  }
  if (!feedback.trim()) {
    throw createError({ statusCode: 400, statusMessage: 'Feedback text required' })
  }

  const record: FeedbackOut = {
    id:         'fb_' + Math.random().toString(36).slice(2, 10) + Date.now().toString(36),
    sessionId,
    persona,
    stepId,
    screen,
    section,
    feedback,
    ts:         asString(body.ts, 64) || new Date().toISOString(),
    userAgent:  asString(body.userAgent, 512),
    receivedAt: new Date().toISOString(),
  }

  const records = await readStore()
  records.push(record)
  await writeStoreAtomic(records)

  return { ok: true, id: record.id, total: records.length }
})
