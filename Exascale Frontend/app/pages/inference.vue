<script setup lang="ts">
/**
 * /inference — Inference Playground + Model Catalog.
 *
 * Dark, inside `app` layout. Split-screen: left ~320px model catalog,
 * right ~1fr playground (chat or code view) with a narrow metadata
 * sidebar inside the playground area.
 */

import { Paperclip, Mic, Volume2, Headphones, Monitor, ChevronDown } from 'lucide-vue-next'
import type { CatalogModel } from '~/composables/useCatalog'
import type { ChatUsage } from '~/composables/useInference'

definePageMeta({ layout: 'app' })
useHead({ title: 'Inference Playground — Exascale' })

// =====================================================
// Model catalog
// =====================================================
type Category = 'text' | 'speech' | 'image' | 'video' | 'embed' | 'vision'
interface ModelDef {
  id: string
  name: string
  version?: string
  provider: string
  category: Category
  // For text: per-1M tokens in/out USD. For non-text: free-form.
  priceIn?: number
  priceOut?: number
  priceLabel?: string     // e.g. "$0.006 / min" for speech
  context?: string
  speed?: string
  sub?: string
}

// ── Live catalog (no mock) ──────────────────────────────────────────────────────────────────────
// The model list is the gateway's real catalog (GET /v1/models via the BFF). The live catalog carries
// id · owned_by · modality · credit_type · unit · price (in credits); display names are derived.
const catalog = useCatalog()

/** humanizeId turns a model id (llama-3.1-8b) into a display name (Llama 3.1 8B). */
function humanizeId(id: string): string {
  return id.split('-')
    .map((p) => /^[0-9]/.test(p) || /^v[0-9]/i.test(p) ? p.toUpperCase() : p.charAt(0).toUpperCase() + p.slice(1))
    .join(' ')
}
/** toCategory maps an Exascale modality to a catalog UI category. */
function toCategory(modality: string): Category {
  if (modality === 'embeddings' || modality === 'embed') return 'embed'
  if (modality === 'speech' || modality === 'image' || modality === 'video' || modality === 'vision') return modality
  return 'text'
}
/**
 * providerLabel frames the model as Exascale's own infrastructure. Exascale-owned catalog entries are
 * served on Exascale GPUs (the upstream that physically runs them is an implementation detail the
 * customer never sees) → "Self-hosted". Anything else shows its owner.
 */
function providerLabel(ownedBy: string): string {
  return !ownedBy || ownedBy.toLowerCase() === 'exascale' ? 'Self-hosted' : ownedBy
}
/** toModelDef maps a live CatalogModel into the card display shape — real price, in credits. */
function toModelDef(m: CatalogModel): ModelDef {
  const x = m.exascale
  return {
    id: m.id,
    name: m.name || humanizeId(m.id),
    provider: providerLabel(m.owned_by),
    category: toCategory(x.modality),
    priceLabel: `${Number(x.price).toLocaleString('en-US')} ${x.credit_type} / ${x.unit}`,
    sub: `${x.unit} · billed in ${x.credit_type} credits`,
  }
}
/** models — the live catalog mapped to cards (empty until loaded; never mock). */
const models = computed<ModelDef[]>(() => catalog.models.value.map(toModelDef))

const CATEGORY_META: Record<Category, { label: string; cls: string }> = {
  text:   { label: 'TEXT',     cls: 'cat-text' },
  speech: { label: 'SPEECH',   cls: 'cat-speech' },
  image:  { label: 'IMAGE',    cls: 'cat-image' },
  video:  { label: 'VIDEO',    cls: 'cat-video' },
  embed:  { label: 'EMBED',    cls: 'cat-embed' },
  vision: { label: 'VISION',   cls: 'cat-speech' },
}

// =====================================================
// Catalog state — search + filter
// =====================================================
const catalogQuery = ref('')
type Filter = 'all' | Category
const FILTERS: Filter[] = ['all', 'text', 'vision', 'speech', 'image', 'video', 'embed']
const filter = ref<Filter>('all')

const filteredModels = computed(() => {
  const q = catalogQuery.value.trim().toLowerCase()
  return models.value.filter(m => {
    if (filter.value !== 'all' && m.category !== filter.value) return false
    if (!q) return true
    return (m.name + ' ' + m.provider + ' ' + (m.sub ?? '')).toLowerCase().includes(q)
  })
})

// Empty placeholder while the live catalog loads — not mock data, just a safe non-null default.
const EMPTY_MODEL: ModelDef = { id: '', name: '—', provider: '', category: 'text' }
const selectedId = ref('')
const selected = computed(() => models.value.find(m => m.id === selectedId.value) ?? models.value[0] ?? EMPTY_MODEL)

// turnModelName resolves the display name for the model that produced a given turn, so each reply keeps
// its own model label even after the picker changes (falls back to the live selection for older turns).
function turnModelName(t: Turn): string {
  const id = t.model || t.backend
  if (!id) return selected.value.name
  return models.value.find(m => m.id === id)?.name ?? humanizeId(id)
}

// ── Runnable code blocks (Codex-style) — split a finished assistant message into prose + fenced code so
//    each code block renders with a ▶ Run button (CodeRunner). Only for settled turns (not mid-stream).
interface MsgSegment { type: 'md' | 'code'; content: string; lang?: string }
const FENCE_RE = /```([\w+-]*)\n?([\s\S]*?)```/g
/** hasCodeBlock reports whether a finished turn contains a fenced code block worth a runnable view. */
function hasCodeBlock(t: Turn): boolean {
  return !t.streaming && !t.html && typeof t.text === 'string' && t.text.includes('```')
}
/** messageSegments splits the text into ordered prose (md) and code segments for segmented rendering. */
function messageSegments(text: string): MsgSegment[] {
  const segs: MsgSegment[] = []
  let last = 0
  FENCE_RE.lastIndex = 0
  for (let m = FENCE_RE.exec(text); m; m = FENCE_RE.exec(text)) {
    if (m.index > last) segs.push({ type: 'md', content: text.slice(last, m.index) })
    segs.push({ type: 'code', content: (m[2] ?? '').replace(/\n$/, ''), lang: (m[1] || 'text').toLowerCase() })
    last = m.index + m[0].length
  }
  if (last < text.length) segs.push({ type: 'md', content: text.slice(last) })
  return segs.length ? segs : [{ type: 'md', content: text }]
}

// Real signed-in identity for the chat transcript (no hardcoded demo user). We only have the email,
// so the label is its local-part and the avatar is up to two initials derived from it.
const { user } = useAuth()
const userLabel = computed(() => (user.value?.email?.split('@')[0]) || 'You')
const userInitials = computed(() => {
  const base = (user.value?.email?.split('@')[0]) || 'you'
  const parts = base.split(/[._-]/).filter(Boolean)
  return ((parts[0]?.[0] || base[0] || 'Y') + (parts[1]?.[0] || '')).toUpperCase()
})

// On mobile the catalog is a collapsible picker; choosing a model closes it so the chat is in view.
const catalogOpen = ref(false)
function selectModel(id: string) {
  selectedId.value = id
  catalogOpen.value = false
}

// =====================================================
// Playground parameters
// =====================================================
const params = reactive({
  temperature: 0.7,
  maxTokens: 1000,
  topP: 0.95,
  freqPenalty: 0,
  systemPrompt: 'You are a precise technical assistant for the Exascale venue. Cite specific subsystems and code where relevant. Default to bullet-point answers under 200 words unless asked otherwise.',
})

const paramsOpen = ref(false)

// =====================================================
// Conversation
// =====================================================
type Role = 'user' | 'assistant'
interface Turn {
  id: number
  role: Role
  /** Plain text for code views; HTML for rendered markdown */
  text: string
  html?: string
  tokens?: { in?: number; out?: number }
  latencyMs?: number
  /** Real credit cost: tokens × catalog price, in `creditType` credits. */
  creditCost?: number
  creditType?: string
  reqId?: string
  backend?: string
  /** Catalog id of the model that produced this turn (so the header label stays correct after the
   *  picker changes — each reply keeps the model it was generated with). */
  model?: string
  /** True while this assistant turn is still receiving streamed tokens (drives the live indicator). */
  streaming?: boolean
  /** Generated images (data URIs or URLs) for an image-model turn — rendered as a grid. */
  images?: string[]
  /** Generated video URL (the BFF content route) for a video-model turn — rendered in a player. */
  video?: string
  /** Generated audio (object URL) for a speech-model turn — rendered in an <audio> player. */
  audio?: string
  /** Names of files attached to this user turn (their content is sent as context, not shown inline). */
  files?: string[]
}

// Live-only: the conversation starts empty — real turns are appended as the user runs inference.
const turns = reactive<Turn[]>([])

// Most recent assistant for the metadata sidebar
const lastAssistant = computed(() => {
  for (let i = turns.length - 1; i >= 0; i--) {
    if (turns[i]!.role === 'assistant') return turns[i]!
  }
  return null
})

// ── Session persistence (so a refresh doesn't wipe the chat — sandbox + local) ────────────────────
// The transcript, params, and selected model are mirrored to localStorage, namespaced per user so a
// shared browser never leaks one tenant's chat into another's. This is UI state only (no credits or
// orders — the real spend already lives in the ledger), so localStorage is the right, client-side home;
// it works the same on the deployed sandbox. Hydration happens in onMounted (post-SSR) to avoid a
// hydration mismatch, matching the useGuidedTour idiom.
const CHAT_STORE_VERSION = 'v1'
const chatStoreKey = computed(() => `exa:inference:chat:${CHAT_STORE_VERSION}:${user.value?.user_id ?? 'anon'}`)
interface PersistedChat { turns?: Turn[]; params?: Partial<typeof params>; selectedId?: string; convId?: string | null }

/** loadChat reads the saved session for the current user (client-only; safe on SSR + malformed JSON). */
function loadChat(): PersistedChat | null {
  if (typeof localStorage === 'undefined') return null
  try {
    const raw = localStorage.getItem(chatStoreKey.value)
    return raw ? (JSON.parse(raw) as PersistedChat) : null
  } catch { return null }
}
// Durable, cross-device history lives server-side (platform-core conversations); localStorage is just
// the instant per-browser restore. currentConvId is the open conversation (null = a fresh, unsaved chat).
const convs = useConversations()
const convList = convs.list // top-level ref so the template auto-unwraps it
const currentConvId = ref<string | null>(null)
const historyOpen = ref(false)
let serverSaveTimer: ReturnType<typeof setTimeout> | null = null

/** serializeTranscript is the persisted form of the transcript: in-flight turns dropped, volatile fields
 *  (html/streaming) removed, and media kept only when it's a durable URL — base64/blob media is uploaded
 *  to object storage (Spaces) separately and swapped in, so neither store carries fat inline bytes. */
