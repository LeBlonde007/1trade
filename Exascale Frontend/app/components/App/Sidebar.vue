<script setup lang="ts">
/**
 * AppSidebar — the platform's primary navigation.
 *
 * Labelled and grouped. It was previously a 64px icon-only rail with hover tooltips, which
 * asked the user to recognise Markets / Index / Portfolio / History from List / TrendingUp /
 * Briefcase / Clock — so finding a screen meant hovering each icon in turn. It also carried a
 * `group` field on every item that was never rendered, leaving up to eight destinations in one
 * flat list. Groups are now real, labels are the default, and the rail collapses to icons for
 * anyone who wants the horizontal space back.
 *
 * Items are tagged with the personas they belong to; the active persona filters the list.
 * The persona switcher is here because persona lives only in localStorage — without a control
 * a user who picked wrong at signup, or who cleared site data, had no way back.
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
  PanelLeftClose,
  PanelLeftOpen,
  Check,
} from 'lucide-vue-next'
import { PERSONA_META, type ActivePersona } from '~/composables/usePersona'

const route = useRoute()
const personaCx = usePersona()
const sidebar = useSidebar()

interface NavItem {
  icon: unknown
  label: string
  to: string
  match?: string
  personas: ActivePersona[] | 'all'
}
interface NavGroup {
  /** Rendered as a section label when expanded; a hairline when collapsed. */
  title: string
  items: NavItem[]
}

// Ordered by how often each surface is actually opened, most-used first.
const groups: NavGroup[] = [
  {
    title: 'Overview',
    items: [
      { icon: LayoutDashboard,  label: 'Console',    to: '/console',   personas: ['enterprise'] },
      { icon: Briefcase,        label: 'Portfolio',  to: '/portfolio', personas: ['trader'] },
      { icon: Database,         label: 'Capacity',   to: '/datacenter', match: '/datacenter', personas: ['partner'] },
    ],
  },
  {
    title: 'Build',
    items: [
      { icon: MessageSquare,    label: 'Inference',  to: '/inference', personas: ['enterprise'] },
      { icon: Server,           label: 'Compute',    to: '/compute',   match: '/compute', personas: ['enterprise'] },
    ],
  },
  {
    title: 'Trade',
    items: [
      { icon: CandlestickChart, label: 'Trade',      to: '/trade',     personas: ['trader'] },
      { icon: List,             label: 'Markets',    to: '/markets/eai-idx', match: '/markets', personas: ['trader'] },
      { icon: Clock,            label: 'History',    to: '/history',   personas: ['trader'] },
    ],
  },
  {
    title: 'Research',
    items: [
      { icon: TrendingUp,       label: 'Index',      to: '/benchmark', personas: ['trader'] },
    ],
  },
  {
    title: 'Account',
    items: [
      { icon: Wallet,           label: 'Wallet',     to: '/wallet',    match: '/wallet', personas: 'all' },
    ],
  },
]

/** Groups with at least one item visible to the active persona. */
const visibleGroups = computed(() =>
  groups
    .map((g) => ({ ...g, items: g.items.filter((i) => personaCx.belongsTo(i.personas)) }))
    .filter((g) => g.items.length > 0),
)

const isActive = (item: { to: string; match?: string }) => {
  const target = item.match ?? item.to
  return route.path === item.to || route.path.startsWith(target)
}

const personaOpen = ref(false)
const personaList = Object.values(PERSONA_META)

/** Switch persona and land on that persona's home, so the nav never shows an empty list. */
function pickPersona(p: ActivePersona) {
  personaCx.set(p)
  personaOpen.value = false
  sidebar.close()
  navigateTo(personaCx.home.value)
}

// Close the persona menu on outside click / Escape.
const personaWrap = ref<HTMLElement | null>(null)
onMounted(() => {
  const onDoc = (e: MouseEvent) => {
    if (personaOpen.value && personaWrap.value && !personaWrap.value.contains(e.target as Node)) {
      personaOpen.value = false
    }
  }
  const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') personaOpen.value = false }
  document.addEventListener('click', onDoc)
  document.addEventListener('keydown', onKey)
  onUnmounted(() => {
    document.removeEventListener('click', onDoc)
    document.removeEventListener('keydown', onKey)
  })
})
</script>

