<script setup lang="ts">
/**
 * /states — Empty · loading · error patterns reference.
 *
 * One page that shows the six canonical state UIs in the same place
 * so designers and engineers can copy the markup into the surfaces
 * that need them. Each tile is mode-correct (dark surfaces for in-app,
 * light surfaces for 404 etc.) and self-contained.
 *
 * Not linked from the app sidebar — internal reference only.
 */

definePageMeta({ layout: false })
useHead({
  title: 'States · patterns reference — 1Trade',
  htmlAttrs: { 'data-theme': 'dark' },
})

// =====================================================
// Toast demo state
// =====================================================
const toastOpen = ref(true)
let toastTimer: ReturnType<typeof setTimeout> | null = null
function showToast() {
  if (toastTimer) clearTimeout(toastTimer)
  toastOpen.value = true
}

// =====================================================
// Banner demo state
// =====================================================
const bannerOpen = ref(true)
const reconnectAttempt = ref(2)
let reconnectTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  // Subtle reconnect counter to feel live
  reconnectTimer = setInterval(() => {
    if (bannerOpen.value) reconnectAttempt.value++
  }, 4500)
})
onBeforeUnmount(() => {
  if (reconnectTimer) clearInterval(reconnectTimer)
  if (toastTimer) clearTimeout(toastTimer)
})

function retryNow() {
  reconnectAttempt.value++
}

// =====================================================
// Order-book empty variant
// =====================================================
type BookEmpty = 'no-orders' | 'thin'
const bookVariant = ref<BookEmpty>('no-orders')
const BOOK_VARIANTS: BookEmpty[] = ['no-orders', 'thin']
</script>

