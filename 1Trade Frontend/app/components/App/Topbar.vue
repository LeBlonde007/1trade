<script setup lang="ts">
/**
 * AppTopbar — top bar for authenticated trading app.
 * Holds the live index ticker, balance pill, utility icons, and the
 * user/org menu (M5).
 */
import {
  Search,
  Bell,
  ChevronDown,
  User as UserIcon,
  Settings as SettingsIcon,
  Key,
  HelpCircle,
  Activity,
  LogOut,
  Building2,
  ExternalLink,
  Wallet,
  Menu,
} from 'lucide-vue-next'

const palette = useCommandPalette()
const notifications = useNotifications()
const sidebar = useSidebar() // mobile drawer toggle (hamburger)
const unreadCount = useState<number>('notif-unread', () => 2)

// Persona drives the brand link + which chrome shows: the AI-Index market pill + USD trading balance
// are trader-only; an AI company gets a plain Wallet link instead of a fake trading balance.
const personaCx = usePersona()
const isTrader = computed(() => personaCx.persona.value === 'trader')

// ─── Live index ticker ───────────────────────────────────────
const indexPrice = ref(0.001005)
const indexChange = ref(0.0018)
let tickerInterval: ReturnType<typeof setInterval> | null = null

// ─── User / org session (real identity from useAuth) ─
interface Org {
  id: string
  name: string
  role: string
  type: 'personal' | 'lab' | 'enterprise'
  current: boolean
}

const auth = useAuth()

/** Display identity from the session: name from the email local-part, mode from is_paper. */
const user = computed(() => {
  const u = auth.user.value
  const email = u?.email ?? ''
  const local = email.split('@')[0] || 'account'
  const name = local.split(/[._-]/).filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(' ') || 'Account'
  return {
    name,
    email,
    initial: (name.charAt(0) || 'A').toUpperCase(),
    mode: (u?.is_paper ? 'paper' : 'capital') as 'paper' | 'capital',
  }
})

/** Current org context from the session (/me has no org display name → 'Personal' vs 'Organization'). */
const currentOrg = computed<Org>(() => {
  const u = auth.user.value
  const role = u?.roles?.[0] ?? 'member'
  return {
    id: u?.org_id || 'personal',
    name: u?.org_id ? 'Organization' : 'Personal',
    role: role.charAt(0).toUpperCase() + role.slice(1),
    type: u?.org_id ? 'enterprise' : 'personal',
    current: true,
  }
})

// ─── Menu open state ────────────────────────────────────────
const menuOpen = ref(false)
const menuRef  = ref<HTMLElement | null>(null)
const avatarRef = ref<HTMLElement | null>(null)

function toggleMenu() { menuOpen.value = !menuOpen.value }
function closeMenu()  { menuOpen.value = false }

async function signOut() {
  closeMenu()
  try { await auth.logout() } catch { /* clear session locally regardless */ }
  await navigateTo('/login')
}

function onDocMouseDown(e: MouseEvent) {
  if (!menuOpen.value) return
  const target = e.target as Node | null
  if (target && menuRef.value?.contains(target)) return
  if (target && avatarRef.value?.contains(target)) return
  closeMenu()
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && menuOpen.value) closeMenu()
}

onMounted(() => {
  document.addEventListener('mousedown', onDocMouseDown)
  document.addEventListener('keydown', onKeyDown)
  // The index ticker only drives the trader market pill — don't run it otherwise.
  if (isTrader.value) {
    tickerInterval = setInterval(() => {
      const drift = (Math.random() - 0.5) * 0.000002
      indexPrice.value = Math.max(0.0009, Math.min(0.0011, indexPrice.value + drift))
      indexChange.value = (indexPrice.value - 0.001) / 0.001
    }, 3000)
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocMouseDown)
  document.removeEventListener('keydown', onKeyDown)
  if (tickerInterval) clearInterval(tickerInterval)
})

function orgTypeLabel(t: 'personal' | 'lab' | 'enterprise'): string {
  if (t === 'personal') return 'PERSONAL'
  if (t === 'lab')      return 'LAB'
  return 'ENTERPRISE'
}
</script>

