<script setup lang="ts">
/**
 * AppCommandPalette — global ⌘K palette.
 *
 * Mounted once inside the `app` layout. Hit ⌘K (or Ctrl+K) anywhere
 * in the authenticated app to open. ↑↓ navigates, ⏎ selects, Esc
 * closes. Chord shortcuts (G then T/P/W/M/H/S/B) jump to pages.
 *
 * Items are filterable by case-insensitive substring across label
 * and hint. Recent items only show when the query is empty.
 */

type Category = 'Recent' | 'Markets' | 'Actions' | 'Pages' | 'Docs'
type IconKey =
  | 'history' | 'plus' | 'briefcase'
  | 'chart' | 'list' | 'wallet' | 'cog' | 'index'
  | 'arrow-up' | 'arrow-down' | 'swap' | 'stop'
  | 'book' | 'shield' | 'pulse'

interface Item {
  id: string
  category: Category
  icon: IconKey
  label: string
  hint?: string
  meta?: string
  metaClass?: 'pos' | 'neg' | 'dim'
  shortcut?: string         // displayed hint, e.g., 'G T'
  chord?: string            // chord activator after 'g', e.g., 't'
  to?: string
  action?: () => void
}

const { isOpen, close } = useCommandPalette()
const router = useRouter()

const query = ref('')
const cursor = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLDivElement | null>(null)

// =====================================================
// Item catalog
// =====================================================
const ALL_ITEMS = computed<Item[]>(() => ([
  // Recent — appears only when query is empty
  { id: 'r-eai',     category: 'Recent', icon: 'history',   label: 'EAI-IDX',        hint: 'AI Index',                                to: '/markets/eai-idx' },
  { id: 'r-buy',     category: 'Recent', icon: 'history',   label: 'Buy credits',    hint: '$10,118 cash available',                  to: '/wallet/buy' },
  { id: 'r-hist',    category: 'Recent', icon: 'history',   label: 'Trade history',  hint: 'Last fill 4 min ago',                      to: '/history' },

  // Markets
  { id: 'm-eai',     category: 'Markets', icon: 'pulse',    label: 'EAI-IDX',         hint: 'AI Index',                meta: '$0.001005 ▲ 0.18%', metaClass: 'pos', to: '/markets/eai-idx' },
  { id: 'm-text',    category: 'Markets', icon: 'pulse',    label: 'TEXT-SPOT',       hint: 'Text Credit',             meta: '$0.001210 ▲ 1.84%', metaClass: 'pos', to: '/markets/text-spot' },
  { id: 'm-image',   category: 'Markets', icon: 'pulse',    label: 'IMAGE-SPOT',      hint: 'Image Credit',            meta: '$0.008000 ▼ 0.62%', metaClass: 'neg', to: '/markets/image-spot' },
  { id: 'm-video',   category: 'Markets', icon: 'pulse',    label: 'VIDEO-SPOT',      hint: 'Video Credit',            meta: '$0.250000 ▲ 0.14%', metaClass: 'pos', to: '/markets/video-spot' },
  { id: 'm-h100',    category: 'Markets', icon: 'pulse',    label: 'H100-SPOT',       hint: 'H100 GPU credit',         meta: '$2.99 ▲ 0.18%',     metaClass: 'pos', to: '/markets/h100-spot' },
  { id: 'm-h200',    category: 'Markets', icon: 'pulse',    label: 'H200-SPOT',       hint: 'H200 GPU credit',         meta: '$3.84 ▼ 0.27%',     metaClass: 'neg', to: '/markets/h200-spot' },

  // Actions
  { id: 'a-buy',     category: 'Actions', icon: 'arrow-up',   label: 'Place buy order…',          hint: 'Open the trade panel pre-filled for buy',            to: '/trade' },
  { id: 'a-sell',    category: 'Actions', icon: 'arrow-down', label: 'Place sell order…',         hint: 'Open the trade panel pre-filled for sell',           to: '/trade' },
  { id: 'a-convert', category: 'Actions', icon: 'swap',       label: 'Convert credits…',          hint: 'Swap between AI / sub-credits at the venue rate',    to: '/wallet' },
  { id: 'a-cash',    category: 'Actions', icon: 'plus',       label: 'Buy credits with cash…',    hint: 'Add funds via card, wire or ACH',                    to: '/wallet/buy' },
  { id: 'a-stop',    category: 'Actions', icon: 'stop',       label: 'Stop all running instances', hint: 'Cancel open orders + flatten compute jobs',         to: '/trade' },

  // Pages (with chord shortcuts)
  { id: 'p-trade',   category: 'Pages', icon: 'chart',     label: 'Trading dashboard',  hint: '/trade',         shortcut: 'G T', chord: 't', to: '/trade' },
  { id: 'p-port',    category: 'Pages', icon: 'briefcase', label: 'Portfolio',          hint: '/portfolio',     shortcut: 'G P', chord: 'p', to: '/portfolio' },
  { id: 'p-mkt',     category: 'Pages', icon: 'list',      label: 'Markets',            hint: '/markets',       shortcut: 'G M', chord: 'm', to: '/markets/eai-idx' },
  { id: 'p-wal',     category: 'Pages', icon: 'wallet',    label: 'Wallet',             hint: '/wallet',        shortcut: 'G W', chord: 'w', to: '/wallet' },
  { id: 'p-hist',    category: 'Pages', icon: 'history',   label: 'History',            hint: '/history',       shortcut: 'G H', chord: 'h', to: '/history' },
  { id: 'p-bench',   category: 'Pages', icon: 'index',     label: 'Index methodology',  hint: '/benchmark',     shortcut: 'G B', chord: 'b', to: '/benchmark' },
  { id: 'p-set',     category: 'Pages', icon: 'cog',       label: 'Settings',           hint: '/settings',      shortcut: 'G S', chord: 's', to: '/settings' },

  // Enterprise / admin
  { id: 'p-ent',     category: 'Admin', icon: 'briefcase', label: 'Enterprise onboarding', hint: '/enterprise/onboarding',          to: '/enterprise/onboarding' },
  { id: 'p-team',    category: 'Admin', icon: 'briefcase', label: 'Team & sub-accounts',   hint: '/enterprise/teams',                to: '/enterprise/teams' },
  { id: 'p-audit',   category: 'Admin', icon: 'shield',    label: 'Audit log',             hint: '/enterprise/audit · compliance',   to: '/enterprise/audit' },
  { id: 'p-bill',    category: 'Admin', icon: 'wallet',    label: 'Billing dashboard',     hint: '/enterprise/billing · invoices, budget, alerts', to: '/enterprise/billing' },

  // Docs
  { id: 'd-api',     category: 'Docs', icon: 'book',  label: 'API Reference',       hint: 'REST + WebSocket · auth, scopes, rate limits' },
  { id: 'd-meth',    category: 'Docs', icon: 'book',  label: 'Index methodology',   hint: 'EAI-IDX calculation · constituents · audit chain', to: '/benchmark' },
  { id: 'd-rules',   category: 'Docs', icon: 'shield',label: 'Trading rules',       hint: 'Order types · circuit breakers · maintenance windows' },
]))