function serializeTranscript(): Turn[] {
  return turns
    .filter((t) => !t.streaming)
    .slice(-200)
    .map((t) => {
      const o: Turn = { id: t.id, role: t.role, text: t.text }
      if (t.model) o.model = t.model
      if (t.backend) o.backend = t.backend
      if (t.files?.length) o.files = t.files
      if (t.creditCost) { o.creditCost = t.creditCost; o.creditType = t.creditType }
      if (t.latencyMs) o.latencyMs = t.latencyMs
      if (t.tokens) o.tokens = t.tokens
      const imgs = (t.images ?? []).filter((u) => /^https?:\/\//.test(u))
      if (imgs.length) o.images = imgs
      if (t.video) o.video = t.video
      if (t.audio && /^https?:\/\//.test(t.audio)) o.audio = t.audio
      return o
    })
}
/** saveChat mirrors the live session to localStorage (instant per-browser restore on refresh). */
function saveChat() {
  if (typeof localStorage === 'undefined') return
  try {
    const payload: PersistedChat = { turns: serializeTranscript(), params, selectedId: selectedId.value, convId: currentConvId.value }
    localStorage.setItem(chatStoreKey.value, JSON.stringify(payload))
  } catch { /* quota / serialization — keep the in-memory session */ }
}
/** convTitle derives a short human title from the first user message. */
function convTitle(): string {
  const first = turns.find((t) => t.role === 'user' && t.text?.trim())
  const s = (first?.text ?? '').replace(/\s+/g, ' ').trim()
  return s ? (s.length > 48 ? s.slice(0, 48) + '…' : s) : 'New chat'
}
/** scheduleServerSave debounces persisting the conversation server-side after edits settle. */
function scheduleServerSave() {
  if (serverSaveTimer) clearTimeout(serverSaveTimer)
  serverSaveTimer = setTimeout(() => { void persistConversation() }, 1500)
}
/** persistConversation creates or updates the server-side conversation from the current transcript. */
async function persistConversation() {
  const transcript = serializeTranscript()
  if (!transcript.length) return
  const title = convTitle()
  if (!currentConvId.value) {
    const c = await convs.create(title, selectedId.value, transcript)
    if (c) { currentConvId.value = c.id; saveChat(); await convs.refresh() }
  } else {
    await convs.save(currentConvId.value, title, selectedId.value, transcript)
    const it = convs.list.value.find((x) => x.id === currentConvId.value)
    if (it) { it.title = title; it.model = selectedId.value; it.updated_at = new Date().toISOString() }
  }
}
/** openConversation loads a saved conversation into the playground. */
async function openConversation(id: string) {
  if (id === currentConvId.value) return
  const c = await convs.load(id)
  if (!c) return
  turns.splice(0, turns.length, ...((c.transcript as Turn[]) ?? []))
  currentConvId.value = c.id
  if (c.model && models.value.some((m) => m.id === c.model)) selectedId.value = c.model
  saveChat()
  await nextTick(); scrollToBottom()
}
/** deleteConversation removes a saved conversation; resets to a new chat if it was the open one. */
async function deleteConversation(id: string) {
  await convs.remove(id)
  if (id === currentConvId.value) clearChat()
}
/** uploadMedia stores a generated media blob in object storage (Spaces) and returns its public URL, so
 *  the transcript references a durable URL rather than fat base64 / a transient blob. Best-effort. */
async function uploadMedia(blob: Blob, filename: string): Promise<string | null> {
  try {
    const sig = await $fetch<{ upload_url: string; file_url: string; headers?: Record<string, string> }>('/api/files/presign', {
      method: 'POST', body: { filename, content_type: blob.type || 'application/octet-stream' },
    })
    const put = await fetch(sig.upload_url, { method: 'PUT', body: blob, headers: { 'content-type': blob.type || 'application/octet-stream', ...(sig.headers ?? {}) } })
    return put.ok ? sig.file_url : null
  } catch { return null }
}
/** persistGeneratedImages uploads inline (data:) generated images to object storage and swaps the turn
 *  to durable URLs, so the saved conversation keeps them. Best-effort — failures keep the session copy. */
async function persistGeneratedImages(turn: Turn, dataURIs: string[]) {
  const urls = await Promise.all(dataURIs.map(async (uri, i) => {
    if (/^https?:\/\//.test(uri)) return uri
    try {
      const blob = await (await fetch(uri)).blob()
      return (await uploadMedia(blob, `image-${Date.now()}-${i}.png`)) ?? uri
    } catch { return uri }
  }))
  turn.images = urls
  scheduleServerSave()
}
/** clearChat starts a fresh, unsaved chat (the saved ones stay in the sidebar). */
function clearChat() {
  turns.splice(0, turns.length)
  currentConvId.value = null
  if (typeof localStorage !== 'undefined') {
    try { localStorage.removeItem(chatStoreKey.value) } catch { /* ignore */ }
  }
}
/** toggleHistory opens/closes the history panel, refreshing the list when it opens. */
function toggleHistory() {
  historyOpen.value = !historyOpen.value
  if (historyOpen.value) void convs.refresh()
}
/** newChatFromHistory starts a fresh chat from the history panel. */
function newChatFromHistory() { clearChat(); draft.value = ''; historyOpen.value = false }
/** openFromHistory loads a saved conversation and closes the panel. */
async function openFromHistory(id: string) { await openConversation(id); historyOpen.value = false }
/** modelLabel resolves a catalog id to its display name for the history list. */
function modelLabel(id: string): string { return models.value.find((m) => m.id === id)?.name ?? humanizeId(id) }

// =====================================================
// Input + cost preview + send
// =====================================================
const draft = ref('')
const sending = ref(false)

// send() calls the real inference gateway; the live path routes to a model the gateway serves.
// F20×F08 — live credit economics: load the real model catalog (credit price per unit) + the tenant's
// text-credit balance, so each run shows its true credit cost and the session meter reflects real
// spend (the gateway debits the same tokens×price async via the ledger).
const wallet = useWallet()
const catalogPrice = ref<Record<string, { price: number; creditType: string; unit: string }>>({})
const textBalance = ref<number | null>(null)

/** refreshBalance pulls the tenant's live `text` credit balance. */
async function refreshBalance() {
  try {
    const bals = await wallet.loadBalances()
    textBalance.value = Number(bals.find((b) => b.credit_type === 'text')?.balance ?? 0)
  } catch { /* keep the prior value */ }
}

/** liveCreditCost computes a token-priced model's real credit cost from the catalog (null if unknown). */
function liveCreditCost(modelId: string, totalTokens: number): { cost: number; creditType: string } | null {
  const p = catalogPrice.value[modelId]
  if (!p || !p.unit.includes('1K')) return null
  return { cost: (totalTokens / 1000) * p.price, creditType: p.creditType }
}

onMounted(async () => {
  // Restore a previous session first (independent of the catalog) so a refresh keeps the chat + params.
  const saved = loadChat()
  if (saved) {
    if (Array.isArray(saved.turns)) turns.splice(0, turns.length, ...saved.turns)
    if (saved.params && typeof saved.params === 'object') Object.assign(params, saved.params)
    if (saved.convId) currentConvId.value = saved.convId
  }
  void convs.refresh() // load the history sidebar (durable, cross-device)
  try {
    const list = await catalog.load()
    const map: Record<string, { price: number; creditType: string; unit: string }> = {}
    for (const m of list) map[m.id] = { price: Number(m.exascale.price), creditType: m.exascale.credit_type, unit: m.exascale.unit }
    catalogPrice.value = map
    // The saved model wins if the gateway still serves it; otherwise default to the first real model.
    if (saved?.selectedId && list.some((m) => m.id === saved.selectedId)) selectedId.value = saved.selectedId
    else if (!selectedId.value && list.length) selectedId.value = list[0]!.id
  } catch { /* meter falls back to estimates only */ }
  await refreshBalance()
  // From here on, mirror every change to localStorage (instant) + the server (durable, debounced).
  watch([turns, params, selectedId], () => { saveChat(); scheduleServerSave() }, { deep: true })
})

// The gateway serves the 8B/70B Llamas; any other showcase pick routes to 8B for the live call.
const liveServedId = computed(() => (['llama-3.1-8b', 'llama-3.1-70b'].includes(selectedId.value) ? selectedId.value : 'llama-3.1-8b'))

// Session meter — only live turns (those carrying a real creditCost) count toward credit spend.
const liveTurns = computed(() => turns.filter((t) => t.creditCost !== undefined))
const sessionCredits = computed(() => liveTurns.value.reduce((s, t) => s + (t.creditCost ?? 0), 0))
const hasLiveUsage = computed(() => liveTurns.value.length > 0)

// Rough tokens estimate (~4 chars/token)
const draftTokens = computed(() => Math.max(0, Math.ceil(draft.value.length / 4)))

// Use system prompt + (last few turns) + draft as the input estimate
const baseContextTokens = computed(() => {
  const sysT = Math.ceil(params.systemPrompt.length / 4)
  const histT = turns.reduce((s, t) => s + Math.ceil((t.text || stripHtml(t.html ?? '')).length / 4), 0)
  return sysT + histT
})
const estimateInputTokens = computed(() => baseContextTokens.value + draftTokens.value)

// Live credit estimate for the served model (input context + the max-out budget) × catalog price.
const estimateCredits = computed(() => {
  const p = catalogPrice.value[liveServedId.value]
  if (!p || !p.unit.includes('1K')) return 0
  return ((estimateInputTokens.value + params.maxTokens) / 1000) * p.price
})

function stripHtml(html: string) {
  return html.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()
}
/** fmtCredits renders a credit amount, trimmed to ≤6dp (tabular-friendly). */
function fmtCredits(n: number) {
  return (Math.round(n * 1e6) / 1e6).toLocaleString('en-US', { maximumFractionDigits: 6 })
}
function fmtNum(n: number) {
  return n.toLocaleString('en-US')
}
function fmtPrice(m: ModelDef) {
  if (m.priceLabel) return m.priceLabel
  if (m.priceIn !== undefined && m.priceOut !== undefined) {
    return `$${m.priceIn.toFixed(2)} / $${m.priceOut.toFixed(2)}`
  }
  return '—'
}

function nextId() { return Math.max(0, ...turns.map(t => t.id)) + 1 }

// The scroll container + the in-flight stream controller. scrollToBottom keeps the latest turn (and
// the live token stream) pinned to view; stopGenerating lets the user abort mid-response.
const chatStream = ref<HTMLElement | null>(null)
let streamAbort: AbortController | null = null
function scrollToBottom() { const el = chatStream.value; if (el) el.scrollTop = el.scrollHeight }
function stopGenerating() { streamAbort?.abort() }
// Keep the newest turn in view when the transcript grows (new send, streamed reply, restored session).
watch(() => turns.length, () => nextTick(scrollToBottom))

// send streams the prompt through the real inference gateway (BFF SSE) and renders the completion
// token-by-token. The selected model is routed to one the gateway serves; a 402 surfaces as a
// buy-credits hint. A placeholder assistant turn shows a live "thinking" indicator until the first
// token lands, so the user always has immediate feedback.
async function send() {
  const text = draft.value.trim()
  if (!text || sending.value) return
  // Attached files are sent to the model as context, but the chat bubble shows only the user's text
  // (with a 📎 chip), so a 50-line file doesn't drown the conversation.
  let payload = text
  const fileNames = attachments.value.map(a => a.name)
  if (attachments.value.length) {
    payload = attachments.value.map(a => `--- file: ${a.name} ---\n${a.content}`).join('\n\n') + '\n\n' + text
  }
  turns.push({ id: nextId(), role: 'user', text, files: fileNames.length ? fileNames : undefined })
  draft.value = ''
  attachments.value = []
  sending.value = true
  const startedAt = Date.now()
  const liveModel = liveServedId.value
  // Push a streaming placeholder and grab the reactive element (not the raw object) so token mutations
  // re-render live.
  turns.push({ id: nextId(), role: 'assistant', text: '', streaming: true, model: selectedId.value })
  const aTurn = turns[turns.length - 1]!
  await nextTick(); scrollToBottom()
  streamAbort = new AbortController()
  let lastScroll = 0
  const onToken = (delta: string) => {
    aTurn.text += delta
    aTurn.html = renderMarkdownLite(aTurn.text)
    const now = Date.now()
    if (now - lastScroll > 80) { lastScroll = now; scrollToBottom() } // throttle reflow while streaming
  }
  try {
    const { content, usage, aborted } = await useInference().runStream(liveModel, payload, params.maxTokens, {
      onToken, signal: streamAbort.signal,
    })
    finalizeAssistant(aTurn, content || aTurn.text, usage, liveModel, startedAt, aborted)
  } catch (e: unknown) {
    const ex = e as { statusCode?: number; data?: { code?: string; message?: string }; message?: string }
    if (ex?.statusCode === 404 && !aTurn.text) {
      // Streaming route unavailable (older build) and nothing streamed yet → run once non-streamed so
      // the model still answers. No double-charge: the stream never reached a running model.
      try {
        const res = await useInference().run(liveModel, payload, params.maxTokens)
        finalizeAssistant(aTurn, res.content, res.usage, liveModel, startedAt, false)
      } catch (e2: unknown) { renderAssistantError(aTurn, e2) }
    } else if (aTurn.text) {
      // Mid-stream failure after partial output — keep what arrived, mark it finished.
      finalizeAssistant(aTurn, aTurn.text, undefined, liveModel, startedAt, true)
    } else {
      renderAssistantError(aTurn, e)
    }
  } finally {
    sending.value = false
    streamAbort = null
    await nextTick(); scrollToBottom()
  }
}

// finalizeAssistant stamps a completed assistant turn with usage, latency, and the real credit cost
// (estimating token counts when the gateway omits a usage chunk). The server-side ledger debit is the
// source of truth; this is the visible mirror.
function finalizeAssistant(t: Turn, content: string, usage: ChatUsage | undefined, model: string, startedAt: number, aborted: boolean) {
  t.text = content
  t.html = renderMarkdownLite(content + (aborted ? '\n\n_(stopped)_' : ''))
  t.streaming = false
  const out = usage?.completion_tokens ?? Math.max(1, Math.ceil(content.length / 4))
  const total = usage?.total_tokens ?? (estimateInputTokens.value + out)
  t.tokens = { in: usage?.prompt_tokens, out }
  t.latencyMs = Date.now() - startedAt
  const cc = liveCreditCost(model, total)
  t.creditCost = cc?.cost
  t.creditType = cc?.creditType
  t.reqId = 'req_' + Math.random().toString(36).slice(2, 12)
  t.backend = model
  void refreshBalance() // the debit settles async via the ledger; pull the new balance shortly after
}

// renderAssistantError replaces a failed assistant turn with a friendly, actionable message.
function renderAssistantError(t: Turn, e: unknown) {
  const ex = e as { data?: { code?: string; message?: string }; message?: string }
  const msg = ex?.data?.code === 'INSUFFICIENT_CREDIT'
    ? 'Insufficient credit — buy credits in the wallet to run this model.'
    : (ex?.data?.message || ex?.message || 'Inference failed.')
  t.text = msg
  t.html = `<p>${msg}</p>`
  t.streaming = false
}

// ── Image generation — text→image for image-modality models, via the BFF → gateway → DO multimodal.
const generating = ref(false)
/** imageCost is the catalog price for one image of the selected model (image credits). */
const imageCost = computed(() => catalogPrice.value[selectedId.value]?.price ?? 0)

// generateImage runs a text→image request and appends the result as an image turn. A placeholder
// assistant turn shows the thinking indicator until the image arrives; a 402 surfaces the buy hint.
async function generateImage() {
  const prompt = draft.value.trim()
  if (!prompt || generating.value) return
  turns.push({ id: nextId(), role: 'user', text: prompt })
  draft.value = ''
  generating.value = true
  const startedAt = Date.now()
  const modelId = selectedId.value
  turns.push({ id: nextId(), role: 'assistant', text: '', streaming: true, backend: modelId, model: modelId })
  const aTurn = turns[turns.length - 1]!
  await nextTick(); scrollToBottom()
  try {
    const res = await $fetch<{ data: Array<{ b64_json?: string; url?: string }> }>('/api/inference/image', {
      method: 'POST', body: { model: modelId, prompt, n: 1, size: '1024x1024' },
    })
    const imgs = (res.data || [])
      .map((d) => d.url || (d.b64_json ? 'data:image/png;base64,' + d.b64_json : ''))
      .filter(Boolean)
    aTurn.streaming = false
    aTurn.latencyMs = Date.now() - startedAt
    aTurn.backend = modelId
    if (imgs.length) {
      aTurn.images = imgs // show instantly (data URIs); then persist to object storage in the background
      const p = catalogPrice.value[modelId]
      if (p) { aTurn.creditCost = p.price * imgs.length; aTurn.creditType = p.creditType }
      void refreshBalance()
      void persistGeneratedImages(aTurn, imgs)
    } else {
      aTurn.text = 'No image was returned.'
      aTurn.html = '<p>No image was returned.</p>'
    }
  } catch (e: unknown) {
    renderAssistantError(aTurn, e)
  } finally {
    generating.value = false
    await nextTick(); scrollToBottom()
  }
}

// ── Video generation (async) — submit → poll status → play. Billed one clip on submit (video credits).
const videoCost = computed(() => catalogPrice.value[selectedId.value]?.price ?? 0)

// generateVideo submits a text→video job, polls until the clip is ready (or fails/times out), then
// renders it in a <video> player. The placeholder turn shows live elapsed time while it generates.
async function generateVideo() {
  const prompt = draft.value.trim()
  if (!prompt || generating.value) return
  turns.push({ id: nextId(), role: 'user', text: prompt })
  draft.value = ''
  generating.value = true
  const startedAt = Date.now()
  const modelId = selectedId.value
  turns.push({ id: nextId(), role: 'assistant', text: 'Submitting video…', streaming: true, backend: modelId, model: modelId })
  const aTurn = turns[turns.length - 1]!
  await nextTick(); scrollToBottom()
  try {
    const sub = await $fetch<{ id: string; status: string }>('/api/inference/video', {
      method: 'POST', body: { model: modelId, prompt, size: '1280x720' },
    })
    let status = sub.status
    const deadline = Date.now() + 6 * 60 * 1000 // cap polling at ~6 min
    while (status !== 'completed' && status !== 'failed' && Date.now() < deadline) {
      aTurn.text = `Generating video… ${Math.round((Date.now() - startedAt) / 1000)}s`
      await new Promise((r) => setTimeout(r, 5000))
      const poll = await $fetch<{ status: string }>(`/api/inference/video/${encodeURIComponent(sub.id)}`)
      status = poll.status
    }
    aTurn.streaming = false
    aTurn.latencyMs = Date.now() - startedAt
    aTurn.backend = modelId
    if (status === 'completed') {
      aTurn.video = `/api/inference/video/${encodeURIComponent(sub.id)}/content`
      aTurn.text = ''
      const p = catalogPrice.value[modelId]
      if (p) { aTurn.creditCost = p.price; aTurn.creditType = p.creditType } // one clip, billed on submit
      void refreshBalance()
    } else if (status === 'failed') {
      aTurn.text = 'Video generation failed.'
      aTurn.html = '<p>Video generation failed.</p>'
    } else {
      aTurn.text = 'Video is taking longer than expected — it keeps rendering on the server; try again shortly.'
      aTurn.html = '<p>Video is taking longer than expected — it keeps rendering on the server.</p>'
    }
  } catch (e: unknown) {
    renderAssistantError(aTurn, e)
  } finally {
    generating.value = false
    await nextTick(); scrollToBottom()
  }
}

// ── Speech generation (text→speech) — synchronous; the BFF streams WAV audio we play inline.
const speechCost = computed(() => {
  const p = catalogPrice.value[selectedId.value]
  return p ? p.price * (Math.max(1, draftTokens.value * 4) / 1000) : 0 // ~chars/1K × price (rough preview)
})

// generateSpeech sends text to a TTS model and plays the returned audio inline. The result is fetched
// as a blob → object URL (audio isn't kept across refresh — it's regenerated on demand).
async function generateSpeech() {
  const input = draft.value.trim()
  if (!input || generating.value) return
  turns.push({ id: nextId(), role: 'user', text: input })
  draft.value = ''
  generating.value = true
  const startedAt = Date.now()
  const modelId = selectedId.value
  turns.push({ id: nextId(), role: 'assistant', text: '', streaming: true, backend: modelId, model: modelId })
  const aTurn = turns[turns.length - 1]!
  await nextTick(); scrollToBottom()
  try {
    const resp = await fetch('/api/inference/speech', {
      method: 'POST', headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ model: modelId, input }),
    })
    if (!resp.ok) {
      const env = await resp.json().catch(() => null)
      const inner = (env && (env.data || env)) as { code?: string; message?: string } | null
      const err = new Error(inner?.message || 'Speech failed') as Error & { statusCode?: number; data?: unknown }
      err.statusCode = resp.status; err.data = inner
      throw err
    }
    const blob = await resp.blob()
    aTurn.streaming = false
    aTurn.latencyMs = Date.now() - startedAt
    aTurn.audio = URL.createObjectURL(blob) // play instantly; persist to object storage in the background
    aTurn.text = ''
    const p = catalogPrice.value[modelId]
    if (p) { aTurn.creditCost = p.price * (input.length / 1000); aTurn.creditType = p.creditType }
    void refreshBalance()
    void uploadMedia(blob, `speech-${Date.now()}.wav`).then((url) => { if (url) { aTurn.audio = url; scheduleServerSave() } })
  } catch (e: unknown) {
    renderAssistantError(aTurn, e)
  } finally {
    generating.value = false
    await nextTick(); scrollToBottom()
  }
}

// ── Vision (share screen → describe → speak) — capture the screen, send a frame to a VLM, read the
// answer aloud. All browser-native capture + the gateway VLM + the speak() TTS.
const screenStream = ref<MediaStream | null>(null)
const screenVideoEl = ref<HTMLVideoElement | null>(null) // hidden element that plays the shared screen
const sharingScreen = computed(() => !!screenStream.value)

/** stopScreen ends the screen share and releases the capture tracks. */
function stopScreen() {
  screenStream.value?.getTracks().forEach((t) => t.stop())
  screenStream.value = null
}
/** shareScreen prompts for a screen/window/tab to share (or stops an active share). */
async function shareScreen() {
  if (screenStream.value) { stopScreen(); return }
  if (!import.meta.client || !navigator.mediaDevices?.getDisplayMedia) { attachError.value = 'screen share needs a modern browser'; return }
  try {
    const stream = await navigator.mediaDevices.getDisplayMedia({ video: { frameRate: 8 }, audio: false })
    screenStream.value = stream
    stream.getVideoTracks()[0]?.addEventListener('ended', stopScreen) // user stopped via the browser bar
    await nextTick()
    const v = screenVideoEl.value
    if (v) {
      v.srcObject = stream
      v.muted = true // a muted video is allowed to autoplay so frames actually decode for capture
      await v.play().catch(() => {})
    }
  } catch { attachError.value = 'screen share cancelled' }
}
/** captureFrame grabs the current screen frame as a downscaled JPEG data URI. Returns null when the
 *  frame isn't decoded yet or looks entirely black (the hidden <video> hasn't painted / protected content). */
function captureFrame(): string | null {
  const v = screenVideoEl.value
  if (!v || !screenStream.value || !v.videoWidth || v.readyState < 2) return null
  const maxW = 1600
  const scale = Math.min(1, maxW / v.videoWidth)
  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.round(v.videoWidth * scale))
  canvas.height = Math.max(1, Math.round(v.videoHeight * scale))
  const ctx = canvas.getContext('2d')
  if (!ctx) return null
  ctx.drawImage(v, 0, 0, canvas.width, canvas.height)
  // Reject an all-black frame: on the first ask the capture surface often hasn't decoded a frame yet, so
  // we'd otherwise send a blank image and the VLM replies "I can't see your screen".
  try {
    const w = Math.min(48, canvas.width), h = Math.min(48, canvas.height)
    const px = ctx.getImageData(0, 0, w, h).data
    let lit = 0
    for (let i = 0; i < px.length; i += 4) { if (px[i]! > 8 || px[i + 1]! > 8 || px[i + 2]! > 8) lit++ }
    if (lit === 0) return null
  } catch { /* same-origin screen capture — getImageData is permitted */ }
  return canvas.toDataURL('image/jpeg', 0.75)
}
/** captureFrameReady captures, retrying briefly so the first decoded frame has time to paint. */
async function captureFrameReady(): Promise<string | null> {
  for (let i = 0; i < 8; i++) {
    const f = captureFrame()
    if (f) return f
    await new Promise((r) => setTimeout(r, 200))
  }
  return null
}

