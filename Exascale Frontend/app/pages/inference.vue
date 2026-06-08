<script setup lang="ts">
/**
 * /inference — Inference Playground + Model Catalog.
 *
 * Dark, inside `app` layout. Split-screen: left ~320px model catalog,
 * right ~1fr playground (chat or code view) with a narrow metadata
 * sidebar inside the playground area.
 */

import type { CatalogModel } from '~/composables/useCatalog'

definePageMeta({ layout: 'app' })
useHead({ title: 'Inference Playground — Exascale' })

// =====================================================
// Model catalog
// =====================================================
type Category = 'text' | 'speech' | 'image' | 'video' | 'embed'
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
  if (modality === 'speech' || modality === 'image' || modality === 'video') return modality
  return 'text'
}
/** toModelDef maps a live CatalogModel into the card display shape — real price, in credits. */
function toModelDef(m: CatalogModel): ModelDef {
  const x = m.exascale
  return {
    id: m.id,
    name: humanizeId(m.id),
    provider: m.owned_by || 'Exascale',
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
}

// =====================================================
// Catalog state — search + filter
// =====================================================
const catalogQuery = ref('')
type Filter = 'all' | Category
const FILTERS: Filter[] = ['all', 'text', 'speech', 'image', 'video', 'embed']
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

// Real signed-in identity for the chat transcript (no hardcoded demo user). We only have the email,
// so the label is its local-part and the avatar is up to two initials derived from it.
const { user } = useAuth()
const userLabel = computed(() => (user.value?.email?.split('@')[0]) || 'You')
const userInitials = computed(() => {
  const base = (user.value?.email?.split('@')[0]) || 'you'
  const parts = base.split(/[._-]/).filter(Boolean)
  return ((parts[0]?.[0] || base[0] || 'Y') + (parts[1]?.[0] || '')).toUpperCase()
})

function selectModel(id: string) {
  selectedId.value = id
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
  try {
    const list = await catalog.load()
    const map: Record<string, { price: number; creditType: string; unit: string }> = {}
    for (const m of list) map[m.id] = { price: Number(m.exascale.price), creditType: m.exascale.credit_type, unit: m.exascale.unit }
    catalogPrice.value = map
    if (!selectedId.value && list.length) selectedId.value = list[0]!.id   // select the first real model
  } catch { /* meter falls back to estimates only */ }
  await refreshBalance()
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

// send runs the prompt through the real inference gateway (BFF) and appends the completion with its
// true token usage + credit cost. The selected model is routed to one the gateway serves; a 402
// surfaces as a buy-credits hint.
async function send() {
  const text = draft.value.trim()
  if (!text || sending.value) return
  turns.push({ id: nextId(), role: 'user', text })
  draft.value = ''
  sending.value = true
  const startedAt = Date.now()
  const liveModel = liveServedId.value
  try {
    const res = await useInference().run(liveModel, text, params.maxTokens)
    const cc = liveCreditCost(liveModel, res.usage.total_tokens)
    turns.push({
      id: nextId(), role: 'assistant',
      text: res.content,
      html: renderMarkdownLite(res.content),
      tokens: { in: res.usage.prompt_tokens, out: res.usage.completion_tokens },
      latencyMs: Date.now() - startedAt,
      creditCost: cc?.cost,
      creditType: cc?.creditType,
      reqId: 'req_' + Math.random().toString(36).slice(2, 12),
      backend: liveModel,
    })
    void refreshBalance() // the debit settles async via the ledger; pull the new balance shortly after
  } catch (e: unknown) {
    const ex = e as { data?: { code?: string; message?: string } }
    const msg = ex?.data?.code === 'INSUFFICIENT_CREDIT'
      ? 'Insufficient credit — buy credits in the wallet to run this model.'
      : (ex?.data?.message || 'Inference failed.')
    turns.push({ id: nextId(), role: 'assistant', text: msg, html: `<p>${msg}</p>` })
  } finally {
    sending.value = false
  }
}

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

    <main class="layout">
      <!-- ============ LEFT: Model catalog ============ -->
      <aside class="catalog">
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
              <div class="chat-stream">
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
                      {{ t.role === 'user' ? userLabel : selected.name }}
                    </span>
                    <span v-if="t.role === 'assistant' && t.latencyMs" class="turn-meta mono">
                      {{ t.latencyMs }}ms · {{ t.tokens?.out ?? 0 }} tok out<template v-if="t.creditCost !== undefined"> · {{ fmtCredits(t.creditCost) }} {{ t.creditType }} credits</template>
                    </span>
                    <span v-else-if="t.role === 'assistant'" class="turn-meta mono pulse-meta">
                      <span class="pulse" /> Streaming…
                    </span>
                    <button
                      v-if="t.role === 'assistant' && t.text"
                      type="button"
                      class="turn-copy"
                      @click="copyText(t.text, 'turn-' + t.id)"
                    >
                      {{ copied === ('turn-' + t.id) ? '✓ Copied' : 'Copy' }}
                    </button>
                  </div>
                  <div class="turn-body" v-if="t.role === 'user'">
                    <p>{{ t.text }}</p>
                  </div>
                  <div class="turn-body assistant-body" v-else v-html="t.html ?? renderMarkdownLite(t.text)" />
                </div>
              </div>

              <!-- Composer -->
              <form class="composer" @submit.prevent="send">
                <div class="composer-box">
                  <textarea
                    v-model="draft"
                    class="composer-input"
                    placeholder="Send a message…"
                    rows="3"
                    @keydown.enter.exact.prevent="send"
                  />
                  <div class="composer-foot">
                    <span class="composer-hint mono">
                      <kbd>⏎</kbd> send · <kbd>⇧</kbd><kbd>⏎</kbd> newline · scoped key required
                    </span>
                    <div class="composer-actions">
                      <span class="cost-preview mono">
                        Estimated cost:
                        <strong>{{ fmtCredits(estimateCredits) }} text credits</strong>
                        <span class="dim">
                          ({{ fmtNum(estimateInputTokens) }} in · {{ fmtNum(params.maxTokens) }} max out)
                        </span>
                      </span>
                      <button
                        type="submit"
                        class="send-btn"
                        :disabled="!draft.trim() || sending"
                      >
                        <template v-if="sending">
                          <span class="spinner" />
                          Streaming
                        </template>
                        <template v-else>
                          Send
                          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                            <path d="M3 8h10M9 4l4 4-4 4" />
                          </svg>
                        </template>
                      </button>
                    </div>
                  </div>
                </div>
              </form>
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
}
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
@media (max-width: 900px) {
  .layout { grid-template-columns: 1fr; }
  .catalog { max-height: 360px; border-right: 0; border-bottom: 1px solid var(--border); }
  .pg-body { grid-template-columns: 1fr; }
  .pg-meta { display: none; }
}
</style>
