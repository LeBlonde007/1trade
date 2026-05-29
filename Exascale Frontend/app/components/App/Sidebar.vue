<script setup lang="ts">
/**
 * AppSidebar — icon-only nav for the trading app (64px wide).
 *
 * Items are tagged with the personas they belong to. The active persona
 * (from usePersona()) filters the visible list. Default persona = 'trader'.
 * Users switch personas via the persona-pill at the bottom or via
 * /onboarding/tour.
 */
import {
  CandlestickChart,
  List,
  TrendingUp,
  Briefcase,
  Clock,
  Wallet,
  Server,
  MessageSquare,
  Settings,
  Building2,
  ShieldCheck,
  Banknote,
  Database,
  Check,
} from 'lucide-vue-next'
import { PERSONA_META, type ActivePersona } from '~/composables/usePersona'

const route = useRoute()
const personaCx = usePersona()

interface NavItem {
  icon: unknown
  label: string
  to: string
  match?: string
  personas: ActivePersona[] | 'all'
  group?: 'core' | 'enterprise' | 'partner'
}

const items: NavItem[] = [
  // Core trading surfaces — visible to traders + enterprise (AI co also trades)
  { icon: CandlestickChart, label: 'Trade',     to: '/trade',                          personas: ['trader', 'enterprise'] },
  { icon: List,             label: 'Markets',   to: '/markets/eai-idx', match: '/markets', personas: ['trader', 'enterprise'] },
  { icon: TrendingUp,       label: 'Index',     to: '/benchmark',                      personas: ['trader', 'enterprise'] },
  { icon: Briefcase,        label: 'Portfolio', to: '/portfolio',                      personas: ['trader', 'enterprise'] },
  { icon: Clock,            label: 'History',   to: '/history',                        personas: ['trader', 'enterprise'] },
  { icon: Wallet,           label: 'Wallet',    to: '/wallet',                         personas: ['trader', 'enterprise'] },

  // Compute / inference — enterprise (AI co) primary user
  { icon: Server,           label: 'Compute',   to: '/compute',                        personas: ['enterprise'] },
  { icon: MessageSquare,    label: 'Inference', to: '/inference',                      personas: ['enterprise'] },

  // Enterprise admin
  { icon: Building2,        label: 'Onboarding', to: '/enterprise/onboarding',         personas: ['enterprise'], group: 'enterprise' },
  { icon: ShieldCheck,      label: 'Audit log',  to: '/enterprise/audit',              personas: ['enterprise'], group: 'enterprise' },
  { icon: Banknote,         label: 'Billing',    to: '/enterprise/billing',            personas: ['enterprise'], group: 'enterprise' },

  // Datacenter partner
  { icon: Database,         label: 'DC dashboard', to: '/datacenter', match: '/datacenter', personas: ['partner'], group: 'partner' },
]

const visibleItems = computed(() =>
  items.filter((i) => personaCx.belongsTo(i.personas)),
)

const isActive = (item: { to: string; match?: string }) => {
  const target = item.match ?? item.to
  return route.path === item.to || route.path.startsWith(target)
}

// Persona switcher popover
const popoverOpen = ref(false)
const personaList: ActivePersona[] = ['trader', 'enterprise', 'partner']
function pickPersona(p: ActivePersona) {
  personaCx.set(p)
  popoverOpen.value = false
}
function onDocClick(e: MouseEvent) {
  if (!popoverOpen.value) return
  const t = e.target as HTMLElement | null
  if (t && t.closest('.persona-pop')) return
  popoverOpen.value = false
}
onMounted(()  => document.addEventListener('mousedown', onDocClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))

const initial = computed(() => personaCx.meta.value.short.charAt(0))
</script>