<template>
  <div class="states-page" data-theme="dark">
    <!-- ============ Top chrome ============ -->
    <header class="topbar">
      <NuxtLink to="/" class="brand">
        <span class="brand-mark" />
        1TRADE
      </NuxtLink>
      <nav class="crumbs">
        <span>Design system</span>
        <span class="sep">›</span>
        <span class="strong">States · patterns reference</span>
      </nav>
      <div class="top-right">
        <span class="env-pill">INTERNAL · REFERENCE</span>
      </div>
    </header>

    <main class="page">
      <section class="page-head">
        <div>
          <div class="eyebrow"><span class="dot" /> Design system · patterns</div>
          <h1 class="page-title">Empty · loading · error states</h1>
          <p class="page-sub">
            The six patterns every screen reaches for. Each is mode-correct, action-oriented,
            and worded plainly — "Insufficient balance · $1,005 needed, $847 available" beats
            "Oops! Something went wrong." Empty states are an invitation, never a dead end.
          </p>
        </div>
        <div class="head-stats">
          <div class="hs-row"><span class="hs-k">Patterns</span><span class="hs-v">6</span></div>
          <div class="hs-row"><span class="hs-k">Modes</span><span class="hs-v">dark · light</span></div>
          <div class="hs-row"><span class="hs-k">Stability</span><span class="hs-v">v1</span></div>
        </div>
      </section>

      <!-- ============ Grid ============ -->
      <div class="grid">

        <!-- ===== 01 — EMPTY: trade history ===== -->
        <article class="tile">
          <header class="tile-head">
            <span class="tile-n">01</span>
            <h2 class="tile-title">Empty · trade history</h2>
            <span class="tile-meta">Page-level empty · dark · in-app</span>
          </header>
          <div class="tile-body surface-dark fill">
            <div class="es-empty">
              <div class="es-ic" aria-hidden="true">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="square">
                  <circle cx="12" cy="12" r="9" stroke-dasharray="2 3" />
                  <path d="M12 7v5l3 2.2" />
                </svg>
              </div>
              <h3 class="es-title">No trades yet</h3>
              <p class="es-body">
                Your trade history will appear here once you place your first order.
                Paper accounts start with $10,000 of simulated capital — try a 1,000-credit
                market buy against AI-INDEX to see the full receipt format.
              </p>
              <div class="es-actions">
                <NuxtLink to="/trade" class="btn primary">
                  Place your first trade
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                    <path d="M3 8h10M9 4l4 4-4 4" />
                  </svg>
                </NuxtLink>
                <NuxtLink to="/benchmark" class="btn ghost">Read the methodology →</NuxtLink>
              </div>
            </div>
          </div>
        </article>

        <!-- ===== 02 — LOADING: dashboard skeleton ===== -->
        <article class="tile">
          <header class="tile-head">
            <span class="tile-n">02</span>
            <h2 class="tile-title">Loading · trading dashboard</h2>
            <span class="tile-meta">Skeleton shimmer · dark · subtle pulse</span>
          </header>
          <div class="tile-body surface-dark padded">
            <!-- Skeleton in-app chrome -->
            <div class="sk-chrome">
              <div class="sk-bar mini w-28" />
              <div class="sk-bar mini w-40" />
              <div class="sk-bar mini w-16" />
              <div class="sk-bar mini w-12" />
            </div>

            <div class="sk-grid">
              <!-- Chart placeholder -->
              <div class="sk-card sk-chart">
                <div class="sk-row">
                  <div class="sk-bar w-32 tall" />
                  <div class="sk-bar w-20" />
                  <div class="sk-bar w-16" />
                </div>
                <div class="sk-chart-area">
                  <!-- Wavy candle skeleton -->
                  <div class="sk-candle h-50" />
                  <div class="sk-candle h-72" />
                  <div class="sk-candle h-58" />
                  <div class="sk-candle h-86" />
                  <div class="sk-candle h-64" />
                  <div class="sk-candle h-92" />
                  <div class="sk-candle h-70" />
                  <div class="sk-candle h-80" />
                  <div class="sk-candle h-66" />
                  <div class="sk-candle h-78" />
                  <div class="sk-candle h-54" />
                  <div class="sk-candle h-90" />
                  <div class="sk-candle h-72" />
                  <div class="sk-candle h-60" />
                  <div class="sk-candle h-84" />
                  <div class="sk-candle h-76" />
                  <div class="sk-candle h-70" />
                  <div class="sk-candle h-88" />
                </div>
              </div>

              <!-- Order book placeholder -->
              <div class="sk-card sk-book">
                <div class="sk-bar w-32 head-bar" />
                <div v-for="i in 5" :key="'ask-' + i" class="sk-row tight">
                  <div class="sk-bar w-24 neg" />
                  <div class="sk-bar w-16" />
                  <div class="sk-bar w-12" />
                </div>
                <div class="sk-mid sk-bar w-28 mid-bar" />
                <div v-for="i in 5" :key="'bid-' + i" class="sk-row tight">
                  <div class="sk-bar w-24 pos" />
                  <div class="sk-bar w-16" />
                  <div class="sk-bar w-12" />
                </div>
              </div>
            </div>

            <!-- Stats row placeholder -->
            <div class="sk-stats">
              <div v-for="i in 4" :key="'s-' + i" class="sk-stat">
                <div class="sk-bar w-20 mini" />
                <div class="sk-bar w-32 stat-num" />
              </div>
            </div>

            <div class="sk-foot mono">
              <span class="sk-pulse" />
              Streaming · 142 tok/s · waiting on first frame
            </div>
          </div>
        </article>

        <!-- ===== 03 — EMPTY: order book ===== -->
        <article class="tile">
          <header class="tile-head">
            <span class="tile-n">03</span>
            <h2 class="tile-title">Empty · order book panel</h2>
            <span class="tile-meta">
              In-panel · dark · subtle, no decoration
              <span class="tile-divider">·</span>
              <span class="seg-mini">
                <button
                  v-for="v in BOOK_VARIANTS"
                  :key="v"
                  type="button"
                  :class="{ active: bookVariant === v }"
                  @click="bookVariant = v"
                >{{ v === 'no-orders' ? 'No orders' : 'Thin book' }}</button>
              </span>
            </span>
          </header>
          <div class="tile-body surface-dark padded ob-tile-body">
            <div class="ob-panel">
              <div class="ob-head">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square" class="ob-head-ic">
                  <path d="M3 4l5 3 5-3M3 8l5 3 5-3M3 12l5 3 5-3" />
                </svg>
                <span class="ob-head-k">Order book</span>
                <span class="ob-head-badge">10 LVL</span>
              </div>
              <div class="ob-cols">
                <span>Price (USD)</span>
                <span>Size</span>
                <span>Total</span>
              </div>
              <div v-if="bookVariant === 'no-orders'" class="ob-empty">
                <p class="ob-empty-title">No orders at this depth</p>
                <p class="ob-empty-sub">
                  Last print <span class="mono">$0.001005 · 14:32:11 UTC</span> · awaiting
                  next snapshot.
                </p>
              </div>
              <div v-else class="ob-thin">
                <div class="ob-row ask thin">
                  <span class="ob-px">0.001012</span>
                  <span class="ob-qty">2,400</span>
                  <span class="ob-tot">2,400</span>
                </div>
                <div class="ob-mid">
                  <span class="mid-px mono">0.001005</span>
                  <span class="mid-spread mono">spread $0.000020 · 1.99%</span>
                </div>
                <div class="ob-row bid thin">
                  <span class="ob-px">0.000990</span>
                  <span class="ob-qty">1,800</span>
                  <span class="ob-tot">1,800</span>
                </div>
                <p class="ob-thin-note">
                  Thin book · 1 level each side. Trade with care or rest a limit at the mid.
                </p>
              </div>
            </div>
          </div>
        </article>

        <!-- ===== 04 — ERROR: order failed toast ===== -->
        <article class="tile">
          <header class="tile-head">
            <span class="tile-n">04</span>
            <h2 class="tile-title">Error · order failed (toast)</h2>
            <span class="tile-meta">
              Top-right · dark · slides in
              <span class="tile-divider">·</span>
              <button type="button" class="tile-btn" @click="showToast">Re-trigger</button>
            </span>
          </header>
          <div class="tile-body surface-dark padded toast-stage">
            <!-- Mock "page" content the toast sits over -->
            <div class="stage-mock">
              <div class="stage-row">
                <div class="stage-card">
                  <div class="stage-lbl">— Buy AI-INDEX · 1,000 cr</div>
                  <div class="stage-val">$1.005</div>
                  <div class="stage-sub">limit · GTC · slippage 0.0%</div>
                </div>
                <div class="stage-card">
                  <div class="stage-lbl">— Cash available</div>
                  <div class="stage-val neg">$847.00</div>
                  <div class="stage-sub">USD · paper account</div>
                </div>
              </div>
              <div class="stage-row tight">
                <button type="button" class="stage-btn">Buy 1,000 AI-INDEX</button>
              </div>
            </div>

            <!-- Toast itself -->
            <Transition name="toast">
              <div v-if="toastOpen" class="toast" role="alert" aria-live="assertive">
                <span class="toast-rail" />
                <span class="toast-ic">
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                    <path d="M8 2l6.5 11h-13L8 2z" />
                    <path d="M8 6.5v3.5M8 11.5v0.1" />
                  </svg>
                </span>
                <div class="toast-body">
                  <div class="toast-title">Order failed</div>
                  <p class="toast-text">
                    Insufficient balance. You need <strong>$1,005</strong> USD but have
                    <strong>$847</strong>. Add cash or reduce the order size.
                  </p>
                  <div class="toast-actions">
                    <NuxtLink to="/wallet/buy" class="toast-cta">Buy credits →</NuxtLink>
                    <button type="button" class="toast-dismiss" @click="toastOpen = false">Dismiss</button>
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </article>

        <!-- ===== 05 — ERROR: 404 ===== -->
        <article class="tile">
          <header class="tile-head">
            <span class="tile-n">05</span>
            <h2 class="tile-title">Error · 404 page</h2>
            <span class="tile-meta">Full page · light · sober</span>
          </header>
          <div class="tile-body surface-light padded">
            <div class="nf-shell">
              <div class="nf-rail">
                <span class="nf-code mono">404</span>
                <div class="nf-meta">
                  <span class="nf-meta-k">Status</span>
                  <span class="nf-meta-v mono">404 · NOT_FOUND</span>
                </div>
                <div class="nf-meta">
                  <span class="nf-meta-k">Request</span>
                  <span class="nf-meta-v mono">/markets/btc-usd</span>
                </div>
                <div class="nf-meta">
                  <span class="nf-meta-k">Trace</span>
                  <span class="nf-meta-v mono">req_9e0d11_2c4a</span>
                </div>
              </div>
              <div class="nf-content">
                <div class="nf-eyebrow">— Page not found</div>
                <h3 class="nf-title">This page doesn't exist.</h3>
                <p class="nf-text">
                  Either the URL is wrong or the market was retired. Markets that retire stay
                  reachable for 90 days under <code class="nf-code-inline">/markets/archive/</code> —
                  check there if you bookmarked a historical contract.
                </p>
                <div class="nf-actions">
                  <NuxtLink to="/trade" class="btn primary light">
                    Go to trading dashboard
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="square">
                      <path d="M3 8h10M9 4l4 4-4 4" />
                    </svg>
                  </NuxtLink>
                  <button type="button" class="btn secondary light">
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="square">
                      <circle cx="7" cy="7" r="4.5" />
                      <path d="M10.4 10.4L14 14" />
                    </svg>
                    Open command palette
                    <span class="nf-kbd"><kbd>⌘</kbd><kbd>K</kbd></span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </article>

        <!-- ===== 06 — ERROR: network lost banner ===== -->
        <article class="tile tile-wide">
          <header class="tile-head">
            <span class="tile-n">06</span>
            <h2 class="tile-title">Error · network connection lost (banner)</h2>
            <span class="tile-meta">
              Sticky top · dark or light · warning
              <span class="tile-divider">·</span>
              <button type="button" class="tile-btn" @click="bannerOpen = !bannerOpen">
                {{ bannerOpen ? 'Hide' : 'Show' }}
              </button>
            </span>
          </header>
          <div class="tile-body surface-dark padded">
            <Transition name="banner">
              <div v-if="bannerOpen" class="net-banner" role="alert">
                <span class="net-spin" aria-hidden="true">
                  <span class="spinner" />
                </span>
                <div class="net-text">
                  <span class="net-title">Connection to 1Trade lost. Reconnecting…</span>
                  <span class="net-sub mono">
                    attempt {{ reconnectAttempt }} · last heartbeat 12s ago · ws://api.exascale.com/v1/stream
                  </span>
                </div>
                <div class="net-actions">
                  <button type="button" class="net-retry" @click="retryNow">Retry now</button>
                  <button type="button" class="net-x" aria-label="Dismiss" @click="bannerOpen = false">
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="square">
                      <path d="M4 4l8 8M12 4l-8 8" />
                    </svg>
                  </button>
                </div>
              </div>
            </Transition>

            <!-- Fake page peek under the banner -->
            <div class="under-banner">
              <div class="ub-row">
                <div class="ub-card">
                  <div class="ub-k">AI-INDEX</div>
                  <div class="ub-v stale mono">$0.001005</div>
                  <div class="ub-sub mono">last good print · stale</div>
                </div>
                <div class="ub-card">
                  <div class="ub-k">Open orders</div>
                  <div class="ub-v stale mono">4</div>
                  <div class="ub-sub mono">trading frozen · resumes on reconnect</div>
                </div>
                <div class="ub-card">
                  <div class="ub-k">Subscriptions</div>
                  <div class="ub-v stale mono">3 / 5 channels</div>
                  <div class="ub-sub mono">order-book + tape disconnected</div>
                </div>
              </div>
            </div>
          </div>
        </article>
      </div>

      <!-- ============ Pattern principles ============ -->
      <section class="principles">
        <h2 class="pr-title">Principles</h2>
        <div class="pr-grid">
          <div class="pr-card">
            <div class="pr-k">— Empty</div>
            <p class="pr-p">
              Every empty state has a single, primary next action. Body text explains what would
              normally appear here, not just that it's empty.
            </p>
          </div>
          <div class="pr-card">
            <div class="pr-k">— Loading</div>
            <p class="pr-p">
              Subtle pulse animation, never jarring. Skeletons match the shape of the eventual
              content so layout doesn't shift on first frame.
            </p>
          </div>
          <div class="pr-card">
            <div class="pr-k">— Error</div>
            <p class="pr-p">
              Specific cause + the actual numbers + a remediation path. Never "Oops" or
              "Something went wrong." Always blame-free.
            </p>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.states-page {
  --t-soft: rgba(255, 255, 255, 0.05);
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.55;
  color: var(--text);
  font-feature-settings: 'ss01' on, 'tnum' on;
  background: var(--canvas);
  min-height: 100vh;
}
.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }

