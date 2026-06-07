<script setup lang="ts">
/**
 * AppNotifications — right-side notifications drawer (~400px wide).
 *
 * Mounted once inside the `app` layout. Opens from the topbar bell
 * button (or programmatically via `useNotifications().open()`).
 * Esc closes. Information-dense and action-oriented — not a social
 * inbox.
 */

type Category = 'trades' | 'account' | 'alerts'

interface Notif {
  id: string
  category: Category
  icon: 'check' | 'alert' | 'budget' | 'key' | 'wrench' | 'sparkle'
  tone: 'pos' | 'neg' | 'warn' | 'info' | 'neutral'
  ts: string
  tsSort: number    // for stable ordering
  title: string
  body?: string
  /** Action text + destination */
  action?: { label: string; to: string }
  unread: boolean
}

const items = ref<Notif[]>([
  {
    id: 'n-1',
    category: 'trades',
    icon: 'check',
    tone: 'pos',
    ts: 'Just now',
    tsSort: 1,
    title: 'Order filled',
    body: 'Bought 1,000 AI Credits at $0.001005 · req_d4a2 · taker',
    action: { label: 'View trade →', to: '/history' },
    unread: true,
  },
  {
    id: 'n-2',
    category: 'alerts',
    icon: 'sparkle',
    tone: 'info',
    ts: '5m ago',
    tsSort: 2,
    title: 'Price alert',
    body: 'AI-INDEX up 0.5% in the last hour · now $0.001005 ▲',
    action: { label: 'Open chart →', to: '/markets/eai-idx' },
    unread: true,
  },
  {
    id: 'n-3',
    category: 'account',
    icon: 'budget',
    tone: 'warn',
    ts: '1h ago',
    tsSort: 3,
    title: 'Budget alert',
    body: '"AI Research" sub-account at 80% of monthly budget ($1.6M of $2M).',
    action: { label: 'Adjust budget →', to: '/enterprise/teams' },
    unread: false,
  },
  {
    id: 'n-4',
    category: 'account',
    icon: 'key',
    tone: 'neutral',
    ts: '3h ago',
    tsSort: 4,
    title: 'New API key created',
    body: '"Production bot" · scopes trade · read · market-data · by jane.doe@walmart.com',
    action: { label: 'Manage keys →', to: '/settings#api' },
    unread: false,
  },
  {
    id: 'n-5',
    category: 'alerts',
    icon: 'wrench',
    tone: 'info',
    ts: '1d ago',
    tsSort: 5,
    title: 'Maintenance scheduled',
    body: '2026-05-20 02:00 UTC · ~5 min · trading paused, order book frozen at last print.',
    unread: false,
  },
  {
    id: 'n-6',
    category: 'account',
    icon: 'sparkle',
    tone: 'neutral',
    ts: '2d ago',
    tsSort: 6,
    title: 'Welcome to Exascale',
    body: 'Take the 60-second tour to find the order book, methodology, and your wallet.',
    action: { label: 'Start tour →', to: '/' },
    unread: false,
  },
])

// =====================================================
// Filtering
// =====================================================
type Filter = 'all' | 'unread' | 'trades' | 'account' | 'alerts'
const FILTERS: Filter[] = ['all', 'unread', 'trades', 'account', 'alerts']
const FILTER_LABELS: Record<Filter, string> = {
  all:      'All',
  unread:   'Unread',
  trades:   'Trades',
  account:  'Account',
  alerts:   'Alerts',
}
const filter = ref<Filter>('all')

const counts = computed(() => ({
  all: items.value.length,
  unread: items.value.filter(n => n.unread).length,
  trades: items.value.filter(n => n.category === 'trades').length,
  account: items.value.filter(n => n.category === 'account').length,
  alerts: items.value.filter(n => n.category === 'alerts').length,
}))

const visible = computed(() => {
  const rows = items.value.filter(n => {
    if (filter.value === 'all')    return true
    if (filter.value === 'unread') return n.unread
    return n.category === filter.value
  })
  return rows.slice().sort((a, b) => a.tsSort - b.tsSort)
})