// =====================================================
// Filter + grouping
// =====================================================
const filtered = computed<Item[]>(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) {
    // No query → show Recent first, then all categories
    return ALL_ITEMS.value
  }
  return ALL_ITEMS.value.filter(it => {
    if (it.category === 'Recent') return false
    const hay = (it.label + ' ' + (it.hint ?? '') + ' ' + (it.meta ?? '')).toLowerCase()
    return hay.includes(q)
  })
})

const grouped = computed<Array<{ category: Category; items: Item[] }>>(() => {
  const order: Category[] = ['Recent', 'Markets', 'Actions', 'Pages', 'Docs']
  const map = new Map<Category, Item[]>()
  for (const it of filtered.value) {
    if (!map.has(it.category)) map.set(it.category, [])
    map.get(it.category)!.push(it)
  }
  return order
    .filter(cat => map.has(cat))
    .map(cat => ({ category: cat, items: map.get(cat)! }))
})

const flatItems = computed<Item[]>(() => grouped.value.flatMap(g => g.items))
const cursorItem = computed<Item | null>(() => flatItems.value[cursor.value] ?? null)

watch(query, () => { cursor.value = 0 })
watch(isOpen, (val) => {
  if (val) {
    query.value = ''
    cursor.value = 0
    nextTick(() => inputRef.value?.focus())
  }
})

// =====================================================
// Run an item
// =====================================================
function runItem(it: Item) {
  close()
  if (it.action) {
    it.action()
  } else if (it.to) {
    router.push(it.to)
  }
}

// =====================================================
// Keyboard nav (inside palette)
// =====================================================
function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    cursor.value = Math.min(flatItems.value.length - 1, cursor.value + 1)
    scrollCursorIntoView()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    cursor.value = Math.max(0, cursor.value - 1)
    scrollCursorIntoView()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (cursorItem.value) runItem(cursorItem.value)
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}

function scrollCursorIntoView() {
  nextTick(() => {
    const el = listRef.value?.querySelector<HTMLElement>('.cmd-row.active')
    el?.scrollIntoView({ block: 'nearest' })
  })
}