<template>
  <header class="topbar">
    <!-- Left: hamburger (mobile) + brand + market selector -->
    <div class="left">
      <button class="hamburger" aria-label="Menu" title="Menu" @click="sidebar.toggle()">
        <Menu :size="20" />
      </button>
      <NuxtLink :to="personaCx.home.value" class="brand-link">
        <BrandLogo variant="mark" size="sm" />
        <span class="brand-text">1TRADE</span>
      </NuxtLink>

      <!-- AI-Index market pill — trading only (paused exchange). Hidden for AI company / datacenter. -->
      <NuxtLink v-if="isTrader" to="/markets/eai-idx" class="market-pill">
        <span class="sym">EAI-IDX</span>
        <span class="name">AI Index</span>
        <span class="px mono">{{ formatPrice(indexPrice) }}</span>
        <BasePriceDelta :pct="indexChange" :show-sign="true" />
        <ChevronDown :size="14" class="caret" />
      </NuxtLink>
    </div>

    <!-- Right: balance + search + bell + avatar -->
    <div class="right">
      <!-- Trader: USD trading balance (paused showcase). AI company / datacenter: a plain Wallet link
           — no fake number; the real per-credit balances live on /wallet + the console strip. -->
      <NuxtLink v-if="isTrader" to="/wallet" class="balance">
        <span class="amount mono">$10,247.83</span>
        <span class="cur">USD</span>
      </NuxtLink>
      <NuxtLink v-else to="/wallet" class="balance" title="Wallet — credit balances">
        <Wallet :size="14" />
        <span class="cur">Wallet</span>
      </NuxtLink>

      <button class="icon" aria-label="Search (⌘K)" title="Search · ⌘K" @click="palette.open()">
        <Search :size="16" />
      </button>

      <button class="icon notif" aria-label="Notifications" title="Notifications" @click="notifications.open()">
        <Bell :size="16" />
        <span v-if="unreadCount > 0" class="badge">{{ unreadCount }}</span>
      </button>

      <!-- Avatar + dropdown menu -->
      <div ref="avatarRef" class="avatar-wrap">
        <button
          class="avatar"
          :class="{ active: menuOpen }"
          :aria-expanded="menuOpen"
          aria-haspopup="true"
          aria-label="Account menu"
          @click="toggleMenu"
        >
          {{ user.initial }}
        </button>

        <Transition name="menu">
          <div
            v-if="menuOpen"
            ref="menuRef"
            class="user-menu"
            role="menu"
            aria-label="User menu"
          >
            <!-- Identity -->
            <div class="um-identity">
              <span class="um-avatar">{{ user.initial }}</span>
              <div class="um-id-text">
                <div class="um-name">{{ user.name }}</div>
                <div class="um-email mono">{{ user.email }}</div>
                <div class="um-mode" :class="user.mode">
                  <span class="dot" />
                  {{ user.mode === 'paper' ? 'Paper trading' : 'Capital trading' }}
                </div>
              </div>
            </div>

            <!-- Current org context -->
            <div class="um-current">
              <Building2 :size="13" class="cur-icon" />
              <div class="cur-text">
                <span class="cur-name">{{ currentOrg.name }}</span>
                <span class="cur-role mono">{{ currentOrg.role }} · {{ orgTypeLabel(currentOrg.type) }}</span>
              </div>
            </div>


            <!-- Account shortcuts -->
            <div class="um-section">
              <NuxtLink to="/settings" class="um-row um-item" role="menuitem" @click="closeMenu">
                <UserIcon :size="14" /><span class="row-name">Profile</span>
              </NuxtLink>
              <NuxtLink to="/settings" class="um-row um-item" role="menuitem" @click="closeMenu">
                <SettingsIcon :size="14" /><span class="row-name">Settings</span>
              </NuxtLink>
              <NuxtLink to="/settings#api" class="um-row um-item" role="menuitem" @click="closeMenu">
                <Key :size="14" /><span class="row-name">API keys</span>
              </NuxtLink>
            </div>

            <!-- Help / status -->
            <div class="um-section">
              <a href="#" class="um-row um-item" role="menuitem">
                <HelpCircle :size="14" /><span class="row-name">Help &amp; docs</span>
                <ExternalLink :size="11" class="ext" />
              </a>
              <a href="#" class="um-row um-item" role="menuitem">
                <Activity :size="14" /><span class="row-name">Status</span>
                <span class="status-pill mono">
                  <span class="status-dot" /> OPERATIONAL
                </span>
              </a>
            </div>

            <!-- Sign out -->
            <button type="button" class="um-row um-signout" role="menuitem" @click="signOut">
              <LogOut :size="14" /><span class="row-name">Sign out</span>
              <span class="row-meta mono">⇧⌘Q</span>
            </button>
          </div>
        </Transition>
      </div>
    </div>
  </header>