<template>
  <aside
    class="sb"
    :class="{ open: sidebar.open.value, collapsed: sidebar.collapsed.value }"
    aria-label="App navigation"
  >
    <nav class="sb-scroll">
      <div v-for="g in visibleGroups" :key="g.title" class="sb-group">
        <p class="sb-gtitle">{{ g.title }}</p>
        <ul class="sb-list">
          <li v-for="item in g.items" :key="item.to">
            <NuxtLink
              :to="item.to"
              class="sb-link"
              :class="{ active: isActive(item) }"
              :aria-label="item.label"
              @click="sidebar.close()"
            >
              <component :is="item.icon" :size="18" :stroke-width="1.8" class="sb-ico" />
              <span class="sb-label">{{ item.label }}</span>
              <span class="sb-tip">{{ item.label }}</span>
            </NuxtLink>
          </li>
        </ul>
      </div>
    </nav>

    <div class="sb-foot">
      <!-- Persona switcher — the only route back if the wrong one was picked at signup. -->
      <div ref="personaWrap" class="sb-persona-wrap">
        <button
          class="sb-persona"
          type="button"
          :aria-expanded="personaOpen"
          aria-haspopup="menu"
          @click="personaOpen = !personaOpen"
        >
          <span class="sb-persona-badge">{{ personaCx.meta.value.short.charAt(0) }}</span>
          <span class="sb-persona-txt">
            <span class="sb-persona-k">Viewing as</span>
            <span class="sb-persona-v">{{ personaCx.meta.value.short }}</span>
          </span>
          <span class="sb-tip">Viewing as {{ personaCx.meta.value.short }}</span>
        </button>

        <Transition name="sb-pop">
          <ul v-if="personaOpen" class="sb-menu" role="menu">
            <li v-for="p in personaList" :key="p.id" role="none">
              <button class="sb-mitem" type="button" role="menuitem" @click="pickPersona(p.id)">
                <Check :size="14" :stroke-width="2.2" :class="['sb-mcheck', { on: p.id === personaCx.persona.value }]" />
                <span>
                  <span class="sb-mshort">{{ p.short }}</span>
                  <span class="sb-mwho">{{ p.who }}</span>
                </span>
              </button>
            </li>
          </ul>
        </Transition>
      </div>

      <NuxtLink
        to="/settings"
        class="sb-link"
        :class="{ active: isActive({ to: '/settings' }) }"
        aria-label="Settings"
        @click="sidebar.close()"
      >
        <Settings :size="18" :stroke-width="1.8" class="sb-ico" />
        <span class="sb-label">Settings</span>
        <span class="sb-tip">Settings</span>
      </NuxtLink>

      <button class="sb-collapse" type="button" @click="sidebar.toggleCollapsed()">
        <component :is="sidebar.collapsed.value ? PanelLeftOpen : PanelLeftClose" :size="18" :stroke-width="1.8" class="sb-ico" />
        <span class="sb-label">Collapse</span>
        <span class="sb-tip">{{ sidebar.collapsed.value ? 'Expand' : 'Collapse' }}</span>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sb {
  grid-area: sidebar;
  display: flex;
  flex-direction: column;
  width: var(--app-sb-w);
  background: var(--canvas);
  border-right: 1px solid var(--border);
  padding: var(--sp-3) 0 0;
  min-height: 0;
  overflow: hidden;
}

.sb-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0 var(--sp-2);
}

.sb-group + .sb-group { margin-top: var(--sp-4); }

.sb-gtitle {
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide);
  text-transform: uppercase;
  color: var(--text-3);
  margin: 0 0 var(--sp-2);
  padding: 0 var(--sp-3);
  white-space: nowrap;
}

.sb-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }

/* Nav rows and the two footer buttons share one appearance. */
.sb-link,
.sb-persona,
.sb-collapse {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  width: 100%;
  padding: 0 var(--sp-3);
  height: 38px;
  color: var(--text-2);
  background: transparent;
  border: 0;
  border-left: 2px solid transparent;
  border-radius: var(--radius-sm);
  text-decoration: none;
  font-family: inherit;
  font-size: var(--fs-sm);
  text-align: left;
  cursor: pointer;
  white-space: nowrap;
  transition: background-color var(--dur) var(--ease), color var(--dur) var(--ease);
}

.sb-link:hover,
.sb-persona:hover,
.sb-collapse:hover { color: var(--text); background: var(--hover); }

.sb-link.active {
  color: var(--brand);
  background: color-mix(in srgb, var(--brand) 9%, transparent);
  border-left-color: var(--brand);
}

.sb-ico { flex: none; }
.sb-label { overflow: hidden; text-overflow: ellipsis; }

/* Tooltips exist only while collapsed — with labels visible they are noise. */
.sb-tip { display: none; }