/* ============================================================
   Top chrome
   ============================================================ */
.topbar {
  height: 52px;
  background: var(--elevated);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 28px;
  gap: 18px;
  position: sticky;
  top: 0;
  z-index: 20;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 14px;
  letter-spacing: -0.005em;
  color: var(--text);
  text-decoration: none;
  padding-right: 18px;
  border-right: 1px solid var(--border);
  height: 32px;
}
.brand-mark {
  width: 8.89px;
  height: 20.6px;
  flex: none;
  background: var(--brand);
  -webkit-mask: url('/brand/mark.svg') center / contain no-repeat;
  mask: url('/brand/mark.svg') center / contain no-repeat;
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
.top-right { margin-left: auto; }
.env-pill {
  font-family: var(--font-mono);
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: 0.2em;
  color: var(--accent);
  background: rgba(74, 144, 226, 0.10);
  border: 1px solid rgba(74, 144, 226, 0.25);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  text-transform: uppercase;
}

/* ============================================================
   Page
   ============================================================ */
.page {
  max-width: 1320px;
  margin: 0 auto;
  padding: 32px 28px 80px;
}
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 32px;
}
.eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.eyebrow .dot { width: 5px; height: 5px; background: var(--brand); }
.page-title {
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 30px;
  letter-spacing: -0.022em;
  line-height: 1.1;
  margin: 0 0 8px;
}
.page-sub {
  color: var(--text-2);
  font-size: 14px;
  line-height: 1.6;
  margin: 0;
  max-width: 720px;
}