// askVision captures a screen frame + the prompt, asks the VLM, then speaks the answer aloud.
async function askVision() {
  if (generating.value) return
  if (!screenStream.value) { attachError.value = 'share your screen first'; return }
  const img = await captureFrameReady()
  if (!img) { attachError.value = 'could not read the shared screen — re-share and pick a Screen or Window (a browser Tab works best)'; return }
  const prompt = draft.value.trim() || 'Describe what is on the screen, concisely.'
  turns.push({ id: nextId(), role: 'user', text: prompt + '  🖥️' })
  draft.value = ''
  generating.value = true
  const startedAt = Date.now()
  const modelId = selectedId.value
  turns.push({ id: nextId(), role: 'assistant', text: '', streaming: true, backend: modelId, model: modelId })
  const aTurn = turns[turns.length - 1]!
  await nextTick(); scrollToBottom()
  try {
    const res = await $fetch<{ text: string; usage?: ChatUsage }>('/api/inference/vision', {
      method: 'POST', body: { model: modelId, prompt, image_url: img, max_tokens: 400 },
    })
    aTurn.streaming = false
    aTurn.latencyMs = Date.now() - startedAt
    aTurn.text = res.text || '(no answer)'
    aTurn.html = renderMarkdownLite(aTurn.text)
    const p = catalogPrice.value[modelId]
    if (p && res.usage) { aTurn.creditCost = p.price * (res.usage.total_tokens / 1000); aTurn.creditType = p.creditType }
    void refreshBalance()
    speak(aTurn.text) // vision → speech
  } catch (e: unknown) {
    renderAssistantError(aTurn, e)
  } finally {
    generating.value = false
    await nextTick(); scrollToBottom()
  }
}

