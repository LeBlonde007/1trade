<script setup lang="ts">
/**
 * /status — public system status page (J3).
 * Light marketing surface. Public — no auth.
 */
import { CheckCircle2, AlertTriangle, XCircle, Rss, Mail, MessageSquare, ChevronDown } from 'lucide-vue-next'

definePageMeta({ layout: 'marketing' })
useHead({ title: 'System status — 1Trade' })

type Status = 'operational' | 'degraded' | 'outage' | 'maintenance'

interface Component {
  name: string
  status: Status
  uptime90d: number
  group: string
  note?: string
}

interface IncidentUpdate {
  ts: string
  status: string
  body: string
}

interface Incident {
  id: string
  title: string
  severity: 'minor' | 'major' | 'critical'
  status: 'investigating' | 'identified' | 'monitoring' | 'resolved'
  components: string[]
  startedAt: string
  resolvedAt?: string
  updates: IncidentUpdate[]
}

const components = ref<Component[]>([
  { group: 'Trading',     name: 'Matching engine',         status: 'operational', uptime90d: 99.998 },
  { group: 'Trading',     name: 'Order entry API',         status: 'operational', uptime90d: 99.991 },
  { group: 'Trading',     name: 'Market data (websocket)', status: 'operational', uptime90d: 99.987 },
  { group: 'Trading',     name: 'Settlement & clearing',   status: 'operational', uptime90d: 99.999 },
  { group: 'Account',     name: 'Wallet & ledger',         status: 'operational', uptime90d: 99.996 },
  { group: 'Account',     name: 'Identity & sign-in',      status: 'operational', uptime90d: 99.972 },
  { group: 'Account',     name: 'Card & ACH funding',      status: 'degraded',    uptime90d: 99.842, note: 'Card 3-DS provider experiencing elevated latency (~6s)' },
  { group: 'Developer',   name: 'REST API (api.exascale.com)', status: 'operational', uptime90d: 99.994 },
  { group: 'Developer',   name: 'Webhooks',                status: 'operational', uptime90d: 99.981 },
  { group: 'Developer',   name: 'Public docs',             status: 'operational', uptime90d: 99.999 },
  { group: 'Compute',     name: 'us-east-1 (Ashburn)',     status: 'operational', uptime90d: 99.995 },
  { group: 'Compute',     name: 'us-west-2 (Hillsboro)',   status: 'operational', uptime90d: 99.991 },
  { group: 'Compute',     name: 'eu-central-1 (Frankfurt)', status: 'operational', uptime90d: 99.988 },
  { group: 'Compute',     name: 'ap-south-1 (Mumbai)',     status: 'maintenance', uptime90d: 99.942, note: 'Planned hypervisor rollover 02:00–04:00 UTC' },
])

const incidents = ref<Incident[]>([
  {
    id: 'INC-2026-0517-01',
    title: 'Elevated latency on card funding',
    severity: 'minor',
    status: 'monitoring',
    components: ['Card & ACH funding'],
    startedAt: '2026-05-24T13:42:00Z',
    updates: [
      { ts: '15:18 UTC', status: 'Monitoring',    body: 'Upstream provider confirms mitigation deployed. 3-DS handshake latency back under 2s. Continuing to monitor.' },
      { ts: '14:31 UTC', status: 'Identified',    body: 'Issue isolated to our card-funding upstream provider\'s 3-DS step. Wire and ACH funding unaffected. Trading and compute fully operational.' },
      { ts: '13:55 UTC', status: 'Investigating', body: 'We are seeing elevated 3-DS verification latency (~6s vs <1s baseline) for new card deposits. Engineers engaged.' },
    ],
  },
])

const past = ref([
  { date: '2026-05-21', title: 'Order entry API degraded performance', duration: '14m', severity: 'minor' as const },
  { date: '2026-05-09', title: 'Planned maintenance — eu-central-1 hypervisor upgrade', duration: '2h 00m', severity: 'minor' as const },
  { date: '2026-04-28', title: 'Market data websocket disconnects (us-east-1)', duration: '8m',  severity: 'minor' as const },
  { date: '2026-04-12', title: 'Webhooks delivery delayed up to 90s',              duration: '47m', severity: 'major' as const },
  { date: '2026-03-30', title: 'Planned maintenance — settlement scheduled cutover', duration: '6m', severity: 'minor' as const },
])

const overallStatus = computed<Status>(() => {
  if (components.value.some(c => c.status === 'outage'))     return 'outage'
  if (components.value.some(c => c.status === 'degraded'))   return 'degraded'
  if (components.value.some(c => c.status === 'maintenance')) return 'maintenance'
  return 'operational'
})

