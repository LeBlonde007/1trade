<script setup lang="ts">
/**
 * AppSidebar — icon-only nav for the trading app (64px wide).
 *
 * Items are tagged with the personas they belong to. The active persona
 * (from usePersona()) filters the visible list. Default persona = 'enterprise' (AI company).
 * Users switch personas via the persona-pill at the bottom or via
 * /onboarding/tour.
 */
import {
  LayoutDashboard,
  CandlestickChart,
  List,
  TrendingUp,
  Briefcase,
  Clock,
  Wallet,
  Server,
  MessageSquare,
  Settings,
  Database,
} from 'lucide-vue-next'
import { type ActivePersona } from '~/composables/usePersona'

const route = useRoute()
const personaCx = usePersona()
// On mobile the rail is an off-canvas drawer; tapping an item should close it.
const sidebar = useSidebar()

interface NavItem {
  icon: unknown
  label: string
  to: string
  match?: string
  personas: ActivePersona[] | 'all'
  group?: 'core' | 'enterprise' | 'partner'
}

const items: NavItem[] = [
  // Console — the AI company's home / overview (first thing after onboarding).
  { icon: LayoutDashboard,  label: 'Console',   to: '/console',                        personas: ['enterprise'] },

  // Exchange / trading surfaces — TRADER only. (The exchange is paused per the GTM pivot; the
  // AI company consumes inference/compute, it does not trade — so these are not on `enterprise`.)
  { icon: CandlestickChart, label: 'Trade',     to: '/trade',                          personas: ['trader'] },
  { icon: List,             label: 'Markets',   to: '/markets/eai-idx', match: '/markets', personas: ['trader'] },
  { icon: TrendingUp,       label: 'Index',     to: '/benchmark',                      personas: ['trader'] },
  { icon: Briefcase,        label: 'Portfolio', to: '/portfolio',                      personas: ['trader'] },
  { icon: Clock,            label: 'History',   to: '/history',                        personas: ['trader'] },

  // Wallet — everyone holds credits.
  { icon: Wallet,           label: 'Wallet',    to: '/wallet',                         personas: 'all' },

  // Compute / inference — the AI company (enterprise).
  { icon: Server,           label: 'Compute',   to: '/compute',                        personas: ['enterprise'] },
  { icon: MessageSquare,    label: 'Inference', to: '/inference',                      personas: ['enterprise'] },

  // AI-company admin (Billing · Audit · Team · SSO) lives under Settings → Account, the single admin
  // home — see app/pages/settings.vue. The sidebar stays product-surface only, so nothing here.

  // Datacenter partner — supply side.
  { icon: Database,         label: 'DC dashboard', to: '/datacenter', match: '/datacenter', personas: ['partner'], group: 'partner' },
]

const visibleItems = computed(() =>
  items.filter((i) => personaCx.belongsTo(i.personas)),
)

const isActive = (item: { to: string; match?: string }) => {
  const target = item.match ?? item.to
  return route.path === item.to || route.path.startsWith(target)
}
// Persona is set at signup/onboarding and only filters the nav (belongsTo) — the in-sidebar persona
// switcher is removed while we work the live AI-company product against test + real data (no mock).
</script>

<template>
  <aside class="sidebar" :class="{ open: sidebar.open.value }" aria-label="App navigation">
    <ul class="nav">
      <li v-for="(item, idx) in visibleItems" :key="idx">
        <NuxtLink
          :to="item.to"
          class="link"
          :class="{ active: isActive(item) }"
          :title="item.label"
          :aria-label="item.label"
          @click="sidebar.close()"
        >
          <component :is="item.icon" :size="20" />
          <span class="tooltip">{{ item.label }}</span>
        </NuxtLink>
      </li>
    </ul>

    <div class="bottom">
      <NuxtLink
        to="/settings"
        class="link"
        :class="{ active: isActive({ to: '/settings' }) }"
        title="Settings"
        aria-label="Settings"
        @click="sidebar.close()"
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

/* ── Mobile: the icon rail becomes an off-canvas drawer with inline labels. ── */
@media (max-width: 768px) {
  .sidebar {
    position: fixed;
    top: var(--topbar-h);
    left: 0;
    bottom: 0;
    width: 240px;
    z-index: 70;
    padding: var(--sp-3) var(--sp-2);
    transform: translateX(-100%);
    transition: transform 220ms cubic-bezier(0.16, 1, 0.3, 1);
    box-shadow: 2px 0 24px rgba(0, 0, 0, 0.4);
    overflow-y: auto;
  }
  .sidebar.open { transform: translateX(0); }

  /* Labelled rows instead of centered icons + hover tooltips (no hover on touch). */
  .nav, .bottom { gap: var(--sp-1); }
  .bottom { align-items: stretch; }
  .link {
    width: 100%;
    height: 44px;
    margin: 0;
    justify-content: flex-start;
    gap: var(--sp-3);
    padding: 0 var(--sp-3);
  }
  .tooltip {
    position: static;
    transform: none;
    opacity: 1;
    pointer-events: auto;
    background: transparent;
    border: 0;
    padding: 0;
    font-size: var(--fs-sm);
    z-index: auto;
  }
}
</style>