// =====================================================
// Global open shortcuts (Cmd/Ctrl+K) + chord (G then …)
// =====================================================
let chordActive = false
let chordTimer: ReturnType<typeof setTimeout> | null = null

function isTypingTarget(t: EventTarget | null): boolean {
  if (!(t instanceof HTMLElement)) return false
  const tag = t.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || t.isContentEditable
}

function onGlobalKeyDown(e: KeyboardEvent) {
  // Open / close via ⌘K · Ctrl+K · slash on empty page
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    isOpen.value = !isOpen.value
    return
  }
  if (isOpen.value) return
  if (isTypingTarget(e.target)) return

  // Chord: G then <key>
  if (chordActive) {
    const k = e.key.toLowerCase()
    const match = ALL_ITEMS.value.find(it => it.chord === k && it.to)
    if (match?.to) {
      e.preventDefault()
      router.push(match.to)
    }
    chordActive = false
    if (chordTimer) clearTimeout(chordTimer)
    return
  }
  if (e.key.toLowerCase() === 'g' && !e.metaKey && !e.ctrlKey && !e.altKey) {
    chordActive = true
    if (chordTimer) clearTimeout(chordTimer)
    chordTimer = setTimeout(() => { chordActive = false }, 1200)
    return
  }
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeyDown)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKeyDown)
  if (chordTimer) clearTimeout(chordTimer)
})