<template>
  <aside class="sidebar" aria-label="App navigation">
    <ul class="nav">
      <li v-for="(item, idx) in visibleItems" :key="idx">
        <NuxtLink
          :to="item.to"
          class="link"
          :class="{ active: isActive(item) }"
          :title="item.label"
          :aria-label="item.label"
        >
          <component :is="item.icon" :size="20" />
          <span class="tooltip">{{ item.label }}</span>
        </NuxtLink>
      </li>
    </ul>

    <div class="bottom">
      <!-- Persona switcher pill -->
      <div class="persona-pop">
        <button
          class="link persona-btn"
          type="button"
          :title="'Persona: ' + personaCx.meta.value.short"
          :aria-label="'Active persona: ' + personaCx.meta.value.short + '. Click to switch.'"
          :aria-expanded="popoverOpen"
          @click="popoverOpen = !popoverOpen"
        >
          <span class="persona-initial">{{ initial }}</span>
          <span class="tooltip">Persona · {{ personaCx.meta.value.short }}</span>
        </button>

        <Transition name="pop">
          <div v-if="popoverOpen" class="pop-panel" role="menu">
            <div class="pop-eyebrow">ACTIVE PERSONA</div>
            <button
              v-for="p in personaList"
              :key="p"
              type="button"
              class="pop-row"
              :class="{ on: personaCx.persona.value === p }"
              role="menuitemradio"
              :aria-checked="personaCx.persona.value === p"
              @click="pickPersona(p)"
            >
              <span class="check"><Check v-if="personaCx.persona.value === p" :size="11" /></span>
              <span class="row-text">
                <span class="row-name">{{ PERSONA_META[p].short }}</span>
                <span class="row-sub">{{ PERSONA_META[p].name }}</span>
              </span>
            </button>
            <div class="pop-foot">
              <NuxtLink to="/onboarding/tour" class="pop-link" @click="popoverOpen = false">
                Replay tour →
              </NuxtLink>
            </div>
          </div>
        </Transition>
      </div>

      <NuxtLink
        to="/settings"
        class="link"
        :class="{ active: isActive({ to: '/settings' }) }"
        title="Settings"
        aria-label="Settings"
      >
        <Settings :size="20" />
        <span class="tooltip">Settings</span>
      </NuxtLink>

      <div class="status" title="Market open">
        <span class="dot" />
      </div>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  grid-area: sidebar;
  display: flex;
  flex-direction: column;
  width: var(--app-sb-w);
  background: var(--canvas);
  border-right: 1px solid var(--border);
  padding: var(--sp-3) 0;
  position: relative;
}

.nav,
.bottom {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--sp-1);
}

.bottom {
  margin-top: auto;
  gap: var(--sp-3);
  padding-bottom: var(--sp-3);
  align-items: center;
}

.link {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  margin: 0 auto;
  color: var(--text-2);
  border-radius: var(--radius-sm);
  border-left: 2px solid transparent;
  text-decoration: none;
  transition: background-color var(--dur) var(--ease), color var(--dur) var(--ease);
  background: transparent;
  border-top: none;
  border-right: none;
  border-bottom: none;
  cursor: pointer;
  font-family: inherit;
}

.link:hover {
  color: var(--text);
  background: var(--hover);
}

.link.active {
  color: var(--brand);
  background: color-mix(in srgb, var(--brand) 8%, transparent);
  border-left-color: var(--brand);
}

/* Tooltip on hover */
.tooltip {
  position: absolute;
  left: calc(100% + 8px);
  top: 50%;
  transform: translateY(-50%);
  padding: 4px 10px;
  background: var(--overlay, var(--elevated));
  color: var(--text);
  font-size: var(--fs-xs);
  border-radius: var(--radius-sm);
  white-space: nowrap;
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--dur) var(--ease);
  z-index: 100;
  border: 1px solid var(--border);
}

.link:hover .tooltip { opacity: 1; }

/* Persona switcher */
.persona-pop { position: relative; }
.persona-btn {
  background: rgba(255,255,255,0.04);
  border: 1px solid var(--border);
}
.persona-initial {
  width: 22px; height: 22px;
  border-radius: 50%;
  background: var(--brand);
  color: var(--text-on-accent);
  display: inline-flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 11px;
  font-family: var(--font-mono);
}

.pop-panel {
  position: absolute;
  left: calc(100% + 8px);
  bottom: 0;
  width: 220px;
  background: var(--overlay, var(--elevated));
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  box-shadow: 0 12px 32px rgba(0,0,0,0.5);
  padding: 6px;
  z-index: 200;
}
.pop-eyebrow {
  font-family: var(--font-mono); font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; color: var(--text-3);
  padding: 6px 8px;
}
.pop-row {
  display: grid;
  grid-template-columns: 14px 1fr;
  gap: 8px;
  width: 100%;
  padding: 8px;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  text-align: left;
  cursor: pointer;
  color: var(--text);
  font-family: inherit;
}
.pop-row:hover { background: var(--hover, rgba(255,255,255,0.04)); }
.pop-row .check {
  display: inline-flex; align-items: center; justify-content: center;
  color: var(--brand);
}
.row-text { display: flex; flex-direction: column; gap: 1px; }
.row-name { font-size: 12px; font-weight: 600; color: var(--text); }
.pop-row.on .row-name { color: var(--brand); }
.row-sub  { font-size: 10.5px; color: var(--text-3); }

.pop-foot {
  padding: 8px;
  border-top: 1px solid var(--border);
  margin-top: 4px;
}
.pop-link {
  display: inline-block;
  font-size: 11.5px; color: var(--accent);
  text-decoration: none;
}
.pop-link:hover { text-decoration: underline; }

.pop-enter-active, .pop-leave-active { transition: opacity 140ms ease, transform 140ms ease; transform-origin: left bottom; }
.pop-enter-from, .pop-leave-to { opacity: 0; transform: scale(0.96) translateX(-4px); }

.status .dot {
  display: block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--pos);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