// =====================================================
// Actions
// =====================================================
const { isOpen, close } = useNotifications()

function markRead(id: string) {
  const n = items.value.find(x => x.id === id)
  if (n) n.unread = false
}
function markAllRead() {
  for (const n of items.value) n.unread = false
}

const router = useRouter()
function runAction(n: Notif) {
  if (n.action) {
    markRead(n.id)
    close()
    router.push(n.action.to)
  } else {
    markRead(n.id)
  }
}

// Esc to close
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && isOpen.value) {
    e.preventDefault()
    close()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
})

// Expose unread count for the topbar badge to read via composable
const notifApi = useNotifications()
// Augment the composable's reactive state via a state-id
const unreadCount = useState<number>('notif-unread', () => 0)
watchEffect(() => {
  unreadCount.value = items.value.filter(n => n.unread).length
})
// Touch to silence unused-warning
void notifApi
</script>

<template>
  <Teleport to="body">
    <Transition name="notif">
      <div
        v-if="isOpen"
        class="notif-scrim"
        role="dialog"
        aria-modal="true"
        aria-label="Notifications"
        @click.self="close"
      >
        <aside class="notif-drawer">
          <!-- ============ Header ============ -->
          <header class="notif-head">
            <div class="head-row">
              <h2 class="head-title">Notifications</h2>
              <div class="head-actions">
                <button
                  type="button"
                  class="link-btn"
                  :disabled="counts.unread === 0"
                  @click="markAllRead"
                >Mark all as read</button>
                <button
                  type="button"
                  class="close-btn"
                  aria-label="Close"
                  @click="close"
                >
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                    <path d="M4 4l8 8M12 4l-8 8" />
                  </svg>
                </button>
              </div>
            </div>
            <div class="filter-row" role="tablist">
              <button
                v-for="f in FILTERS"
                :key="f"
                type="button"
                class="filter-chip"
                :class="{ active: filter === f }"
                role="tab"
                :aria-selected="filter === f"
                @click="filter = f"
              >
                {{ FILTER_LABELS[f] }}
                <span class="chip-count">{{ counts[f] }}</span>
              </button>
            </div>
          </header>

          <!-- ============ Notification list ============ -->
          <div class="notif-list">
            <button
              v-for="n in visible"
              :key="n.id"
              type="button"
              class="notif-row"
              :class="[{ unread: n.unread }, 'tone-' + n.tone]"
              @click="runAction(n)"
            >
              <span class="row-ic" :class="'ic-' + n.tone">
                <svg v-if="n.icon === 'check'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square">
                  <path d="M3 8.5l3.2 3.2L13 5" />
                </svg>
                <svg v-else-if="n.icon === 'alert'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <path d="M8 2l6.5 11h-13L8 2z" />
                  <path d="M8 7v3M8 11.5v0.1" />
                </svg>
                <svg v-else-if="n.icon === 'budget'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <circle cx="8" cy="8" r="6" />
                  <path d="M8 4v4l2.5 2" />
                </svg>
                <svg v-else-if="n.icon === 'key'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <circle cx="5.5" cy="8" r="2.5" />
                  <path d="M8 8h6M11 8v2M13 8v3" />
                </svg>
                <svg v-else-if="n.icon === 'wrench'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <path d="M11 2.5a3 3 0 00-3 3v1L2.5 12l1.5 1.5L9.5 8h1a3 3 0 003-3 3 3 0 00-1-2.2L11.5 4 10 2.5z" />
                </svg>
                <svg v-else viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                  <path d="M8 2l1.5 3.5L13 7l-3.5 1.5L8 12l-1.5-3.5L3 7l3.5-1.5z" />
                </svg>
              </span>

              <div class="row-body">
                <div class="row-head">
                  <span class="row-title">{{ n.title }}</span>
                  <span class="row-ts mono">{{ n.ts }}</span>
                </div>
                <p v-if="n.body" class="row-text">{{ n.body }}</p>
                <span v-if="n.action" class="row-action">{{ n.action.label }}</span>
              </div>

              <span v-if="n.unread" class="unread-dot" aria-label="Unread" />
            </button>

            <div v-if="visible.length === 0" class="empty">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square" class="empty-ic">
                <path d="M4 11V7a4 4 0 018 0v4l1.5 1.5h-11L4 11z" />
                <path d="M6.5 13.5c.3.6.9 1 1.5 1s1.2-.4 1.5-1" />
              </svg>
              <div class="empty-title">You're all caught up.</div>
              <div class="empty-sub">No
                <template v-if="filter === 'unread'"> unread items</template>
                <template v-else-if="filter !== 'all'"> {{ FILTER_LABELS[filter].toLowerCase() }} notifications</template>
                <template v-else> notifications yet</template>
                · check back later.
              </div>
            </div>
          </div>

          <!-- ============ Footer ============ -->
          <footer class="notif-foot">
            <NuxtLink to="/settings" class="foot-link" @click="close">
              Notification settings
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                <path d="M3 8h10M9 4l4 4-4 4" />
              </svg>
            </NuxtLink>
            <span class="foot-meta mono">{{ counts.unread }} unread · {{ counts.all }} total</span>
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* ============================================================
   Scrim
   ============================================================ */