</template>

<style scoped>
.topbar {
  grid-area: topbar;
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  height: var(--topbar-h);
  padding: 0 var(--sp-4);
  background: var(--canvas);
  border-bottom: 1px solid var(--border);
  position: relative;
  z-index: 50;
}

.left,
.right {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
}
.right { gap: var(--sp-2); }

/* Brand */
.brand-link {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  color: var(--text);
  text-decoration: none;
  font-weight: 700;
  letter-spacing: var(--ls-wide);
  font-size: var(--fs-sm);
}
.brand-text { font-family: var(--font-sans); }

/* Market selector pill */
.market-pill {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-2) var(--sp-3);
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  text-decoration: none;
  font-size: var(--fs-sm);
  transition: border-color var(--dur) var(--ease);
}
.market-pill:hover { border-color: var(--border-strong); }
.sym   { font-weight: 600; font-family: var(--font-mono); }
.name  { color: var(--text-2); }
.px    { font-weight: 500; }
.caret { color: var(--text-3); }

/* Balance pill */
.balance {
  display: inline-flex;
  align-items: baseline;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  color: var(--text);
  font-size: var(--fs-sm);
  text-decoration: none;
  transition: border-color var(--dur) var(--ease);
}
.balance:hover { border-color: var(--border-strong); }
.amount { font-weight: 600; }
.cur    { color: var(--text-3); font-size: var(--fs-xs); }

/* Icon buttons */
.icon {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-sm);
  color: var(--text-2);
  transition: background-color var(--dur) var(--ease), color var(--dur) var(--ease);
}
.icon:hover { background: var(--hover); color: var(--text); }
.notif .badge {
  position: absolute;
  top: 4px; right: 4px;
  min-width: 16px; height: 16px;
  padding: 0 4px;
  border-radius: var(--radius-full);
  background: var(--brand);
  color: var(--text-on-accent);
  font-size: 10px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
}

/* ============================================================
   Avatar + dropdown menu (M5)
   ============================================================ */
.avatar-wrap { position: relative; }

.avatar {
  display: inline-flex;
  align-items: center; justify-content: center;
  width: 36px; height: 36px;
  border-radius: var(--radius-full);
  background: var(--brand);
  color: var(--text-on-accent);
  font-weight: 700;
  font-size: var(--fs-sm);
  cursor: pointer;
  border: 2px solid transparent;
  transition: filter var(--dur) var(--ease), border-color var(--dur) var(--ease), box-shadow var(--dur) var(--ease);
}
.avatar:hover { filter: brightness(0.95); }
.avatar.active {
  border-color: var(--brand);
  box-shadow: 0 0 0 2px rgba(200, 242, 92, 0.18);
}

/* Dropdown panel */
.user-menu {
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  width: 296px;
  background: var(--overlay, var(--elevated));
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.55), 0 2px 6px rgba(0, 0, 0, 0.3);
  z-index: 1000;
  overflow: hidden;
  padding: 0;
  font-family: var(--font-sans);
}