// ── Attachments — text/code file content is sent as model context; every file is also uploaded to
// object storage (DigitalOcean Spaces) via a presigned PUT (best-effort) so it persists + gets a URL.
interface Attachment { name: string; content: string; url?: string; uploading?: boolean; failed?: boolean }
const attachments = ref<Attachment[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const attachError = ref('')
function pickFiles() { fileInput.value?.click() }
// isTextLike — read content (→ model context) only for text/code files; binaries (images) upload but
// aren't inlined.
function isTextLike(f: File): boolean {
  return !f.type || f.type.startsWith('text/') || /\.(txt|md|markdown|json|csv|tsv|log|ya?ml|xml|html|css|sql|sh|py|js|ts|tsx|vue|go|java|rb|rs|c|h|cpp)$/i.test(f.name)
}
async function onFiles(e: Event) {
  const input = e.target as HTMLInputElement
  attachError.value = ''
  for (const f of Array.from(input.files ?? [])) {
    if (f.size > 2_000_000) { attachError.value = `${f.name} is too large (max 2 MB)`; continue }
    const att = reactive<Attachment>({ name: f.name, content: isTextLike(f) ? await f.text() : '', uploading: true })
    attachments.value.push(att)
    // Direct-to-Spaces upload via a presigned PUT. Degrades silently when storage is off (501) or the
    // bucket lacks CORS — the file still works as context.
    try {
      const sig = await $fetch<{ upload_url: string; file_url: string; headers?: Record<string, string> }>('/api/files/presign', {
        method: 'POST', body: { filename: f.name, content_type: f.type || 'application/octet-stream' },
      })
      // Echo any headers the presign signed (e.g. x-amz-acl: public-read) — the signature covers them,
      // so the PUT 403s if they're missing.
      const put = await fetch(sig.upload_url, { method: 'PUT', body: f, headers: sig.headers ?? {} })
      if (!put.ok) throw new Error(`upload ${put.status}`)
      att.url = sig.file_url
    } catch { att.failed = true /* storage unavailable / CORS — the file still works as context */ }
    att.uploading = false
  }
  input.value = '' // let the same file be re-picked
}
function removeAttachment(i: number) { attachments.value.splice(i, 1) }

// ── Voice: dictation (speech→text) into the composer, and read-aloud (text→speech) of replies. Both
// use the browser's Web Speech API — Chrome/Edge; degrade gracefully elsewhere. ─────────────────────
const listening = ref(false)
const voiceMode = ref(false) // hands-free conversation loop: listen → send → speak → listen
const speaking = ref(false)  // TTS is talking now (we never listen while speaking → no feedback echo)
const speechSupported = computed(() => import.meta.client && ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window))
let recog: { stop: () => void; start: () => void } | null = null

/**
 * startListening opens one speech-recognition turn, transcribing into the composer. In hands-free voice
 * mode the end of the utterance auto-sends (see onend). Guarded so it never overlaps an active turn,
 * an in-flight response, or the assistant speaking (which would feed TTS audio back into the mic).
 */
function startListening() {
  if (!speechSupported.value) { attachError.value = 'voice input needs Chrome/Edge'; return }
  if (listening.value || speaking.value || sending.value) return
  const SR = (window as unknown as { SpeechRecognition?: new () => unknown; webkitSpeechRecognition?: new () => unknown })
  const Ctor = SR.SpeechRecognition || SR.webkitSpeechRecognition
  if (!Ctor) return
  const base = draft.value.trim() // preserve anything already typed before dictation starts
  const r = new Ctor() as {
    lang: string; interimResults: boolean; continuous: boolean
    onresult: (e: { results: { length: number;[i: number]: { isFinal?: boolean;[j: number]: { transcript: string } } } }) => void
    onend: () => void; onerror: () => void; start: () => void; stop: () => void
  }
  r.lang = 'en-US'; r.interimResults = false; r.continuous = false
  // `results` is cumulative and onresult fires repeatedly as the engine refines the utterance, so
  // rebuild the whole final transcript each event rather than appending deltas — appending re-adds
  // already-captured words and duplicates them ("hello" → "Hello Hello hello Hello hello hello…").
  r.onresult = (ev) => {
    let phrase = ''
    for (let i = 0; i < ev.results.length; i++) {
      const res = ev.results[i]
      if (res && res.isFinal !== false) phrase += res[0]?.transcript ?? ''
    }
    draft.value = (base ? base + ' ' : '') + phrase.trim()
  }
  r.onend = () => {
    listening.value = false
    // Voice mode: a captured utterance auto-sends; silence just goes idle (tap the mic to resume).
    if (voiceMode.value && draft.value.trim() && !sending.value) void runVoiceTurn()
  }
  r.onerror = () => { listening.value = false }
  recog = r
  listening.value = true
  r.start()
}

/** toggleMic — the manual dictation button (transcribe into the composer; independent of voice mode). */
function toggleMic() {
  if (listening.value) { recog?.stop(); return }
  startListening()
}

/** speak reads text aloud via the browser, tracking `speaking` and firing onDone when it finishes. */
function speak(text: string, onDone?: () => void) {
  if (!import.meta.client || !('speechSynthesis' in window)) { onDone?.(); return }
  window.speechSynthesis.cancel()
  const u = new SpeechSynthesisUtterance(text)
  speaking.value = true
  u.onend = () => { speaking.value = false; onDone?.() }
  u.onerror = () => { speaking.value = false; onDone?.() }
  window.speechSynthesis.speak(u)
}

/** runVoiceTurn drives one hands-free round: send the dictated prompt, speak the reply, then re-open
 *  the mic for the next turn — until voice mode is switched off. */
async function runVoiceTurn() {
  await send()
  if (!voiceMode.value) return
  const reply = lastAssistant.value
  if (reply && reply.text && !reply.streaming) speak(reply.text, () => { if (voiceMode.value) startListening() })
  else startListening()
}

/** toggleVoiceMode starts/stops the hands-free conversation loop. */
function toggleVoiceMode() {
  if (!speechSupported.value) { attachError.value = 'voice mode needs Chrome/Edge'; return }
  voiceMode.value = !voiceMode.value
  if (voiceMode.value) {
    startListening()
  } else {
    recog?.stop()
    if (import.meta.client && 'speechSynthesis' in window) window.speechSynthesis.cancel()
    speaking.value = false
  }
}

/** voiceStatus — the live phase shown while hands-free voice mode is active. */
const voiceStatus = computed(() => {
  if (!voiceMode.value) return ''
  if (speaking.value) return 'Speaking…'
  if (sending.value) return 'Thinking…'
  if (listening.value) return 'Listening…'
  return 'Tap mic to talk'
})

function renderMarkdownLite(src: string): string {
  // Tiny markdown-ish renderer: paragraphs, bullets, inline `code` and **bold**.
  const blocks = src.split(/\n\n+/).map(b => b.trim()).filter(Boolean)
  return blocks.map(b => {
    if (b.startsWith('- ')) {
      const items = b.split(/\n- /).map((line, idx) => idx === 0 ? line.slice(2) : line)
      return '<ul>' + items.map(line => '<li>' + inlineMd(line) + '</li>').join('') + '</ul>'
    }
    return '<p>' + inlineMd(b) + '</p>'
  }).join('')
}
function inlineMd(s: string) {
  return s
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\n/g, '<br />')
}
// =====================================================
// View toggle + metadata sidebar
// =====================================================
type View = 'chat' | 'code'
const view = ref<View>('chat')
const metadataOpen = ref(true)

type CodeLang = 'python' | 'curl' | 'javascript'
const codeLang = ref<CodeLang>('python')

const codeSamples = computed(() => {
  const m = selected.value
  const conversation = turns
    .filter(t => t.text || t.html)
    .map(t => ({
      role: t.role,
      content: t.text || stripHtml(t.html ?? ''),
    }))
  const messages = [
    { role: 'system', content: params.systemPrompt },
    ...conversation,
  ]
  const messagesPy = messages.map(msg =>
    `        {"role": "${msg.role}", "content": ${JSON.stringify(msg.content)}}`,
  ).join(',\n')
  const messagesJs = messages.map(msg =>
    `    { role: "${msg.role}", content: ${JSON.stringify(msg.content)} }`,
  ).join(',\n')
  const messagesJson = JSON.stringify(messages, null, 2)

  return {
    python: `from openai import OpenAI

client = OpenAI(
    base_url="https://api.exascale.com/v1",
    api_key="esx_prod_a47b9c…",   # use a scoped key with \`inference\`
)

resp = client.chat.completions.create(
    model="${m.id}",
    temperature=${params.temperature},
    max_tokens=${params.maxTokens},
    top_p=${params.topP},
    messages=[
${messagesPy}
    ],
)

print(resp.choices[0].message.content)
print("usage:", resp.usage)`,

    curl: `curl https://api.exascale.com/v1/chat/completions \\
  -H "Authorization: Bearer esx_prod_a47b9c…" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify({
    model: m.id,
    temperature: params.temperature,
    max_tokens: params.maxTokens,
    top_p: params.topP,
    messages: messagesJson === '' ? [] : JSON.parse(messagesJson),
  }, null, 2)}'`,

    javascript: `import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "https://api.exascale.com/v1",
  apiKey:  process.env.EXASCALE_API_KEY,  // esx_prod_a47b9c…
});

const resp = await client.chat.completions.create({
  model: "${m.id}",
  temperature: ${params.temperature},
  max_tokens: ${params.maxTokens},
  top_p: ${params.topP},
  messages: [
${messagesJs}
  ],
});

console.log(resp.choices[0].message.content);
console.log("usage:", resp.usage);`,
  }
})

const copied = ref('')
async function copyText(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = label
    setTimeout(() => { if (copied.value === label) copied.value = '' }, 1400)
  } catch { /* clipboard unavailable */ }
}

// =====================================================
// cleanup
// =====================================================
</script>