.head-stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 220px;
  padding: 12px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--elevated);
}
.hs-row {
  display: flex;
  justify-content: space-between;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
}
.hs-k {
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: 0.14em;
  font-size: 10px;
  font-weight: 600;
}
.hs-v { color: var(--text); }

/* ============================================================
   Grid + tile
   ============================================================ */
.grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin-bottom: 32px;
}
.tile.tile-wide { grid-column: span 2; }

.tile {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.tile-head {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.02);
}
.tile-n {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.18em;
  color: var(--text-3);
}
.tile-title {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.015em;
  margin: 0;
  color: var(--text);
}
.tile-meta {
  margin-left: auto;
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.tile-divider { color: var(--text-3); opacity: 0.6; }
.tile-btn {
  background: var(--canvas);
  border: 1px solid var(--border);
  color: var(--text-2);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.tile-btn:hover { background: var(--hover); color: var(--text); border-color: var(--border-strong); }

.seg-mini {
  display: inline-flex;
  background: var(--canvas);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.seg-mini button {
  background: transparent;
  border: 0;
  border-right: 1px solid var(--border);
  color: var(--text-3);
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 3px 8px;
  cursor: pointer;
}
.seg-mini button:last-child { border-right: 0; }
.seg-mini button:hover { color: var(--text); background: var(--hover); }
.seg-mini button.active { background: var(--text); color: var(--canvas); }

.tile-body {
  flex: 1;
  min-height: 320px;
  position: relative;
  display: flex;
  flex-direction: column;
}
.tile-body.fill { padding: 28px; }
.tile-body.padded { padding: 20px; }
.surface-dark {
  background: var(--canvas);
  color: var(--text);
}
.surface-light {
  background: #FAF8F3;
  color: #0A0A0A;
}

/* ============================================================
   01 — Empty trade history
   ============================================================ */
.es-empty {
  margin: auto;
  max-width: 420px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}
.es-ic {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  background: var(--canvas);
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--text-3);
}
.es-ic svg { width: 24px; height: 24px; }
.es-title {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text);
  margin: 4px 0 0;
}
.es-body {
  font-size: 13.5px;
  color: var(--text-2);
  margin: 0;
  line-height: 1.6;
}
.es-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
  margin-top: 6px;
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0 14px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.005em;
  cursor: pointer;
  text-decoration: none;
  transition: background 120ms, border-color 120ms, color 120ms;
}
.btn svg { width: 14px; height: 14px; }
.btn.primary {
  background: var(--brand);
  color: var(--canvas);
  border-color: var(--brand);
}
.btn.primary:hover { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn.primary.light { /* same lime works on light surfaces */ }
.btn.secondary {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
}
.btn.secondary:hover { background: var(--hover); border-color: rgba(255, 255, 255, 0.22); }
.btn.secondary.light {
  background: #FFFFFF;
  color: #0A0A0A;
  border-color: rgba(14, 14, 14, 0.18);
}
.btn.secondary.light:hover {
  background: #F7F4ED;
  border-color: rgba(14, 14, 14, 0.32);
}
.btn.ghost {
  background: transparent;
  color: var(--text-2);
}
.btn.ghost:hover { color: var(--text); }

/* ============================================================
   02 — Skeleton dashboard
   ============================================================ */
.sk-chrome {
  display: flex;
  gap: 10px;
  align-items: center;
  padding-bottom: 14px;
  border-bottom: 1px dashed var(--t-soft);
  margin-bottom: 14px;
}
.sk-bar {
  display: block;
  height: 10px;
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.05) 0%, rgba(255, 255, 255, 0.08) 50%, rgba(255, 255, 255, 0.05) 100%);
  background-size: 200% 100%;
  border-radius: 2px;
  animation: shimmer 1.6s ease-in-out infinite;
}
.sk-bar.mini { height: 8px; }
.sk-bar.tall { height: 14px; }
.sk-bar.head-bar { margin-bottom: 8px; height: 10px; }
.sk-bar.mid-bar {
  margin: 6px 0;
  height: 14px;
}
.sk-bar.stat-num { height: 18px; margin-top: 4px; }
.sk-bar.neg { background: linear-gradient(90deg, rgba(239, 68, 68, 0.10), rgba(239, 68, 68, 0.18), rgba(239, 68, 68, 0.10)); background-size: 200% 100%; }
.sk-bar.pos { background: linear-gradient(90deg, rgba(25, 195, 125, 0.10), rgba(25, 195, 125, 0.18), rgba(25, 195, 125, 0.10)); background-size: 200% 100%; }