.notif-scrim {
  position: fixed;
  inset: 0;
  background: rgba(5, 6, 8, 0.55);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
  z-index: 180;
  display: flex;
  justify-content: flex-end;
}

/* ============================================================
   Drawer
   ============================================================ */
.notif-drawer {
  width: 400px;
  max-width: 100vw;
  height: 100%;
  background: var(--overlay);
  border-left: 1px solid var(--border-strong, rgba(255, 255, 255, 0.14));
  box-shadow: -24px 0 60px rgba(0, 0, 0, 0.45);
  display: flex;
  flex-direction: column;
  font-family: var(--font-sans);
  color: var(--text);
  font-feature-settings: 'ss01' on, 'tnum' on;
}
.mono {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

/* ============================================================
   Header
   ============================================================ */
.notif-head {
  padding: 14px 18px 12px;
  border-bottom: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.head-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.head-title {
  font-family: var(--font-display);
  font-size: 17px;
  font-weight: 600;
  letter-spacing: -0.015em;
  margin: 0;
  color: var(--text);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.link-btn {
  background: transparent;
  border: 0;
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 12px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--radius-sm, 2px);
  transition: background 120ms, color 120ms;
}
.link-btn:hover:enabled {
  color: var(--text);
  background: rgba(255, 255, 255, 0.04);
}
.link-btn:disabled {
  color: var(--text-3);
  cursor: not-allowed;
}
.close-btn {
  width: 28px;
  height: 28px;
  background: transparent;
  border: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  color: var(--text-2);
  border-radius: var(--radius-sm, 2px);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.close-btn:hover {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text);
}
.close-btn svg { width: 12px; height: 12px; }

/* Filter chips */
.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.filter-chip {
  background: transparent;
  border: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  color: var(--text-2);
  font-family: var(--font-sans);
  font-size: 11.5px;
  font-weight: 500;
  letter-spacing: -0.005em;
  padding: 4px 9px;
  border-radius: var(--radius-sm, 2px);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: background 120ms, color 120ms, border-color 120ms;
}
.filter-chip:hover {
  color: var(--text);
  background: rgba(255, 255, 255, 0.03);
  border-color: var(--border-strong, rgba(255, 255, 255, 0.16));
}
.filter-chip.active {
  background: var(--text);
  color: var(--canvas);
  border-color: var(--text);
  font-weight: 600;
}
.chip-count {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-3);
  background: var(--canvas);
  border: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  padding: 1px 5px;
  border-radius: 2px;
  letter-spacing: 0.02em;
  font-variant-numeric: tabular-nums;
}
.filter-chip.active .chip-count {
  color: var(--canvas);
  background: rgba(255, 255, 255, 0.85);
  border-color: transparent;
}

/* ============================================================
   List
   ============================================================ */
.notif-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}
.notif-list::-webkit-scrollbar { width: 8px; }
.notif-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.08);
  border-radius: 2px;
}
.notif-list::-webkit-scrollbar-track { background: transparent; }