<template>
  <div class="inference-page">
    <!-- Sub-topbar -->
    <div class="subbar">
      <nav class="crumbs">
        <span>Compute</span>
        <span class="sep">›</span>
        <span class="strong">Inference</span>
        <span class="sep">›</span>
        <span class="strong">Playground</span>
      </nav>
      <div class="subbar-right">
        <span class="catalog-meta mono">{{ models.length }} models · curated · live catalog</span>
        <span class="api-pill mono">api · /v1/chat/completions</span>
      </div>
    </div>

    <!-- ── Install the CLI (one-line, copy-able) ──────────────────────────────── -->
    <AppCliInstall class="inf-cli" />

    <main class="layout">
      <!-- ============ LEFT: Model catalog ============ -->
      <aside class="catalog" :class="{ 'cat-open': catalogOpen }">
        <!-- Mobile: a collapsed picker bar; the full list drops down when tapped. -->
        <button type="button" class="catalog-toggle" @click="catalogOpen = !catalogOpen">
          <span class="ct-label">Model</span>
          <span class="ct-current">{{ selected.name }}</span>
          <ChevronDown :size="16" class="ct-caret" :class="{ open: catalogOpen }" />
        </button>
        <div class="catalog-head">
          <div class="search">
            <span class="search-ic" aria-hidden="true">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                <circle cx="7" cy="7" r="4.5" />
                <path d="M10.4 10.4L14 14" />
              </svg>
            </span>
            <input
              v-model="catalogQuery"
              type="text"
              class="search-input"
              :placeholder="`Search ${models.length} models…`"
              autocomplete="off"
            />
          </div>
          <div class="filter-row">
            <button
              v-for="f in FILTERS"
              :key="f"
              type="button"
              class="filter-chip"
              :class="{ active: filter === f }"
              @click="filter = f"
            >
              {{ f === 'all' ? 'All' : CATEGORY_META[f].label.charAt(0) + CATEGORY_META[f].label.slice(1).toLowerCase() }}
            </button>
          </div>
        </div>

        <div class="catalog-list">
          <div
            v-for="m in filteredModels"
            :key="m.id"
            class="model-card"
            :class="{ active: m.id === selectedId }"
            @click="selectModel(m.id)"
          >
            <div class="mc-head">
              <h4 class="mc-name">{{ m.name }}</h4>
              <span class="cat-tag" :class="CATEGORY_META[m.category].cls">{{ CATEGORY_META[m.category].label }}</span>
            </div>
            <div class="mc-provider">
              <span>{{ m.provider }}</span>
              <span v-if="m.version" class="mc-version mono">{{ m.version }}</span>
            </div>
            <div class="mc-price mono">
              <template v-if="m.priceIn !== undefined && m.priceOut !== undefined">
                {{ fmtPrice(m) }}
                <span class="dim">per 1M · in / out</span>
              </template>
              <template v-else>
                {{ m.priceLabel }}
              </template>
            </div>
            <div class="mc-meta mono">
              <template v-if="m.context">{{ m.context }}</template>
              <template v-else-if="m.sub">{{ m.sub }}</template>
              <span v-if="m.speed" class="mc-speed">· {{ m.speed }}</span>
            </div>
            <button
              v-if="m.id !== selectedId"
              type="button"
              class="mc-cta"
              @click.stop="selectModel(m.id)"
            >
              Try in playground
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                <path d="M3 8h10M9 4l4 4-4 4" />
              </svg>
            </button>
            <span v-else class="mc-active">
              <span class="dot" /> Active model
            </span>
          </div>

          <div v-if="filteredModels.length === 0" class="catalog-empty">
            <span class="mono">No models match "{{ catalogQuery }}"</span>
          </div>
        </div>
      </aside>

      <!-- ============ RIGHT: Playground ============ -->
      <section class="playground" :class="{ 'meta-closed': !metadataOpen }">
        <!-- Playground top bar -->
        <header class="pg-top">
          <div class="pg-model">
            <div class="pg-model-name">
              <span class="cat-tag inline" :class="CATEGORY_META[selected.category].cls">{{ CATEGORY_META[selected.category].label }}</span>
              <h2 class="pg-name">{{ selected.name }}</h2>
              <span class="pg-version mono">{{ selected.version }}</span>
              <span class="pg-backend mono dim">· {{ selected.priceLabel }}</span>
            </div>
            <div class="pg-status">
              <span class="status-tag">
                <span class="pulse" />
                <template v-if="lastAssistant?.latencyMs">Ready · {{ lastAssistant.latencyMs }}ms last call</template>
                <template v-else>Ready</template>
              </span>
            </div>
          </div>
          <div class="pg-top-right">
            <div class="view-seg">
              <button
                type="button"
                :class="{ active: view === 'chat' }"
                @click="view = 'chat'"
              >
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <path d="M2.5 4.5h11v6h-4l-3 2v-2H2.5z" />
                </svg>
                Chat
              </button>
              <button
                type="button"
                :class="{ active: view === 'code' }"
                @click="view = 'code'"
              >
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <path d="M5 4l-3 4 3 4M11 4l3 4-3 4" />
                </svg>
                Code
              </button>
            </div>
            <button
              type="button"
              class="icon-btn"
              :class="{ active: historyOpen }"
              :aria-pressed="historyOpen"
              title="Chat history"
              @click="toggleHistory"
            >
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                <circle cx="8" cy="8" r="6" />
                <path d="M8 4.5V8l2.4 1.4" />
              </svg>
            </button>
            <button
              type="button"
              class="icon-btn"
              :class="{ active: paramsOpen }"
              :aria-pressed="paramsOpen"
              :title="paramsOpen ? 'Hide parameters' : 'Show parameters'"
              @click="paramsOpen = !paramsOpen"
            >
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                <circle cx="8" cy="8" r="2" />
                <path d="M8 1.5v2M8 12.5v2M14.5 8h-2M3.5 8h-2M12.6 3.4l-1.4 1.4M4.8 11.2l-1.4 1.4M12.6 12.6l-1.4-1.4M4.8 4.8L3.4 3.4" />
              </svg>
            </button>
            <button
              type="button"
              class="icon-btn"
              :class="{ active: metadataOpen }"
              :title="metadataOpen ? 'Hide metadata' : 'Show metadata'"
              @click="metadataOpen = !metadataOpen"
            >
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                <rect x="2.5" y="2.5" width="11" height="11" />
                <path d="M10 2.5v11" />
              </svg>
            </button>
            <button
              v-if="turns.length"
              type="button"
              class="icon-btn"
              title="Clear chat — empties this saved transcript"
              @click="clearChat"
            >
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                <path d="M3 4.5h10M6 4.5V3h4v1.5M4.5 4.5l.6 8.5h5.8l.6-8.5" />
              </svg>
            </button>
          </div>

          <!-- Chat history (durable, cross-device — platform-core conversations). -->
          <div v-if="historyOpen" class="history-panel">
            <div class="history-head">
              <span class="history-title mono">Chat history</span>
              <button type="button" class="history-new" @click="newChatFromHistory">+ New chat</button>
            </div>
            <ul v-if="convList.length" class="history-list">
              <li v-for="c in convList" :key="c.id" :class="{ active: c.id === currentConvId }">
                <button type="button" class="history-open" @click="openFromHistory(c.id)">
                  <span class="history-name">{{ c.title || 'Untitled chat' }}</span>
                  <span v-if="c.model" class="history-model mono">{{ modelLabel(c.model) }}</span>
                </button>
                <button type="button" class="history-del" title="Delete conversation" @click.stop="deleteConversation(c.id)">×</button>
              </li>
            </ul>
            <p v-else class="history-empty">No saved chats yet — start typing and it'll appear here.</p>
          </div>
        </header>

        <!-- Parameters panel (collapsible) -->
        <section v-if="paramsOpen" class="params">
          <div class="params-grid">
            <div class="param">
              <div class="param-head">
                <label class="param-k">— Temperature</label>
                <span class="param-v mono">{{ params.temperature.toFixed(2) }}</span>
              </div>
              <input
                v-model.number="params.temperature"
                type="range"
                min="0"
                max="2"
                step="0.05"
                class="slider"
              />
            </div>
            <div class="param">
              <div class="param-head">
                <label class="param-k">— Max tokens</label>
                <span class="param-v mono">{{ params.maxTokens }}</span>
              </div>
              <input
                v-model.number="params.maxTokens"
                type="range"
                min="64"
                max="4096"
                step="32"
                class="slider"
              />
            </div>
            <div class="param">
              <div class="param-head">
                <label class="param-k">— Top P</label>
                <span class="param-v mono">{{ params.topP.toFixed(2) }}</span>
              </div>
              <input
                v-model.number="params.topP"
                type="range"
                min="0.1"
                max="1"
                step="0.01"
                class="slider"
              />
            </div>
            <div class="param">
              <div class="param-head">
                <label class="param-k">— Frequency penalty</label>
                <span class="param-v mono">{{ params.freqPenalty.toFixed(2) }}</span>
              </div>
              <input
                v-model.number="params.freqPenalty"
                type="range"
                min="-2"
                max="2"
                step="0.05"
                class="slider"
              />
            </div>
          </div>
          <div class="sys-prompt">
            <div class="param-head">
              <label class="param-k">— System prompt</label>
              <span class="param-meta mono">
                {{ Math.ceil(params.systemPrompt.length / 4) }} tok · prepended every request
              </span>
            </div>
            <textarea
              v-model="params.systemPrompt"
              class="textarea mono-input"
              rows="3"
            />
          </div>
        </section>

        <!-- Playground body: chat or code -->
        <div class="pg-body">
          <div class="pg-main">
            <!-- ===== CHAT ===== -->
            <template v-if="view === 'chat'">
              <div ref="chatStream" class="chat-stream">
                <div
                  v-for="t in turns"
                  :key="t.id"
                  class="turn"
                  :class="t.role"
                >
                  <div class="turn-head">
                    <span class="turn-avatar" :class="t.role">
                      <template v-if="t.role === 'user'">{{ userInitials }}</template>
                      <template v-else>AI</template>
                    </span>
                    <span class="turn-who">
                      {{ t.role === 'user' ? userLabel : turnModelName(t) }}
                    </span>
                    <span v-if="t.role === 'assistant' && t.latencyMs" class="turn-meta mono">
                      {{ t.latencyMs }}ms · {{ t.tokens?.out ?? 0 }} tok out<template v-if="t.creditCost !== undefined"> · {{ fmtCredits(t.creditCost) }} {{ t.creditType }} credits</template>
                    </span>
                    <span v-else-if="t.role === 'assistant' && t.streaming" class="turn-meta mono pulse-meta">
                      <span class="pulse" /> {{ t.text ? 'Streaming…' : 'Thinking…' }}
                    </span>
                    <button
                      v-if="t.role === 'assistant' && t.text && !t.streaming"
                      type="button"
                      class="turn-copy"
                      title="Read aloud"
                      @click="speak(t.text)"
                    >
                      <Volume2 :size="13" />
                    </button>
                    <button
                      v-if="t.role === 'assistant' && t.text && !t.streaming"
                      type="button"
                      class="turn-copy"
                      @click="copyText(t.text, 'turn-' + t.id)"
                    >
                      {{ copied === ('turn-' + t.id) ? '✓ Copied' : 'Copy' }}
                    </button>
                  </div>
                  <div class="turn-body" v-if="t.role === 'user'">
                    <p>{{ t.text }}</p>
                    <div v-if="t.files?.length" class="turn-files">
                      <span v-for="f in t.files" :key="f" class="turn-file"><Paperclip :size="11" /> {{ f }}</span>
                    </div>
                  </div>
                  <!-- Assistant: animated dots until the first token, then live markdown with a blinking caret -->
                  <div class="turn-body assistant-body" v-else>
                    <div v-if="t.video" class="vid-out">
                      <video :src="t.video" controls preload="metadata" playsinline />
                    </div>
                    <div v-else-if="t.audio" class="aud-out">
                      <audio :src="t.audio" controls preload="metadata" />
                    </div>
                    <div v-else-if="t.images && t.images.length" class="img-grid">
                      <a v-for="(src, idx) in t.images" :key="idx" :href="src" target="_blank" rel="noopener" class="img-out" title="Open full size">
                        <img :src="src" :alt="'generated image ' + (idx + 1)" loading="lazy" />
                      </a>
                    </div>
                    <div v-else-if="t.streaming && !t.text" class="thinking" aria-label="Thinking">
                      <span class="thinking-dot" /><span class="thinking-dot" /><span class="thinking-dot" />
                    </div>
                    <div v-else-if="hasCodeBlock(t)" class="md-segmented">
                      <template v-for="(seg, i) in messageSegments(t.text)" :key="i">
                        <div v-if="seg.type === 'md'" class="md" v-html="renderMarkdownLite(seg.content)" />
                        <CodeRunner v-else :code="seg.content" :lang="seg.lang" />
                      </template>
                    </div>
                    <div v-else class="md" :class="{ 'is-streaming': t.streaming }" v-html="t.html ?? renderMarkdownLite(t.text)" />
                  </div>
                </div>
              </div>

              <!-- Composer — chat is text-only; non-text models run via the API/SDK (see below) -->
              <form v-if="selected.category === 'text'" class="composer" @submit.prevent="send">
                <div class="composer-box">
                  <input
                    ref="fileInput" type="file" multiple class="cmp-file" @change="onFiles"
                    accept=".txt,.md,.markdown,.json,.csv,.tsv,.log,.py,.js,.ts,.tsx,.vue,.go,.java,.rb,.rs,.c,.h,.cpp,.html,.css,.yaml,.yml,.xml,.sh,.sql"
                  />
                  <div v-if="attachments.length || attachError" class="cmp-chips">
                    <span
                      v-for="(a, i) in attachments" :key="a.name + i"
                      class="cmp-chip" :class="{ uploading: a.uploading, failed: a.failed }"
                    >
                      <Paperclip :size="12" /> {{ a.name }}
                      <a v-if="a.url" :href="a.url" target="_blank" rel="noopener" class="cmp-chip-link" title="Stored in object storage">↗</a>
                      <span v-else-if="a.uploading" class="cmp-chip-spin" title="Uploading…" />
                      <span v-else-if="a.failed" class="cmp-chip-fail" title="Upload failed — the file is still sent as context">!</span>
                      <button type="button" class="cmp-chip-x" title="Remove" @click="removeAttachment(i)">×</button>
                    </span>
                    <span v-if="attachError" class="cmp-attach-err">{{ attachError }}</span>
                  </div>
                  <textarea
                    v-model="draft"
                    class="composer-input"
                    placeholder="Send a message…  (attach a file or use the mic)"
                    rows="3"
                    @keydown.enter.exact.prevent="send"
                  />
                  <div class="composer-foot">
                    <div class="composer-tools">
                      <button type="button" class="cmp-tool" title="Attach a text or code file" @click="pickFiles">
                        <Paperclip :size="14" />
                      </button>
                      <button
                        type="button" class="cmp-tool" :class="{ rec: listening }"
                        :title="speechSupported ? 'Dictate with your voice' : 'Voice input needs Chrome/Edge'"
                        @click="toggleMic"
                      >
                        <Mic :size="14" />
                      </button>
                      <button
                        type="button" class="cmp-tool" :class="{ active: voiceMode }"
                        :title="speechSupported ? 'Voice mode — speak, hear the reply, repeat (hands-free)' : 'Voice mode needs Chrome/Edge'"
                        @click="toggleVoiceMode"
                      >
                        <Headphones :size="14" />
                      </button>
                      <span v-if="voiceMode" class="composer-hint mono voice"><span class="voice-dot" /> Voice mode · {{ voiceStatus }}</span>
                      <span v-else class="composer-hint mono">{{ listening ? 'Listening…' : 'scoped key required' }}</span>
                    </div>
                    <div class="composer-actions">
                      <span class="cost-preview mono">
                        Estimated cost:
                        <strong>{{ fmtCredits(estimateCredits) }} text credits</strong>
                        <span class="dim">
                          ({{ fmtNum(estimateInputTokens) }} in · {{ fmtNum(params.maxTokens) }} max out)
                        </span>
                      </span>
                      <!-- While generating, the action becomes a Stop button that aborts the stream. -->
                      <button
                        v-if="sending"
                        type="button"
                        class="send-btn stop-btn"
                        @click="stopGenerating"
                      >
                        <span class="stop-square" />
                        Stop
                      </button>
                      <button
                        v-else
                        type="submit"
                        class="send-btn"
                        :disabled="!draft.trim()"
                      >
                        Send
                        <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                          <path d="M3 8h10M9 4l4 4-4 4" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </form>

              <!-- Image models: text→image runs inline (via the BFF → gateway → DO multimodal). -->
              <form v-else-if="selected.category === 'image'" class="composer" @submit.prevent="generateImage">
                <div class="composer-box">
                  <textarea
                    v-model="draft"
                    class="composer-input"
                    placeholder="Describe the image to generate…"
                    rows="3"
                    @keydown.enter.exact.prevent="generateImage"
                  />
                  <div class="composer-foot">
                    <span class="composer-hint mono">{{ selected.name }} · {{ selected.priceLabel }}</span>
                    <div class="composer-actions">
                      <span class="cost-preview mono">Cost: <strong>{{ fmtCredits(imageCost) }} image credits</strong></span>
                      <button v-if="generating" type="button" class="send-btn stop-btn" disabled>
                        <span class="spinner" /> Generating…
                      </button>
                      <button v-else type="submit" class="send-btn" :disabled="!draft.trim()">
                        Generate
                        <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                          <path d="M3 8h10M9 4l4 4-4 4" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </form>

              <!-- Video models: async text→video (submit → poll → play), via the BFF → gateway → DO. -->
              <form v-else-if="selected.category === 'video'" class="composer" @submit.prevent="generateVideo">
                <div class="composer-box">
                  <textarea
                    v-model="draft"
                    class="composer-input"
                    placeholder="Describe the video to generate…  (a clip takes ~1–3 min)"
                    rows="3"
                    @keydown.enter.exact.prevent="generateVideo"
                  />
                  <div class="composer-foot">
                    <span class="composer-hint mono">{{ selected.name }} · {{ selected.priceLabel }}</span>
                    <div class="composer-actions">
                      <span class="cost-preview mono">Cost: <strong>{{ fmtCredits(videoCost) }} video credits</strong></span>
                      <button v-if="generating" type="button" class="send-btn stop-btn" disabled>
                        <span class="spinner" /> Generating…
                      </button>
                      <button v-else type="submit" class="send-btn" :disabled="!draft.trim()">
                        Generate
                        <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                          <path d="M3 8h10M9 4l4 4-4 4" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </form>

              <!-- Speech models: text→speech (synchronous), via the BFF → gateway → DO TTS. -->
              <form v-else-if="selected.category === 'speech'" class="composer" @submit.prevent="generateSpeech">
                <div class="composer-box">
                  <textarea
                    v-model="draft"
                    class="composer-input"
                    placeholder="Type the text to speak…"
                    rows="3"
                    @keydown.enter.exact.prevent="generateSpeech"
                  />
                  <div class="composer-foot">
                    <span class="composer-hint mono">{{ selected.name }} · {{ selected.priceLabel }}</span>
                    <div class="composer-actions">
                      <span class="cost-preview mono">~{{ fmtCredits(speechCost) }} speech credits</span>
                      <button v-if="generating" type="button" class="send-btn stop-btn" disabled>
                        <span class="spinner" /> Generating…
                      </button>
                      <button v-else type="submit" class="send-btn" :disabled="!draft.trim()">
                        Speak
                        <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                          <path d="M3 8h10M9 4l4 4-4 4" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </form>

              <!-- Vision models: share your screen → a frame goes to the VLM → the answer is read aloud. -->
              <form v-else-if="selected.category === 'vision'" class="composer" @submit.prevent="askVision">
                <video ref="screenVideoEl" class="screen-hidden" autoplay muted playsinline />
                <div class="composer-box">
                  <div class="vision-bar">
                    <button
                      type="button" class="cmp-tool" :class="{ active: sharingScreen }"
                      :title="sharingScreen ? 'Stop sharing' : 'Share your screen'" @click="shareScreen"
                    >
                      <Monitor :size="14" />
                    </button>
                    <span class="composer-hint mono">{{ sharingScreen ? 'Sharing — ask about your screen; the answer is read aloud' : 'Share your screen, then ask' }}</span>
                  </div>
                  <textarea
                    v-model="draft"
                    class="composer-input"
                    placeholder="Ask about your screen…  (or leave blank to describe it)"
                    rows="2"
                    @keydown.enter.exact.prevent="askVision"
                  />
                  <div class="composer-foot">
                    <span class="composer-hint mono">{{ selected.name }} · reads the reply aloud</span>
                    <div class="composer-actions">
                      <button v-if="generating" type="button" class="send-btn stop-btn" disabled>
                        <span class="spinner" /> Looking…
                      </button>
                      <button v-else type="submit" class="send-btn" :disabled="!sharingScreen">
                        Ask
                        <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                          <path d="M3 8h10M9 4l4 4-4 4" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </form>

              <!-- Embeddings: not inline (no UI) — point to the API/CLI. Catalog + pricing live. -->
              <div v-else class="api-only">
                <div class="api-only-icon">{{ CATEGORY_META[selected.category].label }}</div>
                <h4>{{ selected.name }} runs via the API</h4>
                <p>{{ CATEGORY_META[selected.category].label.toLowerCase() }} generation isn't in the inline
                  playground yet — call it from the API or CLI with the same OpenAI-compatible interface.</p>
                <button type="button" class="api-only-copy" @click="copyText('exascale infer ' + selected.id, 'api-' + selected.id)">
                  {{ copied === ('api-' + selected.id) ? '✓ Copied' : 'Copy CLI command' }}
                </button>
              </div>
            </template>

            <!-- ===== CODE ===== -->
            <template v-else>
              <div class="code-pane">
                <div class="code-head">
                  <div class="code-tabs">
                    <button
                      type="button"
                      :class="{ active: codeLang === 'python' }"
                      @click="codeLang = 'python'"
                    >Python</button>
                    <button
                      type="button"
                      :class="{ active: codeLang === 'curl' }"
                      @click="codeLang = 'curl'"
                    >cURL</button>
                    <button
                      type="button"
                      :class="{ active: codeLang === 'javascript' }"
                      @click="codeLang = 'javascript'"
                    >JavaScript</button>
                  </div>
                  <div class="code-head-right">
                    <span class="code-meta mono">
                      reproduces the current conversation · OpenAI-compatible
                    </span>
                    <button
                      type="button"
                      class="code-copy"
                      :class="{ copied: copied === 'code-' + codeLang }"
                      @click="copyText(codeSamples[codeLang], 'code-' + codeLang)"
                    >
                      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
                        <rect x="5" y="5" width="8" height="8" />
                        <path d="M3 11V3h8" />
                      </svg>
                      {{ copied === 'code-' + codeLang ? '✓ Copied' : 'Copy' }}
                    </button>
                  </div>
                </div>
                <pre class="code-pre"><code>{{ codeSamples[codeLang] }}</code></pre>
                <div class="code-foot">
                  <span class="mono dim">
                    base URL <code>https://api.exascale.com/v1</code> · auth via Bearer key from
                    <NuxtLink to="/settings#api">Settings · API Keys</NuxtLink>
                  </span>
                </div>
              </div>
            </template>
          </div>

          <!-- ===== Metadata sidebar ===== -->
          <aside v-if="metadataOpen" class="pg-meta">
            <div class="meta-head">
              <span class="meta-eyebrow">— Last response</span>
              <span class="meta-time mono">{{ lastAssistant?.latencyMs ?? 0 }}ms</span>
            </div>
            <dl class="meta-list">
              <dt>Tokens · in / out</dt>
              <dd class="mono">
                {{ fmtNum(lastAssistant?.tokens?.in ?? 0) }} / {{ fmtNum(lastAssistant?.tokens?.out ?? 0) }}
              </dd>

              <dt>Latency · last</dt>
              <dd class="mono">{{ lastAssistant?.latencyMs ?? 0 }}ms</dd>

              <dt>Backend</dt>
              <dd class="mono small">{{ lastAssistant?.backend ?? '—' }}</dd>

              <dt>Cost</dt>
              <dd v-if="lastAssistant?.creditCost !== undefined" class="mono pos">
                {{ fmtCredits(lastAssistant.creditCost) }} {{ lastAssistant.creditType }} credits
              </dd>
              <dd v-else class="mono dim">—</dd>

              <dt>Request ID</dt>
              <dd class="mono small flex">
                <span class="req-id">{{ lastAssistant?.reqId ?? '—' }}</span>
                <button
                  type="button"
                  class="copy-mini"
                  :class="{ copied: copied === 'reqid' }"
                  @click="copyText(lastAssistant?.reqId ?? '', 'reqid')"
                >
                  {{ copied === 'reqid' ? '✓' : 'Copy' }}
                </button>
              </dd>
            </dl>

            <div class="meta-cta">
              <button type="button" class="curl-btn" @click="view = 'code'; codeLang = 'curl'">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <path d="M5 4l-3 4 3 4M11 4l3 4-3 4" />
                </svg>
                View as cURL →
              </button>
            </div>

            <div class="meta-section">
              <div class="meta-eyebrow">— Session totals</div>
              <div class="totals-row">
                <span class="t-k mono">Turns</span>
                <span class="t-v mono">{{ turns.length }}</span>
              </div>
              <div class="totals-row">
                <span class="t-k mono">Total tokens</span>
                <span class="t-v mono">
                  {{ fmtNum(turns.reduce((s, t) => s + (t.tokens?.in ?? 0) + (t.tokens?.out ?? 0), 0)) }}
                </span>
              </div>
              <div class="totals-row">
                <span class="t-k mono">Credits spent</span>
                <span class="t-v mono pos">{{ fmtCredits(sessionCredits) }}</span>
              </div>
              <!-- Live credit meter (once a real run has happened) -->
              <template v-if="hasLiveUsage">
                <div class="totals-row">
                  <span class="t-k mono">Credits spent · session</span>
                  <span class="t-v mono pos">{{ fmtCredits(sessionCredits) }}</span>
                </div>
                <div class="totals-row">
                  <span class="t-k mono">Text balance · live</span>
                  <span class="t-v mono">{{ textBalance === null ? '—' : fmtCredits(textBalance) }}</span>
                </div>
              </template>
            </div>

            <div class="meta-section">
              <div class="meta-eyebrow">— Headers · last request</div>
              <pre class="meta-headers mono">x-exascale-request-id: {{ lastAssistant?.reqId ?? '—' }}