.w-12 { width: 12px; } .w-16 { width: 16px; }
.w-20 { width: 20%; } .w-24 { width: 24%; } .w-28 { width: 28%; }
.w-32 { width: 32%; } .w-40 { width: 40%; }

@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
@keyframes pulse-soft {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

.sk-grid {
  display: grid;
  grid-template-columns: 1.6fr 1fr;
  gap: 14px;
  margin-bottom: 14px;
}
.sk-card {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 14px;
}
.sk-row {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 8px;
}
.sk-row.tight { margin-bottom: 4px; }
.sk-row.tight .sk-bar { height: 8px; }
.sk-chart-area {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 140px;
  padding-top: 10px;
}
.sk-candle {
  flex: 1;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.10), rgba(255, 255, 255, 0.04));
  border-radius: 1px;
  animation: pulse-soft 2.2s ease-in-out infinite;
}
.h-50 { height: 50%; } .h-54 { height: 54%; } .h-58 { height: 58%; }
.h-60 { height: 60%; } .h-64 { height: 64%; } .h-66 { height: 66%; }
.h-70 { height: 70%; } .h-72 { height: 72%; } .h-76 { height: 76%; }
.h-78 { height: 78%; } .h-80 { height: 80%; } .h-84 { height: 84%; }
.h-86 { height: 86%; } .h-88 { height: 88%; } .h-90 { height: 90%; }
.h-92 { height: 92%; }