/* Identity card */
.um-identity {
  display: grid;
  grid-template-columns: 40px 1fr;
  gap: var(--sp-3);
  padding: var(--sp-4);
  border-bottom: 1px solid var(--border);
  align-items: flex-start;
}
.um-avatar {
  width: 40px; height: 40px;
  border-radius: var(--radius-full);
  background: var(--brand);
  color: var(--text-on-accent);
  display: inline-flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 16px;
}
.um-id-text { min-width: 0; }
.um-name {
  font-weight: 600; font-size: 13.5px;
  color: var(--text); letter-spacing: -0.005em;
}
.um-email {
  font-size: 11px; color: var(--text-2);
  margin-top: 2px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.um-mode {
  display: inline-flex; align-items: center; gap: 5px;
  margin-top: 8px;
  font-family: var(--font-mono);
  font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em;
  padding: 3px 7px;
  border-radius: var(--radius-sm);
  text-transform: uppercase;
}
.um-mode.paper   { background: rgba(74, 144, 226, 0.14); color: var(--info); }
.um-mode.capital { background: var(--pos-soft); color: var(--pos); }
.um-mode .dot {
  width: 5px; height: 5px;
  border-radius: 50%;
  background: currentColor;
}

/* Current-org context bar */
.um-current {
  display: grid;
  grid-template-columns: 16px 1fr;
  gap: var(--sp-3);
  padding: 10px var(--sp-4);
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--border);
}
.cur-icon { color: var(--text-3); margin-top: 2px; }
.cur-text { display: flex; flex-direction: column; gap: 1px; min-width: 0; }
.cur-name { font-size: 12.5px; font-weight: 600; color: var(--text); }
.cur-role {
  font-size: 9.5px;
  letter-spacing: 0.12em;
  color: var(--text-3);
  text-transform: uppercase;
}

/* Sections */
.um-section {
  padding: 6px;
  border-bottom: 1px solid var(--border);
}
.um-section:last-of-type { border-bottom: none; }
.um-eyebrow {
  font-family: var(--font-mono);
  font-size: 9.5px; font-weight: 700;
  letter-spacing: 0.14em; text-transform: uppercase;
  color: var(--text-3);
  padding: 6px 10px 4px;
}

/* Row primitive */
.um-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 7px 10px;
  border-radius: var(--radius-sm);
  font-size: 12.5px;
  color: var(--text);
  text-decoration: none;
  cursor: pointer;
  background: transparent;
  border: none;
  text-align: left;
  font-family: var(--font-sans);
  transition: background-color 100ms ease, color 100ms ease;
}
.um-row:hover { background: var(--hover, rgba(255, 255, 255, 0.04)); }
.um-row .row-name { flex: 1; }
.um-row .row-meta {
  color: var(--text-3);
  font-size: 10px;
  letter-spacing: 0.06em;
}
.um-row :deep(svg) { color: var(--text-3); flex-shrink: 0; }

/* Org row variant */
.um-org .check {
  width: 14px; height: 14px;
  display: inline-flex; align-items: center; justify-content: center;
  color: var(--text-3);
}
.um-org.current .check :deep(svg) { color: var(--brand); }
.um-org.current .row-name { font-weight: 600; }
.um-org.add .row-name { color: var(--text-2); }
.um-org.add:hover .row-name { color: var(--text); }

/* External-link indicator + status pill */
.ext { color: var(--text-3); }
.status-pill {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  background: var(--pos-soft);
  color: var(--pos);
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: 0.1em;
}
.status-pill .status-dot {
  width: 5px; height: 5px;
  border-radius: 50%;
  background: currentColor;
}

/* Sign out — destructive */
.um-signout {
  width: calc(100% - 12px);
  margin: 6px;
  color: var(--text);
}
.um-signout:hover {
  background: var(--neg-soft);
  color: var(--neg);
}
.um-signout:hover :deep(svg) { color: var(--neg); }
.um-signout:hover .row-meta  { color: var(--neg); }

/* Menu enter/leave transition */
.menu-enter-active,
.menu-leave-active {
  transition: opacity 140ms var(--ease), transform 140ms var(--ease);
  transform-origin: top right;
}
.menu-enter-from {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
.menu-leave-to {
  opacity: 0;
  transform: scale(0.98) translateY(-2px);
}

/* Hamburger — drawer toggle, hidden on desktop. */
.hamburger {
  display: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  color: var(--text-2);
  cursor: pointer;
}
.hamburger:hover { color: var(--text); background: var(--hover); }

/* ── Mobile: show the hamburger, tighten spacing, drop the dense market pill. ── */
@media (max-width: 768px) {
  .hamburger { display: inline-flex; }
  .topbar { padding: 0 var(--sp-2); }
  .left, .right { gap: var(--sp-2); }
  .market-pill { display: none; }
  .brand-link { padding: var(--sp-2); }
}
@media (max-width: 420px) {
  .brand-text { display: none; } /* keep just the mark on very small screens */
}
</style>