x-exascale-backend:    {{ lastAssistant?.backend ?? '—' }}
x-exascale-region:     us-east-1
x-exascale-model-ver:  {{ selected.version }}
ratelimit-remaining:   58 / 60 RPS</pre>
            </div>
          </aside>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.inference-page {
  --bd-soft: rgba(255, 255, 255, 0.05);
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.55;
  color: var(--text);
  font-feature-settings: 'tnum' on, 'ss01' on;
  background: var(--canvas);
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.dim { color: var(--text-3); }
.pos { color: var(--pos); }

/* ============================================================
   Sub-topbar
   ============================================================ */
.subbar {
  height: 40px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 24px;
  gap: 16px;
  flex-shrink: 0;
  background: var(--canvas);
}
.crumbs {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-3);
  letter-spacing: 0.02em;
}
.crumbs .sep { color: var(--text-3); opacity: 0.6; }
.crumbs .strong { color: var(--text); }
.subbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}
.catalog-meta {
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.api-pill {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--accent);
  background: rgba(74, 144, 226, 0.10);
  border: 1px solid rgba(74, 144, 226, 0.25);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  letter-spacing: 0.06em;
}

/* CLI install strip — sits between the sub-topbar and the split layout (doesn't scroll). */
.inf-cli { margin: 10px 24px; flex-shrink: 0; }

/* ============================================================
   Layout
   ============================================================ */
.layout {
  display: grid;
  grid-template-columns: 340px minmax(0, 1fr);
  flex: 1;
  min-height: 0;
}

/* ============================================================
   LEFT — Catalog
   ============================================================ */
.catalog {
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--canvas);
}
.catalog-head {
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.search {
  display: grid;
  grid-template-columns: 28px 1fr;
  align-items: center;
  height: 34px;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
}
.search:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}
.search-ic {
  color: var(--text-3);
  display: grid;
  place-items: center;
}
.search-ic svg { width: 14px; height: 14px; }
.search-input {
  background: transparent;
  border: 0;
  outline: 0;
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 13px;
  width: 100%;
  padding: 0 12px 0 0;
}
.search-input::placeholder { color: var(--text-3); }