const overallLabel = computed(() => ({
  operational: 'All systems operational',
  degraded:    'Partial service degradation',
  outage:      'Major outage in progress',
  maintenance: 'Scheduled maintenance in progress',
}[overallStatus.value]))

function statusTone(s: Status) {
  return {
    operational: 'pos',
    degraded:    'warn',
    outage:      'neg',
    maintenance: 'info',
  }[s]
}

function statusLabel(s: Status) {
  return {
    operational: 'Operational',
    degraded:    'Degraded',
    outage:      'Outage',
    maintenance: 'Maintenance',
  }[s]
}

const groups = computed(() => {
  const order = ['Trading', 'Account', 'Developer', 'Compute']
  return order.map(g => ({
    name: g,
    items: components.value.filter(c => c.group === g),
  }))
})

// 90-day uptime strip — synthetic per-day status.
// Deterministic per component+day, mostly green, occasional amber/red bar.
function dayBars(comp: Component): Status[] {
  const days: Status[] = []
  const seedBase = comp.name.split('').reduce((a, c) => a + c.charCodeAt(0), 0)
  for (let i = 0; i < 90; i++) {
    const r = ((seedBase * 9301 + (i + 1) * 49297) % 233280) / 233280
    // Bias towards operational; tail of degraded/outage scaled to component's uptime.
    const downBudget = (100 - comp.uptime90d) / 100 * 4
    if (r < downBudget * 0.15) days.push('outage')
    else if (r < downBudget) days.push('degraded')
    else days.push('operational')
  }
  // Today reflects current status.
  days[89] = comp.status
  return days
}

const subscribeOpen = ref(false)
const subscribeEmail = ref('')
const subscribed = ref(false)

function onSubscribe(e: Event) {
  e.preventDefault()
  if (!subscribeEmail.value) return
  subscribed.value = true
}