.sk-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}
.sk-stat {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 10px 12px;
}
.sk-foot {
  margin-top: 14px;
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.sk-pulse {
  width: 6px;
  height: 6px;
  background: var(--brand);
  border-radius: 50%;
  animation: pulse-soft 1.6s ease-in-out infinite;
}

/* ============================================================
   03 — Empty order book
   ============================================================ */
.ob-tile-body { padding: 20px; display: grid; place-items: center; }
.ob-panel {
  width: 100%;
  max-width: 320px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.ob-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
}
.ob-head-ic { width: 12px; height: 12px; color: var(--text-3); }
.ob-head-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
}
.ob-head-badge {
  margin-left: auto;
  font-family: var(--font-mono);
  font-size: 9.5px;
  color: var(--text-3);
  background: var(--canvas);
  border: 1px solid var(--border);
  padding: 1px 6px;
  border-radius: 2px;
  letter-spacing: 0.04em;
}
.ob-cols {
  display: grid;
  grid-template-columns: 1.05fr 1fr 1fr;
  padding: 7px 12px;
  color: var(--text-3);
  font-size: 9.5px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  font-family: var(--font-mono);
  border-bottom: 1px solid var(--border);
}
.ob-cols span { text-align: right; }
.ob-cols span:first-child { text-align: left; }

.ob-empty {
  padding: 36px 18px 38px;
  text-align: center;
  color: var(--text-3);
}
.ob-empty-title {
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-2);
  margin: 0 0 4px;
  letter-spacing: -0.005em;
}
.ob-empty-sub {
  font-size: 11.5px;
  color: var(--text-3);
  margin: 0;
  line-height: 1.5;
}
.ob-empty-sub .mono {
  color: var(--text-2);
  font-size: 11px;
}