.notif-row {
  display: grid;
  grid-template-columns: 28px 1fr 14px;
  gap: 12px;
  width: 100%;
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  padding: 12px 18px 14px;
  text-align: left;
  font-family: inherit;
  color: inherit;
  cursor: pointer;
  transition: background 120ms;
  position: relative;
}
.notif-row:hover {
  background: rgba(255, 255, 255, 0.03);
}
.notif-row::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 2px;
  background: transparent;
  transition: background 120ms;
}
.notif-row.unread::before {
  background: var(--brand);
}

.row-ic {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-sm, 2px);
  flex-shrink: 0;
  margin-top: 2px;
}
.row-ic svg { width: 14px; height: 14px; }
.row-ic.ic-pos {
  background: rgba(25, 195, 125, 0.12);
  color: var(--pos);
}
.row-ic.ic-neg {
  background: rgba(239, 68, 68, 0.12);
  color: var(--neg);
}
.row-ic.ic-warn {
  background: rgba(245, 158, 11, 0.12);
  color: var(--warn);
}
.row-ic.ic-info {
  background: rgba(74, 144, 226, 0.12);
  color: var(--accent);
}
.row-ic.ic-neutral {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-2);
}

.row-body { min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.row-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
}
.row-title {
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.005em;
  color: var(--text);
  line-height: 1.35;
}
.row-ts {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  flex-shrink: 0;
}
.row-text {
  font-size: 12px;
  color: var(--text-2);
  line-height: 1.5;
  margin: 0;
}
.row-action {
  margin-top: 2px;
  align-self: flex-start;
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--brand);
  font-weight: 600;
  letter-spacing: -0.005em;
  border-bottom: 1px solid transparent;
  padding-bottom: 1px;
  transition: border-color 120ms;
}
.notif-row:hover .row-action {
  border-bottom-color: var(--brand);
}

.unread-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--brand);
  align-self: center;
  flex-shrink: 0;
  box-shadow: 0 0 0 2px rgba(200, 242, 92, 0.15);
}

/* ============================================================
   Empty state
   ============================================================ */
.empty {
  padding: 48px 24px 32px;
  text-align: center;
  color: var(--text-2);
}
.empty-ic {
  width: 28px;
  height: 28px;
  margin: 0 auto 12px;
  color: var(--text-3);
}
.empty-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 4px;
}
.empty-sub {
  font-size: 12px;
  color: var(--text-3);
  line-height: 1.5;
}

/* ============================================================
   Footer
   ============================================================ */
.notif-foot {
  padding: 10px 18px;
  border-top: 1px solid var(--border, rgba(255, 255, 255, 0.08));
  background: rgba(255, 255, 255, 0.02);
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-shrink: 0;
}
.foot-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-2);
  text-decoration: none;
  font-family: var(--font-sans);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: -0.005em;
}
.foot-link:hover { color: var(--text); }
.foot-link svg { width: 12px; height: 12px; }
.foot-meta {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
}

/* ============================================================
   Open / close transition
   ============================================================ */
.notif-enter-active,
.notif-leave-active {
  transition: opacity 160ms ease-out;
}
.notif-enter-active .notif-drawer,
.notif-leave-active .notif-drawer {
  transition: transform 240ms cubic-bezier(0.2, 0, 0, 1);
}
.notif-enter-from,
.notif-leave-to { opacity: 0; }
.notif-enter-from .notif-drawer,
.notif-leave-to .notif-drawer { transform: translateX(100%); }
</style>