const nowLabel = computed(() => {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())} UTC`
})
</script>

<template>
  <div class="page">
    <!-- Hero / overall -->
    <section class="hero" :class="`tone-${statusTone(overallStatus)}`">
      <div class="hero-inner">
        <div class="hero-status">
          <CheckCircle2 v-if="overallStatus === 'operational'" :size="28" :stroke-width="1.6" />
          <AlertTriangle v-else-if="overallStatus === 'degraded' || overallStatus === 'maintenance'" :size="28" :stroke-width="1.6" />
          <XCircle v-else :size="28" :stroke-width="1.6" />
          <h1>{{ overallLabel }}</h1>
        </div>
        <div class="hero-meta">
          <span class="meta-label">As of</span>
          <span class="meta-value mono">{{ nowLabel }}</span>
          <span class="meta-dot">·</span>
          <span class="meta-label">Refreshes every 30s</span>
        </div>

        <div class="hero-actions">
          <button class="btn-primary" @click="subscribeOpen = !subscribeOpen">
            <Mail :size="14" :stroke-width="1.7" /> Subscribe to updates
          </button>
          <a class="btn-ghost" href="/status/rss.xml"><Rss :size="14" :stroke-width="1.7" /> RSS</a>
          <a class="btn-ghost" href="https://status.exascale.com/sms"><MessageSquare :size="14" :stroke-width="1.7" /> SMS</a>
        </div>

        <Transition name="reveal">
          <form v-if="subscribeOpen && !subscribed" class="subscribe-form" @submit="onSubscribe">
            <input
              v-model="subscribeEmail"
              type="email"
              required
              placeholder="you@company.com"
              autocomplete="email"
            >
            <select>
              <option>All components</option>
              <option>Trading only</option>
              <option>Compute only</option>
              <option>Developer / API only</option>
            </select>
            <button type="submit">Subscribe</button>
          </form>
          <div v-else-if="subscribed" class="subscribed-note">
            Confirmation sent to <span class="mono">{{ subscribeEmail }}</span>. Click the link to activate.
          </div>
        </Transition>
      </div>
    </section>

    <div class="container">
      <!-- Active incidents -->
      <section v-if="incidents.length" class="incidents">
        <header class="section-head">
          <h2>Active incidents</h2>
          <span class="count">{{ incidents.length }}</span>
        </header>

        <article v-for="inc in incidents" :key="inc.id" class="incident" :class="`sev-${inc.severity}`">
          <div class="inc-head">
            <div class="inc-title">
              <span class="sev-chip">{{ inc.severity }}</span>
              <h3>{{ inc.title }}</h3>
            </div>
            <div class="inc-meta">
              <span class="status-pill">{{ inc.status }}</span>
              <span class="mono inc-id">{{ inc.id }}</span>
            </div>
          </div>

          <div class="inc-affects">
            <span class="caps">Affects</span>
            <span v-for="c in inc.components" :key="c" class="comp-chip">{{ c }}</span>
          </div>

          <ol class="updates">
            <li v-for="(u, i) in inc.updates" :key="i" :class="{ latest: i === 0 }">
              <div class="upd-rail" />
              <div class="upd-body">
                <div class="upd-head">
                  <span class="upd-status">{{ u.status }}</span>
                  <span class="mono upd-ts">{{ u.ts }}</span>
                </div>
                <p>{{ u.body }}</p>
              </div>
            </li>
          </ol>
        </article>
      </section>

      <!-- Components -->
      <section class="components">
        <header class="section-head">
          <h2>Components</h2>
          <span class="caps subtitle">Last 90 days uptime</span>
        </header>

        <div v-for="g in groups" :key="g.name" class="group">
          <h3 class="group-name">{{ g.name }}</h3>
          <ul class="comp-list">
            <li v-for="c in g.items" :key="c.name" class="comp-row">
              <div class="comp-left">
                <span class="dot" :class="`tone-${statusTone(c.status)}`" />
                <div>
                  <div class="comp-name">{{ c.name }}</div>
                  <div v-if="c.note" class="comp-note">{{ c.note }}</div>
                </div>
              </div>

              <div class="comp-strip">
                <span
                  v-for="(d, i) in dayBars(c)"
                  :key="i"
                  class="bar"
                  :class="`tone-${statusTone(d)}`"
                  :title="`Day -${89 - i}: ${statusLabel(d)}`"
                />
              </div>

              <div class="comp-right">
                <span class="mono uptime">{{ c.uptime90d.toFixed(3) }}%</span>
                <span class="status-tag" :class="`tone-${statusTone(c.status)}`">{{ statusLabel(c.status) }}</span>
              </div>
            </li>
          </ul>
        </div>
      </section>

      <!-- Past incidents -->
      <section class="history">
        <header class="section-head">
          <h2>Past incidents</h2>
          <a href="/status/history" class="see-all">See all <ChevronDown :size="14" :stroke-width="1.7" style="transform:rotate(-90deg)" /></a>
        </header>

        <ul class="hist-list">
          <li v-for="p in past" :key="p.date">
            <span class="mono hist-date">{{ p.date }}</span>
            <span class="hist-title">{{ p.title }}</span>
            <span class="hist-meta">
              <span class="hist-sev" :class="`sev-${p.severity}`">{{ p.severity }}</span>
              <span class="mono">{{ p.duration }}</span>
            </span>
          </li>
        </ul>
      </section>

      <footer class="page-foot">
        <span>All times UTC.</span>
        <span class="dot-sep">·</span>
        <a href="/trust">Security & compliance</a>
        <span class="dot-sep">·</span>
        <a href="/docs/api/status">Programmatic status (JSON)</a>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.page {
  background: var(--surface-canvas, #F7F4ED);
  color: var(--text-primary, #0A0A0A);
  min-height: 100vh;
  font-family: 'Inter', system-ui, sans-serif;
  font-feature-settings: 'tnum';
}

.mono { font-family: 'JetBrains Mono', ui-monospace, monospace; font-variant-numeric: tabular-nums; }

.caps {
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-weight: 600;
  color: var(--text-tertiary, #A8A196);
}

/* Hero ---------------------------------------------- */
.hero {
  border-bottom: 1px solid rgba(0,0,0,0.08);
  background: #FFFFFF;
}
.hero.tone-pos  { box-shadow: inset 0 2px 0 #19C37D; }
.hero.tone-warn { box-shadow: inset 0 2px 0 #F5A524; }
.hero.tone-neg  { box-shadow: inset 0 2px 0 #EF4444; }
.hero.tone-info { box-shadow: inset 0 2px 0 var(--info); }

.hero-inner {
  max-width: 1080px;
  margin: 0 auto;
  padding: 48px 24px 32px;
}
.hero-status {
  display: flex;
  align-items: center;
  gap: 12px;
}
.tone-pos  .hero-status > svg:first-child { color: #19C37D; }
.tone-warn .hero-status > svg:first-child { color: #F5A524; }
.tone-neg  .hero-status > svg:first-child { color: #EF4444; }
.tone-info .hero-status > svg:first-child { color: var(--info); }

.hero-status h1 {
  font-size: 32px;
  font-weight: 600;
  letter-spacing: -0.01em;
  margin: 0;
}

.hero-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 13px;
  color: var(--text-secondary, #7E786C);
}
.meta-dot { color: var(--text-tertiary, #A8A196); }
.meta-value { color: var(--text-primary); }

.hero-actions {
  display: flex;
  gap: 8px;
  margin-top: 24px;
  flex-wrap: wrap;
}
.btn-primary,
.btn-ghost {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  font-size: 13px;
  font-weight: 500;
  border-radius: 4px;
  text-decoration: none;
  cursor: pointer;
  border: 1px solid rgba(0,0,0,0.08);
  background: #FFFFFF;
  color: var(--text-primary);
  transition: background 100ms ease, border-color 100ms ease;
}
.btn-primary {
  background: var(--text-primary);
  color: #F7F4ED;
  border-color: var(--text-primary);
}
.btn-primary:hover { background: #1A1A1A; }
.btn-ghost:hover   { border-color: rgba(0,0,0,0.16); }

/* Subscribe form */
.subscribe-form {
  display: flex;
  gap: 8px;
  margin-top: 16px;
  max-width: 560px;
}
.subscribe-form input,
.subscribe-form select,
.subscribe-form button {
  height: 36px;
  font-size: 13px;
  border-radius: 4px;
  border: 1px solid rgba(0,0,0,0.16);
  background: #FFFFFF;
  padding: 0 12px;
  color: var(--text-primary);
  font-family: inherit;
}
.subscribe-form input { flex: 1; }
.subscribe-form input:focus,
.subscribe-form select:focus {
  outline: none;
  border-color: var(--info);
  box-shadow: 0 0 0 3px rgba(74,144,226,0.15);
}
.subscribe-form button {
  background: var(--text-primary);
  color: #F7F4ED;
  border-color: var(--text-primary);
  cursor: pointer;
  font-weight: 500;
  padding: 0 16px;
}
.subscribed-note {
  margin-top: 16px;
  font-size: 13px;
  color: var(--text-secondary);
}

.reveal-enter-from, .reveal-leave-to { opacity: 0; transform: translateY(-4px); }
.reveal-enter-active, .reveal-leave-active { transition: opacity 160ms ease, transform 160ms ease; }

/* Container ------------------------------------------ */
.container {
  max-width: 1080px;
  margin: 0 auto;
  padding: 32px 24px 96px;
}

.section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin: 32px 0 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(0,0,0,0.08);
}
.section-head:first-child { margin-top: 0; }
.section-head h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.section-head .count {
  display: inline-block;
  margin-left: 8px;
  padding: 1px 8px;
  font-size: 11px;
  font-weight: 600;
  background: rgba(239,68,68,0.12);
  color: #EF4444;
  border-radius: 999px;
}
.subtitle { color: var(--text-tertiary); }
.see-all {
  font-size: 12px;
  color: var(--text-secondary);
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.see-all:hover { color: var(--text-primary); }

/* Incidents ------------------------------------------ */
.incident {
  background: #FFFFFF;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 6px;
  padding: 16px 20px;
  margin-bottom: 12px;
}
.incident.sev-minor    { border-left: 3px solid #F5A524; }
.incident.sev-major    { border-left: 3px solid #EF4444; }
.incident.sev-critical { border-left: 3px solid #EF4444; background: rgba(239,68,68,0.04); }

.inc-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}
.inc-title {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.inc-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.sev-chip {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 2px 6px;
  border-radius: 2px;
  background: rgba(245,158,11,0.12);
  color: #A16207;
}
.sev-major   .sev-chip { background: rgba(239,68,68,0.12); color: #B91C1C; }
.sev-critical .sev-chip { background: #EF4444; color: #FFF; }

.inc-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.status-pill {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-weight: 600;
  padding: 2px 8px;
  border: 1px solid rgba(0,0,0,0.16);
  border-radius: 2px;
}
.inc-id { font-size: 11px; color: var(--text-tertiary); }

.inc-affects {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 12px 0 16px;
  flex-wrap: wrap;
}
.comp-chip {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 2px;
  background: #F0EFEC;
  color: var(--text-secondary);
}

.updates {
  list-style: none;
  padding: 0;
  margin: 0;
  border-top: 1px solid rgba(0,0,0,0.08);
}
.updates li {
  position: relative;
  display: flex;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px dashed rgba(0,0,0,0.06);
}
.updates li:last-child { border-bottom: none; }
.upd-rail {
  width: 8px;
  flex-shrink: 0;
  position: relative;
}
.upd-rail::before {
  content: '';
  position: absolute;
  left: 3px;
  top: 8px;
  width: 2px;
  height: 2px;
  border-radius: 50%;
  background: var(--text-tertiary);
}
.updates li.latest .upd-rail::before {
  width: 8px;
  height: 8px;
  left: 0;
  top: 8px;
  background: #19C37D;
  box-shadow: 0 0 0 3px rgba(25,195,125,0.18);
}
.upd-body { flex: 1; }
.upd-head {
  display: flex;
  gap: 8px;
  align-items: baseline;
  margin-bottom: 2px;
}
.upd-status {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-primary);
}
.upd-ts {
  font-size: 11px;
  color: var(--text-tertiary);
}
.upd-body p {
  margin: 0;
  font-size: 14px;
  line-height: 1.55;
  color: var(--text-secondary);
}

/* Components ----------------------------------------- */
.group { margin-bottom: 24px; }
.group-name {
  margin: 0 0 8px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}
.comp-list {
  list-style: none;
  padding: 0;
  margin: 0;
  background: #FFFFFF;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 6px;
}
.comp-row {
  display: grid;
  grid-template-columns: 1fr 360px 180px;
  align-items: center;
  gap: 16px;
  padding: 14px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.06);
}
.comp-row:last-child { border-bottom: none; }

.comp-left {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  min-width: 0;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 6px;
  flex-shrink: 0;
}
.dot.tone-pos  { background: #19C37D; box-shadow: 0 0 0 3px rgba(25,195,125,0.18); }
.dot.tone-warn { background: #F5A524; box-shadow: 0 0 0 3px rgba(245,158,11,0.18); }
.dot.tone-neg  { background: #EF4444; box-shadow: 0 0 0 3px rgba(239,68,68,0.18); }
.dot.tone-info { background: var(--info); box-shadow: 0 0 0 3px rgba(74,144,226,0.18); }

.comp-name {
  font-size: 14px;
  font-weight: 500;
}
.comp-note {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
}

.comp-strip {
  display: flex;
  gap: 1px;
  height: 28px;
  overflow: hidden;
}
.comp-strip .bar {
  flex: 1;
  border-radius: 1px;
  min-width: 2px;
}
.bar.tone-pos  { background: #19C37D; opacity: 0.85; }
.bar.tone-warn { background: #F5A524; }
.bar.tone-neg  { background: #EF4444; }
.bar.tone-info { background: var(--info); }
.comp-strip .bar:hover { opacity: 1; transform: scaleY(1.08); }

.comp-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.uptime {
  font-size: 13px;
  color: var(--text-primary);
}
.status-tag {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 2px;
}
.status-tag.tone-pos  { background: rgba(25,195,125,0.12); color: #128050; }
.status-tag.tone-warn { background: rgba(245,158,11,0.12); color: #A16207; }
.status-tag.tone-neg  { background: rgba(239,68,68,0.12);  color: #B91C1C; }
.status-tag.tone-info { background: rgba(74,144,226,0.12); color: #2A5FA6; }

@media (max-width: 900px) {
  .comp-row { grid-template-columns: 1fr; }
  .comp-strip { height: 16px; }
}

/* History -------------------------------------------- */
.hist-list {
  list-style: none;
  padding: 0;
  margin: 0;
  background: #FFFFFF;
  border: 1px solid rgba(0,0,0,0.08);
  border-radius: 6px;
}
.hist-list li {
  display: grid;
  grid-template-columns: 110px 1fr auto;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.06);
  font-size: 13px;
}
.hist-list li:last-child { border-bottom: none; }
.hist-date  { color: var(--text-tertiary); font-size: 12px; }
.hist-title { color: var(--text-primary); }
.hist-meta  { display: flex; align-items: center; gap: 12px; color: var(--text-secondary); font-size: 12px; }
.hist-sev {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 2px 6px;
  border-radius: 2px;
}
.hist-sev.sev-minor { background: rgba(245,158,11,0.12); color: #A16207; }
.hist-sev.sev-major { background: rgba(239,68,68,0.12); color: #B91C1C; }

/* Footer --------------------------------------------- */
.page-foot {
  margin-top: 32px;
  padding-top: 16px;
  border-top: 1px solid rgba(0,0,0,0.08);
  font-size: 12px;
  color: var(--text-tertiary);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.page-foot a { color: var(--text-secondary); text-decoration: none; }
.page-foot a:hover { color: var(--text-primary); }
.dot-sep { color: var(--text-tertiary); }
</style>