.ob-thin { font-family: var(--font-mono); font-size: 11px; font-variant-numeric: tabular-nums; }
.ob-row {
  display: grid;
  grid-template-columns: 1.05fr 1fr 1fr;
  padding: 6px 12px;
  align-items: center;
}
.ob-row.thin { opacity: 0.75; }
.ob-row > * { text-align: right; }
.ob-row .ob-px { text-align: left; }
.ob-row.ask .ob-px { color: var(--neg); }
.ob-row.bid .ob-px { color: var(--pos); }
.ob-row .ob-qty { color: var(--text); }
.ob-row .ob-tot { color: var(--text-3); }
.ob-mid {
  text-align: center;
  padding: 8px 12px;
  background: var(--elevated);
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mid-px { font-size: 13px; font-weight: 600; color: var(--text); }
.mid-spread { font-size: 10px; color: var(--text-3); letter-spacing: 0.04em; }

.ob-thin-note {
  padding: 10px 14px;
  border-top: 1px dashed var(--border);
  background: rgba(245, 158, 11, 0.04);
  font-family: var(--font-sans);
  font-size: 11.5px;
  color: var(--warn);
  margin: 0;
  letter-spacing: 0.02em;
}

/* ============================================================
   04 — Order failed toast
   ============================================================ */
.toast-stage {
  position: relative;
  overflow: hidden;
}
.stage-mock {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 24px;
  filter: blur(0.4px);
  opacity: 0.6;
}
.stage-row {
  display: flex;
  gap: 12px;
}
.stage-row.tight { gap: 8px; }
.stage-card {
  flex: 1;
  background: rgba(255, 255, 255, 0.025);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 14px;
}
.stage-lbl {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  margin-bottom: 6px;
}
.stage-val {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text);
}
.stage-val.neg { color: var(--neg); }
.stage-sub {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 4px;
}
.stage-btn {
  background: var(--pos);
  color: var(--canvas);
  border: 0;
  height: 36px;
  padding: 0 16px;
  border-radius: var(--radius-sm);
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 13px;
  letter-spacing: -0.005em;
  cursor: pointer;
}

.toast {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 360px;
  background: var(--overlay, #1A1A1A);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.4);
  display: grid;
  grid-template-columns: 3px 28px 1fr;
  gap: 12px;
  padding: 14px 16px 14px 0;
  align-items: flex-start;
}
.toast-rail {
  width: 3px;
  background: var(--neg);
  align-self: stretch;
  border-radius: 1px 0 0 1px;
  margin-right: -6px;
}
.toast-ic {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  background: rgba(239, 68, 68, 0.12);
  border-radius: var(--radius-sm);
  color: var(--neg);
  margin-top: 2px;
}
.toast-ic svg { width: 16px; height: 16px; }
.toast-body { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.toast-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.015em;
  color: var(--text);
}
.toast-text {
  font-size: 12.5px;
  color: var(--text-2);
  margin: 0;
  line-height: 1.5;
}
.toast-text strong { color: var(--text); font-weight: 600; font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
.toast-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 4px;
}
.toast-cta {
  font-family: var(--font-sans);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: -0.005em;
  color: var(--brand);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  padding-bottom: 1px;
}
.toast-cta:hover { border-bottom-color: var(--brand); }
.toast-dismiss {
  background: transparent;
  border: 0;
  font-family: var(--font-sans);
  font-size: 12px;
  color: var(--text-3);
  cursor: pointer;
  padding: 2px 4px;
}
.toast-dismiss:hover { color: var(--text); }

.toast-enter-active,
.toast-leave-active { transition: transform 240ms cubic-bezier(0.2, 0, 0, 1), opacity 200ms ease-out; }
.toast-enter-from,
.toast-leave-to { transform: translateX(120%); opacity: 0; }

/* ============================================================
   05 — 404
   ============================================================ */
.nf-shell {
  margin: auto;
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 28px;
  max-width: 720px;
  background: #FFFFFF;
  border: 1px solid rgba(14, 14, 14, 0.10);
  border-radius: var(--radius-sm);
  padding: 28px 32px;
}
.nf-rail {
  border-right: 1px solid rgba(14, 14, 14, 0.10);
  padding-right: 22px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.nf-code {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 72px;
  line-height: 1;
  letter-spacing: -0.04em;
  font-weight: 600;
  color: #0A0A0A;
  margin-bottom: 4px;
}
.nf-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.nf-meta-k {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(74, 74, 69, 0.7);
  font-weight: 600;
}
.nf-meta-v {
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: #0A0A0A;
  letter-spacing: 0.02em;
}

.nf-content { padding-top: 4px; }
.nf-eyebrow {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(74, 74, 69, 0.85);
  font-weight: 600;
  margin-bottom: 8px;
}
.nf-title {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: #0A0A0A;
  margin: 0 0 10px;
  line-height: 1.2;
}
.nf-text {
  font-size: 13.5px;
  color: rgba(74, 74, 69, 1);
  line-height: 1.6;
  margin: 0 0 18px;
}
.nf-code-inline {
  font-family: var(--font-mono);
  font-size: 12px;
  background: #F1ECE1;
  border: 1px solid rgba(14, 14, 14, 0.08);
  padding: 1px 5px;
  border-radius: 2px;
  color: #0A0A0A;
}
.nf-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.nf-kbd { display: inline-flex; gap: 2px; margin-left: 6px; }
.nf-kbd kbd {
  display: inline-grid;
  place-items: center;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  font-family: var(--font-mono);
  font-size: 10px;
  color: rgba(74, 74, 69, 1);
  background: #F7F4ED;
  border: 1px solid rgba(14, 14, 14, 0.10);
  border-bottom-width: 2px;
  border-radius: 2px;
}

/* ============================================================
   06 — Network banner
   ============================================================ */
.net-banner {
  display: grid;
  grid-template-columns: 36px 1fr auto;
  gap: 12px;
  align-items: center;
  padding: 12px 16px;
  background: rgba(245, 158, 11, 0.08);
  border: 1px solid rgba(245, 158, 11, 0.30);
  border-radius: var(--radius-sm);
  margin-bottom: 18px;
  position: relative;
}
.net-banner::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--warn);
  border-radius: 1px 0 0 1px;
}
.net-spin {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  background: rgba(245, 158, 11, 0.14);
  color: var(--warn);
  border-radius: var(--radius-sm);
}
.spinner {
  width: 14px;
  height: 14px;
  border: 1.6px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 700ms linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.net-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.net-title {
  font-family: var(--font-sans);
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.005em;
}
.net-sub {
  font-size: 11px;
  color: var(--text-2);
  letter-spacing: 0.04em;
}

.net-actions { display: inline-flex; gap: 8px; align-items: center; }
.net-retry {
  background: var(--warn);
  color: var(--canvas);
  border: 0;
  height: 28px;
  padding: 0 12px;
  border-radius: var(--radius-sm);
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 12px;
  letter-spacing: -0.005em;
  cursor: pointer;
}
.net-retry:hover { filter: brightness(1.06); }
.net-x {
  width: 26px;
  height: 26px;
  background: transparent;
  border: 1px solid rgba(245, 158, 11, 0.30);
  color: var(--warn);
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.net-x:hover { background: rgba(245, 158, 11, 0.10); }
.net-x svg { width: 12px; height: 12px; }

.banner-enter-active,
.banner-leave-active { transition: transform 200ms cubic-bezier(0.2, 0, 0, 1), opacity 200ms; }
.banner-enter-from,
.banner-leave-to { transform: translateY(-110%); opacity: 0; }

.under-banner { padding-top: 4px; }
.ub-row { display: flex; gap: 12px; }
.ub-card {
  flex: 1;
  background: rgba(255, 255, 255, 0.025);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 12px 14px;
  opacity: 0.65;
}
.ub-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  margin-bottom: 6px;
}
.ub-v {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text);
}
.ub-v.stale { color: var(--text-2); }
.ub-v.stale::after {
  content: 'STALE';
  display: inline-block;
  margin-left: 6px;
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.18em;
  color: var(--warn);
  background: rgba(245, 158, 11, 0.10);
  border: 1px solid rgba(245, 158, 11, 0.25);
  padding: 1px 5px;
  border-radius: 2px;
  vertical-align: 5px;
  font-weight: 700;
}
.ub-sub {
  font-family: var(--font-mono);
  font-size: 10.5px;
  color: var(--text-3);
  letter-spacing: 0.04em;
  margin-top: 4px;
}