// =====================================================
// Cursor flat-index helper for click selection
// =====================================================
function cursorIndexFor(it: Item): number {
  return flatItems.value.findIndex(x => x.id === it.id)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="cmd">
      <div
        v-if="isOpen"
        class="cmd-scrim"
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        @click.self="close"
        @keydown="onKeyDown"
      >
        <div class="cmd-palette" @click.stop>
          <!-- Search input -->
          <div class="cmd-head">
            <span class="cmd-search-ic" aria-hidden="true">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                <circle cx="7" cy="7" r="4.5" />
                <path d="M10.4 10.4L14 14" />
              </svg>
            </span>
            <input
              ref="inputRef"
              v-model="query"
              type="text"
              class="cmd-input"
              placeholder="Search markets, actions, docs…"
              autocomplete="off"
              spellcheck="false"
              autocapitalize="off"
              @keydown="onKeyDown"
            />
            <span class="cmd-shortcut-bg">
              <kbd>⌘</kbd><kbd>K</kbd>
            </span>
          </div>

          <!-- Results -->
          <div ref="listRef" class="cmd-list">
            <template v-if="grouped.length === 0">
              <div class="cmd-empty">
                <span class="cmd-empty-ic" aria-hidden="true">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                    <circle cx="7" cy="7" r="4.5" />
                    <path d="M10.4 10.4L14 14" />
                  </svg>
                </span>
                <div class="cmd-empty-text">
                  No results for "<strong>{{ query }}</strong>"
                  <div class="cmd-empty-sub">Try a market symbol, page name or action verb.</div>
                </div>
              </div>
            </template>

            <template v-for="group in grouped" :key="group.category">
              <div class="cmd-group-head">{{ group.category }}</div>
              <button
                v-for="it in group.items"
                :key="it.id"
                type="button"
                class="cmd-row"
                :class="{ active: cursorIndexFor(it) === cursor }"
                @mouseenter="cursor = cursorIndexFor(it)"
                @click="runItem(it)"
              >
                <span class="cmd-row-ic" :data-icon="it.icon">
                  <!-- inline icons by key -->
                  <svg v-if="it.icon === 'history'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <circle cx="8" cy="8" r="6" />
                    <path d="M8 4v4l2.5 2" />
                  </svg>
                  <svg v-else-if="it.icon === 'plus'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                    <path d="M8 3v10M3 8h10" />
                  </svg>
                  <svg v-else-if="it.icon === 'briefcase'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <rect x="2.5" y="5" width="11" height="8" />
                    <path d="M6 5V3.5h4V5M2.5 9h11" />
                  </svg>
                  <svg v-else-if="it.icon === 'chart'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <path d="M2 12l4-6 3 4 5-7M2 14h12" />
                  </svg>
                  <svg v-else-if="it.icon === 'list'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <path d="M3 4h10M3 8h10M3 12h7" />
                  </svg>
                  <svg v-else-if="it.icon === 'wallet'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <rect x="2.5" y="4.5" width="11" height="8" />
                    <path d="M2.5 6.5h11M11 9h1" />
                  </svg>
                  <svg v-else-if="it.icon === 'cog'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                    <circle cx="8" cy="8" r="2" />
                    <path d="M8 1.5v2M8 12.5v2M14.5 8h-2M3.5 8h-2M12.6 3.4l-1.4 1.4M4.8 11.2l-1.4 1.4M12.6 12.6l-1.4-1.4M4.8 4.8L3.4 3.4" />
                  </svg>
                  <svg v-else-if="it.icon === 'index'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <path d="M2 11l3-3 3 3 5-7" />
                    <path d="M2 14h12" />
                  </svg>
                  <svg v-else-if="it.icon === 'arrow-up'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                    <path d="M8 13V3M4 7l4-4 4 4" />
                  </svg>
                  <svg v-else-if="it.icon === 'arrow-down'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                    <path d="M8 3v10M4 9l4 4 4-4" />
                  </svg>
                  <svg v-else-if="it.icon === 'swap'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <path d="M3 5h8l-2-2M13 11H5l2 2" />
                  </svg>
                  <svg v-else-if="it.icon === 'stop'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                    <rect x="3.5" y="3.5" width="9" height="9" />
                  </svg>
                  <svg v-else-if="it.icon === 'book'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                    <path d="M3 2.5h6L13 6v7.5H3z" />
                    <path d="M9 2.5V6h3M6 9h4M6 11h4" />
                  </svg>
                  <svg v-else-if="it.icon === 'shield'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <path d="M8 2L3 4v4c0 3 2.5 5 5 6 2.5-1 5-3 5-6V4L8 2z" />
                  </svg>
                  <svg v-else-if="it.icon === 'pulse'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                    <path d="M2 8h3l2-4 2 8 2-5 2 3h1" />
                  </svg>
                </span>
                <span class="cmd-row-body">
                  <span class="cmd-row-label">{{ it.label }}</span>
                  <span v-if="it.hint" class="cmd-row-hint">{{ it.hint }}</span>
                </span>
                <span v-if="it.meta" class="cmd-row-meta" :class="it.metaClass">{{ it.meta }}</span>
                <span v-if="it.shortcut" class="cmd-row-shortcut">
                  <kbd v-for="(k, i) in it.shortcut.split(' ')" :key="i">{{ k }}</kbd>
                </span>
              </button>
            </template>
          </div>

          <!-- Footer hint bar -->
          <footer class="cmd-foot">
            <span class="hint-group">
              <kbd>↑</kbd><kbd>↓</kbd>
              <span>navigate</span>
            </span>
            <span class="hint-group">
              <kbd>⏎</kbd>
              <span>select</span>
            </span>
            <span class="hint-group">
              <kbd>esc</kbd>
              <span>close</span>
            </span>
            <span class="hint-group right">
              <span class="dim">tip — </span>
              <kbd>G</kbd><span class="dim">then</span><kbd>T</kbd>
              <span>to jump to Trade</span>
            </span>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* ============================================================
   Scrim
   ============================================================ */
.cmd-scrim {
  position: fixed;
  inset: 0;
  background: rgba(5, 6, 8, 0.62);
  -webkit-backdrop-filter: blur(4px) saturate(110%);
  backdrop-filter: blur(4px) saturate(110%);
  z-index: 200;
  display: grid;
  grid-template-rows: 18vh 1fr;
  justify-items: center;
  padding: 0 16px;
}

/* ============================================================
   Palette
   ============================================================ */
.cmd-palette {
  width: 100%;
  max-width: 640px;
  background: var(--overlay, #1C1F26);
  border: 1px solid var(--border-strong, rgba(255, 255, 255, 0.14));
  border-radius: var(--radius-sm, 2px);
  box-shadow: 0 24px 56px rgba(0, 0, 0, 0.55), 0 2px 6px rgba(0, 0, 0, 0.30);
  color: var(--text, #E8E6E0);
  font-family: var(--font-sans);
  font-feature-settings: 'ss01' on, 'tnum' on;
  display: flex;
  flex-direction: column;
  max-height: 64vh;
  overflow: hidden;
}

/* ============================================================
   Head — search input
   ============================================================ */
.cmd-head {
  display: grid;
  grid-template-columns: 36px 1fr auto;
  align-items: center;
  height: 52px;
  padding: 0 14px 0 16px;
  border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  gap: 8px;
}
.cmd-search-ic {
  color: var(--text-3, #5F5F5C);
  display: grid;
  place-items: center;
}
.cmd-search-ic svg { width: 16px; height: 16px; }
.cmd-input {
  background: transparent;
  border: 0;
  outline: 0;
  color: var(--text, #E8E6E0);
  font-size: 15px;
  font-family: inherit;
  letter-spacing: -0.005em;
  height: 100%;
  width: 100%;
  padding: 0;
}
.cmd-input::placeholder { color: var(--text-3, #5F5F5C); }
.cmd-shortcut-bg {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

/* ============================================================
   Result list
   ============================================================ */
.cmd-list {
  overflow-y: auto;
  flex: 1;
  padding: 6px 0;
  scrollbar-width: thin;
}
.cmd-list::-webkit-scrollbar { width: 8px; }
.cmd-list::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.08); border-radius: 2px; }
.cmd-list::-webkit-scrollbar-track { background: transparent; }

.cmd-group-head {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3, #5F5F5C);
  padding: 12px 16px 6px;
}

.cmd-row {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) auto auto;
  gap: 12px;
  align-items: center;
  width: 100%;
  height: 40px;
  padding: 0 12px 0 14px;
  border: 0;
  background: transparent;
  color: var(--text, #E8E6E0);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  position: relative;
  transition: background 90ms ease-out;
}
.cmd-row::before {
  content: '';
  position: absolute;
  left: 0;
  top: 6px;
  bottom: 6px;
  width: 2px;
  background: transparent;
  transition: background 90ms;
}
.cmd-row.active {
  background: rgba(255, 255, 255, 0.05);
}
.cmd-row.active::before { background: var(--brand, #C8F25C); }

.cmd-row-ic {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  color: var(--text-2, #9A9A95);
  border-radius: 2px;
  flex-shrink: 0;
}
.cmd-row-ic svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.5; }
.cmd-row.active .cmd-row-ic { color: var(--text, #E8E6E0); }
/* Subtle accent for Markets pulse icon */
.cmd-row .cmd-row-ic[data-icon='pulse'] { color: var(--brand, #C8F25C); opacity: 0.85; }

.cmd-row-body {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
  overflow: hidden;
}
.cmd-row-label {
  font-size: 13.5px;
  font-weight: 500;
  letter-spacing: -0.005em;
  color: var(--text, #E8E6E0);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cmd-row-hint {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3, #5F5F5C);
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cmd-row-meta {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 12px;
  letter-spacing: 0.02em;
  color: var(--text-2, #9A9A95);
  white-space: nowrap;
}
.cmd-row-meta.pos { color: var(--pos, #19C37D); }
.cmd-row-meta.neg { color: var(--neg, #EF4444); }
.cmd-row-meta.dim { color: var(--text-3, #5F5F5C); }

.cmd-row-shortcut {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

/* ============================================================
   Empty state
   ============================================================ */
.cmd-empty {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 28px 22px;
  color: var(--text-2, #9A9A95);
}
.cmd-empty-ic {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  border-radius: 2px;
  color: var(--text-3, #5F5F5C);
  flex-shrink: 0;
}
.cmd-empty-ic svg { width: 14px; height: 14px; }
.cmd-empty-text {
  font-size: 13px;
  line-height: 1.5;
}
.cmd-empty-text strong { color: var(--text, #E8E6E0); font-weight: 600; }
.cmd-empty-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3, #5F5F5C);
  letter-spacing: 0.04em;
  margin-top: 4px;
}

/* ============================================================
   Footer hint bar
   ============================================================ */
.cmd-foot {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 10px 16px;
  border-top: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  background: rgba(255, 255, 255, 0.02);
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3, #5F5F5C);
  letter-spacing: 0.04em;
}
.hint-group {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.hint-group.right { margin-left: auto; }
.hint-group .dim { color: var(--text-4, rgba(255, 255, 255, 0.32)); }

kbd {
  display: inline-grid;
  place-items: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-2, #9A9A95);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  border-bottom-width: 2px;
  border-radius: 2px;
  letter-spacing: 0;
}

/* ============================================================
   Open/close transition
   ============================================================ */
.cmd-enter-active,
.cmd-leave-active { transition: opacity 160ms ease-out; }
.cmd-enter-active .cmd-palette,
.cmd-leave-active .cmd-palette {
  transition: transform 200ms cubic-bezier(0.2, 0, 0, 1), opacity 160ms ease-out;
}
.cmd-enter-from,
.cmd-leave-to { opacity: 0; }
.cmd-enter-from .cmd-palette,
.cmd-leave-to .cmd-palette { transform: translateY(-6px) scale(0.99); opacity: 0; }

/* Responsive */
@media (max-width: 640px) {
  .cmd-scrim { grid-template-rows: 8vh 1fr; padding: 0 8px; }
  .cmd-row { grid-template-columns: 28px 1fr auto; }
  .cmd-row-shortcut { display: none; }
}
</style>