.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.filter-chip {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 120ms, color 120ms, border-color 120ms;
}
.filter-chip:hover { color: var(--text); background: var(--hover); border-color: var(--border-strong); }
.filter-chip.active {
  background: var(--text);
  color: var(--canvas);
  border-color: var(--text);
}

.catalog-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.catalog-list::-webkit-scrollbar { width: 8px; }
.catalog-list::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.08); border-radius: 2px; }
.catalog-list::-webkit-scrollbar-track { background: transparent; }

.model-card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 12px 12px 10px;
  cursor: pointer;
  transition: background 120ms, border-color 120ms;
  display: flex;
  flex-direction: column;
  gap: 6px;
  position: relative;
}
.model-card:hover {
  background: rgba(255, 255, 255, 0.025);
  border-color: var(--border-strong);
}
.model-card.active {
  background: rgba(200, 242, 92, 0.04);
  border-color: var(--brand);
}
.model-card.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--brand);
}

.mc-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
}
.mc-name {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 14px;
  letter-spacing: -0.015em;
  margin: 0;
  color: var(--text);
  line-height: 1.25;
  flex: 1;
  min-width: 0;
}

.cat-tag {
  display: inline-flex;
  align-items: center;
  font-family: var(--font-mono);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.18em;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  flex-shrink: 0;
}
.cat-tag.inline { font-size: 9.5px; padding: 3px 7px; }
.cat-tag.cat-text {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-2);
  border-color: var(--border);
}
.cat-tag.cat-speech {
  background: rgba(74, 144, 226, 0.10);
  color: var(--accent);
  border-color: rgba(74, 144, 226, 0.25);
}
.cat-tag.cat-image {
  background: rgba(245, 158, 11, 0.10);
  color: var(--warn);
  border-color: rgba(245, 158, 11, 0.25);
}
.cat-tag.cat-video {
  background: rgba(239, 68, 68, 0.10);
  color: var(--neg);
  border-color: rgba(239, 68, 68, 0.25);
}
.cat-tag.cat-embed {
  background: rgba(200, 242, 92, 0.10);
  color: var(--brand);
  border-color: rgba(200, 242, 92, 0.25);
}

.mc-provider {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.mc-version {
  font-size: 10px;
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  letter-spacing: 0.04em;
  color: var(--text-2);
}
.mc-price {
  font-size: 12px;
  color: var(--text);
  letter-spacing: 0.02em;
}
.mc-meta {
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.mc-speed { color: var(--text-3); }

.mc-cta {
  margin-top: 4px;
  width: 100%;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10.5px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  font-weight: 600;
  padding: 5px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: background 120ms, color 120ms, border-color 120ms;
}
.mc-cta:hover {
  background: var(--hover);
  color: var(--text);
  border-color: var(--border-strong);
}
.mc-cta svg { width: 11px; height: 11px; }

.mc-active {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--brand);
  font-weight: 600;
  padding: 5px 10px;
  background: rgba(200, 242, 92, 0.06);
  border: 1px solid rgba(200, 242, 92, 0.25);
  border-radius: var(--radius-sm);
  justify-content: center;
}
.mc-active .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--brand);
  animation: pulse-dot 2.4s infinite;
}

.catalog-empty {
  text-align: center;
  color: var(--text-3);
  padding: 24px 12px;
  font-size: 12px;
}

/* ============================================================
   RIGHT — Playground
   ============================================================ */
.playground {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.pg-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-bottom: 1px solid var(--border);
  gap: 16px;
  flex-shrink: 0;
  position: relative;
}
/* Chat history dropdown (durable, cross-device conversations). */
.history-panel {
  position: absolute; top: calc(100% + 6px); right: 20px; z-index: 30;
  width: 320px; max-height: 60vh; overflow-y: auto;
  background: var(--surface, var(--canvas)); border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.history-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 12px; border-bottom: 1px solid var(--border);
  position: sticky; top: 0; background: var(--surface, var(--canvas));
}
.history-title { font-size: 11px; letter-spacing: 0.04em; color: var(--muted, var(--text)); }
.history-new {
  background: none; border: 1px solid var(--border); color: var(--text);
  border-radius: var(--radius-sm); padding: 3px 8px; font-size: 12px; cursor: pointer;
}
.history-new:hover { border-color: var(--accent, var(--text)); color: var(--accent, var(--text)); }
.history-list { list-style: none; margin: 0; padding: 4px; }
.history-list li { display: flex; align-items: center; gap: 4px; border-radius: var(--radius-sm); }
.history-list li:hover, .history-list li.active { background: var(--elevated, var(--canvas)); }
.history-open {
  flex: 1; min-width: 0; display: flex; flex-direction: column; align-items: flex-start; gap: 2px;
  background: none; border: none; color: var(--text); padding: 7px 8px; cursor: pointer; text-align: left;
}
.history-name { font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
.history-model { font-size: 10px; color: var(--muted, var(--text)); }
.history-del {
  background: none; border: none; color: var(--muted, var(--text));
  font-size: 16px; line-height: 1; padding: 4px 8px; cursor: pointer; border-radius: var(--radius-sm);
}
.history-del:hover { color: var(--warn); }
.history-empty { padding: 16px 12px; font-size: 12px; color: var(--muted, var(--text)); text-align: center; }
.pg-model-name {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.pg-name {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.018em;
  margin: 0;
  color: var(--text);
}
.pg-version {
  font-family: var(--font-mono);
  font-size: 11px;
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  color: var(--text-2);
}
.pg-backend {
  font-size: 11px;
  letter-spacing: 0.04em;
}
.pg-status {
  margin-top: 4px;
}
.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--pos);
  background: rgba(25, 195, 125, 0.08);
  border: 1px solid rgba(25, 195, 125, 0.25);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  letter-spacing: 0.04em;
}
.status-tag .pulse {
  width: 5px;
  height: 5px;
  background: var(--pos);
  border-radius: 50%;
  animation: pulse-dot 2.4s infinite;
}
@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.45; }
}

.pg-top-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.view-seg {
  display: inline-flex;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.view-seg button {
  background: transparent;
  border: 0;
  border-right: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 12px;
  height: 30px;
  padding: 0 12px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  letter-spacing: -0.005em;
}
.view-seg button:last-child { border-right: 0; }
.view-seg button:hover { background: var(--hover); color: var(--text); }
.view-seg button.active { background: var(--text); color: var(--canvas); }
.view-seg button svg { width: 12px; height: 12px; }

.icon-btn {
  width: 30px;
  height: 30px;
  background: var(--elevated);
  border: 1px solid var(--border);
  color: var(--text-2);
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.icon-btn:hover { background: var(--hover); color: var(--text); border-color: var(--border-strong); }
.icon-btn.active { background: var(--overlay); color: var(--text); }
.icon-btn svg { width: 14px; height: 14px; }

/* Parameters panel */
.params {
  border-bottom: 1px solid var(--border);
  padding: 14px 20px;
  background: var(--canvas);
}
.params-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin-bottom: 12px;
}
.param { display: flex; flex-direction: column; gap: 6px; }
.param-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.param-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.param-v {
  font-size: 12px;
  color: var(--text);
  font-weight: 600;
}
.param-meta {
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}

.slider {
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  background: transparent;
  cursor: pointer;
}
.slider::-webkit-slider-runnable-track {
  height: 4px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: 2px;
}
.slider::-moz-range-track {
  height: 4px;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: 2px;
}
.slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 14px;
  height: 14px;
  background: var(--brand);
  border: 2px solid var(--canvas);
  border-radius: 50%;
  margin-top: -6px;
  box-shadow: 0 0 0 1px var(--border-strong);
}
.slider::-moz-range-thumb {
  width: 14px;
  height: 14px;
  background: var(--brand);
  border: 2px solid var(--canvas);
  border-radius: 50%;
  box-shadow: 0 0 0 1px var(--border-strong);
}
.slider:focus { outline: none; }

.sys-prompt {
  border-top: 1px dashed var(--bd-soft);
  padding-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.textarea {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 12.5px;
  padding: 10px 12px;
  outline: none;
  resize: vertical;
  line-height: 1.55;
  transition: border-color 120ms, box-shadow 120ms;
}
.textarea:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}
.mono-input {
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.02em;
}

/* ============================================================
   Body: chat or code + metadata sidebar
   ============================================================ */
.pg-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  min-height: 0;
  flex: 1;
  overflow: hidden;
}
.playground.meta-closed .pg-body { grid-template-columns: minmax(0, 1fr); }

.pg-main {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-right: 1px solid var(--border);
}
.playground.meta-closed .pg-main { border-right: 0; }

/* Chat stream */
.chat-stream {
  flex: 1;
  overflow-y: auto;
  padding: 20px 28px 12px;
  display: flex;
  flex-direction: column;
  gap: 22px;
  scroll-behavior: smooth;
}
.chat-stream::-webkit-scrollbar { width: 8px; }
.chat-stream::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.08); border-radius: 2px; }
.chat-stream::-webkit-scrollbar-track { background: transparent; }