.sb-foot {
  border-top: 1px solid var(--border);
  padding: var(--sp-2);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* ── persona switcher ── */
.sb-persona-wrap { position: relative; }
.sb-persona { height: 44px; }
.sb-persona-badge {
  flex: none;
  width: 22px; height: 22px;
  display: grid; place-items: center;
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--brand) 16%, transparent);
  border: 1px solid color-mix(in srgb, var(--brand) 38%, transparent);
  color: var(--brand);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-weight: 600;
}
.sb-persona-txt { display: grid; min-width: 0; line-height: 1.25; }
.sb-persona-k {
  font-family: var(--font-mono); font-size: 9px;
  letter-spacing: var(--ls-wide); text-transform: uppercase; color: var(--text-3);
}
.sb-persona-v { font-size: var(--fs-sm); color: var(--text); overflow: hidden; text-overflow: ellipsis; }

.sb-menu {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  min-width: 208px;
  list-style: none;
  margin: 0;
  padding: var(--sp-1);
  background: var(--overlay);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-2);
  z-index: 120;
}
.sb-mitem {
  display: flex; align-items: flex-start; gap: var(--sp-3);
  width: 100%; padding: var(--sp-2) var(--sp-3);
  background: transparent; border: 0; border-radius: var(--radius-sm);
  color: var(--text-2); font-family: inherit; font-size: var(--fs-sm);
  text-align: left; cursor: pointer;
}
.sb-mitem:hover { background: var(--hover); color: var(--text); }
.sb-mcheck { flex: none; margin-top: 3px; opacity: 0; color: var(--brand); }
.sb-mcheck.on { opacity: 1; }
.sb-mshort { display: block; color: var(--text); }
.sb-mwho { display: block; font-size: var(--fs-xs); color: var(--text-3); }

.sb-pop-enter-active, .sb-pop-leave-active { transition: opacity 140ms var(--ease), transform 140ms var(--ease); }
.sb-pop-enter-from, .sb-pop-leave-to { opacity: 0; transform: translateY(4px); }

/* ── collapsed: icons only, tooltips return ── */
.sb.collapsed .sb-scroll { padding: 0 var(--sp-2); }
.sb.collapsed .sb-label,
.sb.collapsed .sb-persona-txt { display: none; }
.sb.collapsed .sb-gtitle {
  /* a hairline stands in for the group label, so grouping survives the collapse */
  height: 1px; margin: var(--sp-3) var(--sp-2) var(--sp-2);
  padding: 0; overflow: hidden; color: transparent;
  background: var(--border);
}
.sb.collapsed .sb-group:first-child .sb-gtitle { margin-top: 0; }
.sb.collapsed .sb-link,
.sb.collapsed .sb-persona,
.sb.collapsed .sb-collapse { justify-content: center; padding: 0; gap: 0; }
.sb.collapsed .sb-persona { height: 38px; }

.sb.collapsed .sb-tip {
  display: block;
  position: absolute;
  left: calc(100% + 10px);
  top: 50%;
  transform: translateY(-50%);
  padding: 4px 10px;
  background: var(--overlay);
  color: var(--text);
  font-size: var(--fs-xs);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  white-space: nowrap;
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--dur) var(--ease);
  z-index: 120;
}
.sb.collapsed .sb-link:hover .sb-tip,
.sb.collapsed .sb-persona:hover .sb-tip,
.sb.collapsed .sb-collapse:hover .sb-tip { opacity: 1; }

/* ── mobile: off-canvas drawer, always labelled (a phone has no hover) ── */
@media (max-width: 768px) {
  .sb {
    position: fixed;
    inset: var(--topbar-h) auto 0 0;
    width: 244px;
    z-index: 70;
    transform: translateX(-100%);
    transition: transform 220ms var(--ease);
  }
  .sb.open { transform: none; }
  .sb.collapsed .sb-label,
  .sb.collapsed .sb-persona-txt { display: revert; }
  .sb.collapsed .sb-link,
  .sb.collapsed .sb-persona,
  .sb.collapsed .sb-collapse { justify-content: flex-start; padding: 0 var(--sp-3); gap: var(--sp-3); }
  .sb.collapsed .sb-gtitle {
    height: auto; margin: 0 0 var(--sp-2); padding: 0 var(--sp-3);
    color: var(--text-3); background: none; overflow: visible;
  }
  .sb.collapsed .sb-tip { display: none; }
  .sb-collapse { display: none; } /* width is not the user's problem on mobile */
}
</style>