/* ============================================================
   Principles
   ============================================================ */
.principles {
  padding: 24px 4px 0;
}
.pr-title {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.015em;
  margin: 0 0 14px;
  color: var(--text);
}
.pr-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
}
.pr-card {
  background: var(--elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 16px 18px;
}
.pr-k {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-3);
  font-weight: 600;
  margin-bottom: 8px;
}
.pr-p {
  font-size: 13px;
  color: var(--text-2);
  margin: 0;
  line-height: 1.55;
}

/* ============================================================
   Responsive
   ============================================================ */
@media (max-width: 1100px) {
  .grid { grid-template-columns: 1fr; }
  .tile.tile-wide { grid-column: auto; }
  .sk-grid { grid-template-columns: 1fr; }
  .sk-stats { grid-template-columns: repeat(2, 1fr); }
  .nf-shell { grid-template-columns: 1fr; }
  .nf-rail { border-right: 0; border-bottom: 1px solid rgba(14, 14, 14, 0.10); padding-right: 0; padding-bottom: 16px; }
  .pr-grid { grid-template-columns: 1fr; }
}
@media (max-width: 720px) {
  .page-head { flex-direction: column; align-items: stretch; }
  .head-stats { width: 100%; }
  .toast { right: 8px; left: 8px; width: auto; }
}
</style>