.turn { display: flex; flex-direction: column; gap: 8px; }
.turn-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.turn-avatar {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.02em;
}
.turn-avatar.user {
  background: var(--text);
  color: var(--canvas);
}
.turn-avatar.assistant {
  background: var(--brand);
  color: var(--canvas);
}
.turn-who {
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 13px;
  color: var(--text);
  letter-spacing: -0.005em;
}
.turn.user .turn-who { color: var(--text); }
.turn-meta {
  margin-left: auto;
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.turn-meta.pulse-meta {
  color: var(--accent);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.turn-meta .pulse {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--accent);
  animation: pulse-dot 1.4s infinite;
}
.turn-copy {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.turn-copy:hover { color: var(--text); border-color: var(--border-strong); }

.turn-body {
  padding-left: 34px;
  color: var(--text);
  font-size: 14px;
  line-height: 1.65;
}
.turn.user .turn-body p {
  margin: 0;
  color: var(--text);
}
.turn-body :deep(p) { margin: 0 0 10px; }
.turn-body :deep(p:last-child) { margin-bottom: 0; }
.turn-body :deep(ul) {
  margin: 6px 0 10px;
  padding-left: 22px;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.turn-body :deep(ul li) {
  position: relative;
  color: var(--text);
  line-height: 1.55;
}
.turn-body :deep(ul li::before) {
  content: '—';
  position: absolute;
  left: -22px;
  color: var(--text-3);
}
.turn-body :deep(code) {
  font-family: var(--font-mono);
  font-size: 0.88em;
  background: var(--elevated);
  border: 1px solid var(--border);
  padding: 1px 5px;
  border-radius: 2px;
  color: var(--brand);
  letter-spacing: 0.02em;
}
.turn-body :deep(strong) {
  font-weight: 600;
  color: var(--text);
}
.turn-body :deep(.acc) { color: var(--brand); font-weight: 500; }
.turn-body :deep(.cursor) {
  display: inline-block;
  width: 6px;
  height: 14px;
  background: var(--brand);
  vertical-align: -2px;
  margin-left: 2px;
  animation: blink 1s steps(2) infinite;
}
@keyframes blink {
  50% { opacity: 0; }
}

/* Each turn eases in so new messages don't pop. */
.turn { animation: turn-in 0.22s ease both; }
@keyframes turn-in {
  from { opacity: 0; transform: translateY(6px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* "Thinking…" — three bouncing dots shown until the first streamed token arrives. */
.thinking { display: inline-flex; align-items: center; gap: 5px; padding: 4px 2px; }
.thinking-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-3);
  animation: thinking-bounce 1.2s ease-in-out infinite;
}
.thinking-dot:nth-child(2) { animation-delay: 0.16s; }
.thinking-dot:nth-child(3) { animation-delay: 0.32s; }
@keyframes thinking-bounce {
  0%, 80%, 100% { opacity: 0.35; transform: translateY(0); }
  40%           { opacity: 1; transform: translateY(-4px); }
}

/* Blinking caret trailing the live response while tokens are still streaming in. */
.assistant-body .md.is-streaming::after {
  content: '';
  display: inline-block;
  width: 6px;
  height: 14px;
  margin-left: 2px;
  vertical-align: -2px;
  background: var(--brand);
  animation: blink 1s steps(2) infinite;
}

/* Generated-image output grid (text→image turns). */
.img-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 10px; }
.img-out {
  display: block; border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden;
  background: var(--elevated); transition: border-color 120ms;
}
.img-out:hover { border-color: var(--brand); }
.img-out img { display: block; width: 100%; height: auto; }
/* Generated-video player (text→video turns). */
.vid-out { border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden; background: var(--canvas); max-width: 640px; }
.vid-out video { display: block; width: 100%; height: auto; }
/* Generated-audio player (text→speech turns). */
.aud-out { max-width: 420px; }
.aud-out audio { display: block; width: 100%; }
/* Vision (screen share): hidden capture surface + the share toolbar. */
/* Capture surface for screen share: kept off-screen at a real size (NOT display:none / 1px / opacity:0)
 * so the browser actually decodes + paints frames — otherwise drawImage() captures an all-black frame. */
.screen-hidden { position: fixed; left: -10000px; top: 0; width: 480px; height: 270px; pointer-events: none; }
.vision-bar { display: flex; align-items: center; gap: 10px; padding: 10px 12px 0; }

/* Composer */
.composer {
  padding: 16px 28px 20px;
  border-top: 1px solid var(--border);
  background: var(--canvas);
  flex-shrink: 0;
}
.composer-box {
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  display: flex;
  flex-direction: column;
  transition: border-color 120ms, box-shadow 120ms;
}
.composer-box:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.18);
}
.composer-input {
  background: transparent;
  border: 0;
  outline: 0;
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  padding: 12px 14px;
  resize: vertical;
  min-height: 64px;
  max-height: 220px;
  line-height: 1.55;
}
.composer-input::placeholder { color: var(--text-3); }

/* Non-text model panel (image/speech/video/embeddings) — chat playground is text-only. */
.api-only { padding: 40px 32px; text-align: center; color: var(--text-2); }
.api-only-icon {
  display: inline-block; font-family: var(--font-mono); font-size: 11px; letter-spacing: 0.08em;
  color: var(--text-2); border: 1px solid var(--border); border-radius: 2px; padding: 3px 9px; margin-bottom: 16px;
}
.api-only h4 { font-family: var(--font-display); font-size: 18px; color: var(--text); margin: 0 0 8px; font-weight: 600; }
.api-only p { font-size: 13px; line-height: 1.6; max-width: 440px; margin: 0 auto 18px; }
.api-only-copy {
  background: none; border: 1px solid var(--border-strong); border-radius: 2px; padding: 6px 14px;
  font-size: 13px; color: var(--text); cursor: pointer;
}
.api-only-copy:hover { background: rgba(0, 0, 0, 0.03); }

/* Composer: attachments + voice tools */
.cmp-file { display: none; }
.cmp-chips { display: flex; flex-wrap: wrap; gap: 6px; padding: 6px 2px 0; }
.cmp-chip {
  display: inline-flex; align-items: center; gap: 5px; font-size: 12px;
  background: var(--canvas); border: 1px solid var(--border); border-radius: 2px; padding: 2px 6px;
  color: var(--text-2); font-family: var(--font-mono);
}
.cmp-chip-x { background: none; border: none; color: var(--text-3); cursor: pointer; font-size: 14px; line-height: 1; padding: 0 0 0 2px; }
.cmp-chip-x:hover { color: var(--neg); }
.cmp-chip-link { color: var(--accent); text-decoration: none; font-weight: 600; }
.cmp-chip-up { color: var(--text-3); animation: rec-pulse 1s ease-in-out infinite; }
/* Upload states: a spinner while the file PUTs to object storage, a warning glyph if it failed. */
.cmp-chip.uploading { border-color: var(--border-strong); }
.cmp-chip.failed { border-color: var(--warn); color: var(--warn); }
.cmp-chip-spin {
  width: 10px; height: 10px; border-radius: 50%;
  border: 1.5px solid var(--text-3); border-top-color: transparent;
  animation: spin 0.7s linear infinite;
}
.cmp-chip-fail {
  display: inline-flex; align-items: center; justify-content: center;
  width: 13px; height: 13px; border-radius: 50%;
  font-size: 9px; font-weight: 700; color: var(--canvas);
  background: var(--warn);
}
.cmp-attach-err { font-size: 12px; color: var(--neg); align-self: center; }
.composer-tools { display: inline-flex; align-items: center; gap: 6px; }
.cmp-tool {
  display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px;
  background: none; border: 1px solid var(--border-strong); border-radius: 2px; color: var(--text-2); cursor: pointer;
  transition: color 160ms ease, border-color 160ms ease, background-color 160ms ease;
}
.cmp-tool:hover { color: var(--text); background: rgba(0, 0, 0, 0.03); }
.cmp-tool.rec { color: var(--neg); border-color: var(--neg); animation: rec-pulse 1.2s ease-in-out infinite; }
/* Voice mode active — the headset toggle goes solid brand. */
.cmp-tool.active { color: var(--canvas); background: var(--brand); border-color: var(--brand); }
.cmp-tool.active:hover { background: var(--brand-hov); color: var(--canvas); }
@keyframes rec-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
/* Live voice-mode status with a pulsing dot. */
.composer-hint.voice { color: var(--brand); display: inline-flex; align-items: center; gap: 6px; }
.voice-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--brand); animation: rec-pulse 1.2s ease-in-out infinite; }
.turn-files { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 6px; }
.turn-file {
  display: inline-flex; align-items: center; gap: 4px; font-size: 11px; color: var(--text-3);
  font-family: var(--font-mono); border: 1px solid var(--border); border-radius: 2px; padding: 1px 6px;
}

.composer-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px 8px 14px;
  border-top: 1px solid var(--bd-soft);
  gap: 12px;
  flex-wrap: wrap;
}
.composer-hint {
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.composer-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-left: auto;
}
.cost-preview {
  font-size: 11.5px;
  color: var(--text-2);
  letter-spacing: 0.02em;
}
.cost-preview strong {
  color: var(--text);
  font-weight: 600;
}
.send-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 14px;
  background: var(--brand);
  color: var(--canvas);
  border: 0;
  border-radius: var(--radius-sm);
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 12.5px;
  letter-spacing: -0.005em;
  cursor: pointer;
  transition: background 120ms;
}
.send-btn:hover:enabled { background: var(--brand-hov); }
.send-btn:disabled {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-3);
  cursor: not-allowed;
}
.send-btn svg { width: 13px; height: 13px; }
/* Stop button — replaces Send while a response streams; click aborts the in-flight generation. */
.stop-btn { background: var(--elevated); color: var(--text); border: 1px solid var(--border-strong); }
.stop-btn:hover { background: var(--hover); border-color: var(--neg); color: var(--neg); }
.stop-square { width: 9px; height: 9px; border-radius: 1px; background: currentColor; }
.spinner {
  width: 11px;
  height: 11px;
  border: 1.6px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 700ms linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

kbd {
  display: inline-grid;
  place-items: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  font-family: var(--font-mono);
  font-size: 9.5px;
  color: var(--text-2);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border);
  border-bottom-width: 2px;
  border-radius: 2px;
  margin-right: 2px;
}

/* ============================================================
   Code pane
   ============================================================ */
.code-pane {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}
.code-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-bottom: 1px solid var(--border);
  gap: 16px;
}
.code-tabs {
  display: inline-flex;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.code-tabs button {
  background: transparent;
  border: 0;
  border-right: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 12px;
  height: 30px;
  padding: 0 14px;
  cursor: pointer;
  font-weight: 500;
}
.code-tabs button:last-child { border-right: 0; }
.code-tabs button.active { background: var(--text); color: var(--canvas); }
.code-tabs button:hover { background: var(--hover); color: var(--text); }
.code-tabs button.active:hover { background: var(--text); color: var(--canvas); }
.code-head-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.code-meta {
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}
.code-copy {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--elevated);
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  text-transform: uppercase;
  font-weight: 600;
}
.code-copy:hover { color: var(--text); border-color: var(--border-strong); }
.code-copy.copied { color: var(--pos); border-color: rgba(25, 195, 125, 0.40); }
.code-copy svg { width: 11px; height: 11px; stroke: currentColor; fill: none; stroke-width: 1.4; }
.code-pre {
  margin: 0;
  padding: 18px 24px;
  background: rgba(0, 0, 0, 0.32);
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text);
  letter-spacing: 0.02em;
  line-height: 1.6;
  flex: 1;
  overflow: auto;
}
.code-pre::-webkit-scrollbar { width: 8px; height: 8px; }
.code-pre::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.10); border-radius: 2px; }
.code-foot {
  padding: 10px 20px;
  border-top: 1px solid var(--border);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  flex-shrink: 0;
}
.code-foot code {
  font-family: var(--font-mono);
  background: var(--elevated);
  border: 1px solid var(--border);
  padding: 1px 5px;
  border-radius: 2px;
  color: var(--text);
}
.code-foot a {
  color: var(--text);
  text-decoration: none;
  border-bottom: 1px solid var(--border-strong);
  margin-left: 2px;
}
.code-foot a:hover { color: var(--brand); border-bottom-color: var(--brand); }

/* ============================================================
   Metadata sidebar
   ============================================================ */
.pg-meta {
  background: var(--canvas);
  display: flex;
  flex-direction: column;
  padding: 16px 18px;
  gap: 18px;
  overflow-y: auto;
  min-height: 0;
}
.meta-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.meta-eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
}
.meta-time {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text);
  font-weight: 600;
}

.meta-list {
  display: grid;
  grid-template-columns: 1fr;
  margin: 0;
  padding: 0;
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.meta-list dt {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  padding: 8px 12px 2px;
}
.meta-list dd {
  margin: 0;
  padding: 0 12px 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text);
  letter-spacing: 0.02em;
  border-bottom: 1px solid var(--bd-soft);
}
.meta-list dd:last-of-type { border-bottom: 0; }
.meta-list dd.small { font-size: 11px; word-break: break-all; }
.meta-list dd.pos { color: var(--pos); font-weight: 600; }
.meta-list dd.flex {
  display: flex;
  align-items: center;
  gap: 6px;
  justify-content: space-between;
}
.req-id { min-width: 0; overflow: hidden; text-overflow: ellipsis; }
.copy-mini {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.06em;
  padding: 1px 6px;
  border-radius: 2px;
  cursor: pointer;
  flex-shrink: 0;
}
.copy-mini:hover { color: var(--text); border-color: var(--border-strong); }
.copy-mini.copied { color: var(--pos); border-color: rgba(25, 195, 125, 0.40); }

.meta-cta {
  display: flex;
  flex-direction: column;
}
.curl-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: var(--text);
  color: var(--canvas);
  border: 0;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  font-weight: 700;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  justify-content: center;
}
.curl-btn:hover { background: var(--brand); color: var(--canvas); }
.curl-btn svg { width: 12px; height: 12px; }

.meta-section { display: flex; flex-direction: column; gap: 6px; }
.totals-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
  font-size: 11.5px;
  letter-spacing: 0.04em;
  border-bottom: 1px dashed var(--bd-soft);
}
.totals-row:last-child { border-bottom: 0; }
.t-k { color: var(--text-3); font-weight: 600; }
.t-v { color: var(--text); }

.meta-headers {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text);
  padding: 10px 12px;
  letter-spacing: 0.02em;
  line-height: 1.55;
  margin: 0;
  white-space: pre;
  overflow-x: auto;
}

/* ============================================================
   Responsive
   ============================================================ */
@media (max-width: 1200px) {
  .layout { grid-template-columns: 280px minmax(0, 1fr); }
  .pg-body { grid-template-columns: minmax(0, 1fr) 240px; }
  .params-grid { grid-template-columns: repeat(2, 1fr); }
}
/* Desktop: the full catalog is always shown, so the mobile picker bar is hidden. */
.catalog-toggle { display: none; }

@media (max-width: 900px) {
  .layout { grid-template-columns: 1fr; }
  .pg-body { grid-template-columns: 1fr; }
  .pg-meta { display: none; }

  /* Catalog becomes a collapsible model picker (default closed → the chat is primary). */
  .catalog { max-height: none; border-right: 0; border-bottom: 1px solid var(--border); }
  .catalog-toggle {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 12px 16px;
    background: var(--elevated);
    border: 0;
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .ct-label { font-size: 11px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-3); }
  .ct-current { font-weight: 600; font-size: 14px; }
  .ct-caret { margin-left: auto; color: var(--text-3); transition: transform 200ms var(--ease, ease); }
  .ct-caret.open { transform: rotate(180deg); }

  /* Hide the catalog body until the picker is opened. */
  .catalog-head, .catalog-list { display: none; }
  .catalog.cat-open .catalog-head { display: flex; }
  .catalog.cat-open .catalog-list { display: flex; max-height: 60vh; }
}

@media (max-width: 640px) {
  .layout, .playground, .pg-main { min-width: 0; }
  .params-grid { grid-template-columns: 1fr; }
  /* The composer footer wraps so the cost estimate + send button never overflow. */
  .composer-foot { flex-wrap: wrap; gap: 8px; }
  .composer-actions { width: 100%; justify-content: space-between; }
  .cost-preview { font-size: 11px; }
}
</style>
