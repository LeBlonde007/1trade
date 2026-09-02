/**
 * tour-scripts.ts — persona-driven guided tours.
 *
 * Flow logic mirrors /mnt/f/ex/docs/exascale_persona_demo_flows.md.
 * Each step describes the cards + actions on the screen the user is on.
 * Bodies are author-controlled HTML (rendered via DOMParser in the runner).
 */

export type PersonaId = 'trader' | 'enterprise' | 'datacenter' | 'lab'

export interface Persona {
  id: PersonaId
  name: string
  pitch: string
  blurb: string
  glyph: string
  estimate: string
}

export interface TourStep {
  id: string
  route: string
  section: string
  element?: string
  title: string
  body: string
  side?: 'top' | 'right' | 'bottom' | 'left' | 'over'
  askFeedback?: boolean
}

export const personas: Persona[] = [
  {
    id: 'trader',
    name: 'Jordan Park — Independent quant',
    pitch: 'Trade AI compute the way you used to trade FX.',
    blurb: 'Marketing → signup → KYC → trading floor → portfolio → wallet. Two minutes from landing to live order.',
    glyph: '↗',
    estimate: '~4 min · 10 stops',
  },
  {
    id: 'enterprise',
    name: 'Maya Chen — Enterprise buyer',
    pitch: 'Procure compute at scale. Budgets, seats, SSO, audit.',
    blurb: 'Sales-assisted activation, team setup, compute provisioning, inference, API keys, governance.',
    glyph: '⌘',
    estimate: '~5 min · 10 stops',
  },
  {
    id: 'datacenter',
    name: 'Tom Reyes — Datacenter partner',
    pitch: 'Sell GPU-hours into the venue.',
    blurb: 'Partner portal: capacity grid, fill rate, payouts. Register a new site end-to-end.',
    glyph: '◫',
    estimate: '~3 min · 7 stops',
  },
  {
    id: 'lab',
    name: 'AI Lab / Researcher (bonus)',
    pitch: 'Compute + inference deep dive.',
    blurb: 'Provision an H100 box, tail logs, browse the model catalog, watch cost in real time.',
    glyph: '⌗',
    estimate: '~2 min · 7 stops',
  },
]

export const DEMO_INSTANCE_ID = 'inst_8c4f2a1e'

// ────────────────────────────────────────────────────────────────────────────
// Persona 1 — Jordan Park
// ────────────────────────────────────────────────────────────────────────────
const traderTour: TourStep[] = [
  {
    id: 'trader.intro',
    route: '/',
    section: 'Marketing landing',
    title: 'Meet Jordan — independent quant',
    body: `
      <p>Jordan trades AI compute the way they used to trade FX. Two minutes from landing on the homepage to a live order on the floor.</p>
      <p>This tour walks the full flow: <strong>marketing → signup → KYC → trade → portfolio → wallet → benchmark</strong>. About 4 minutes, ten stops.</p>
    `,
    side: 'over',
    askFeedback: false,
  },
  {
    id: 'trader.home.hero',
    route: '/',
    section: 'Homepage hero',
    element: '.hero, h1, [data-tour="hero"]',
    title: 'Live AI Index in the chrome',
    body: `
      <p>The marketing site is information-dense, not flashy. The hero pitches the thesis — <em>"one credit equals one dollar of work"</em> — and the footer status bar runs a live AI Index ticker that auto-updates every 5 s. This signals <strong>real market</strong> from the first frame.</p>
      <p>Scroll path: hero → markets grid → AI Index card → products row → audiences split → final CTA.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'trader.home.markets',
    route: '/',
    section: 'Markets preview & CTA',
    element: '.markets, .market-grid, section',
    title: 'Markets preview & "Open account"',
    body: `
      <p>Above the fold below the hero, a compact markets grid shows EAI-IDX, TEXT-SPOT, IMAGE-SPOT, GPU-SPOT with live prices. Right-hand call-out: the index methodology link to <a href="/benchmark">/benchmark</a>.</p>
      <p><strong>Primary action:</strong> <em>Open account</em> top-right (and again in the hero). One click into the funnel.</p>
    `,
    side: 'top',
    askFeedback: true,
  },
  {
    id: 'trader.signup',
    route: '/signup',
    section: 'Signup form',
    element: 'form, h1, .signup',
    title: 'Signup — email + password',
    body: `
      <p>Single-column, light theme, no marketing chrome. Fields:</p>
      <ul>
        <li><strong>Email</strong> — used for login + audit notifications.</li>
        <li><strong>Password</strong> — strength meter updates as you type (entropy bar).</li>
        <li><strong>Terms checkbox</strong> — links to MSA + privacy policy.</li>
      </ul>
      <p>"Continue" hits <code>/onboarding/kyc</code>. Note: an email-verify interstitial belongs here (D2 in the roadmap).</p>
    `,
    side: 'right',
    askFeedback: true,
  },
  {
    id: 'trader.kyc',
    route: '/onboarding/kyc',
    section: 'KYC funnel',
    element: '.kyc, form, h1',
    title: 'KYC — 4 steps, light touch',
    body: `
      <p>Light-touch because users trade <strong>credits</strong> (commodities), not securities. Four steps with a sticky progress rail:</p>
      <ol>
        <li><strong>Name + DOB + country</strong> — country combobox is a real autocomplete.</li>
        <li><strong>Address</strong> — auto-fills via postcode.</li>
        <li><strong>Tax residency + investor type</strong> — radio set.</li>
        <li><strong>Review &amp; submit</strong> — green-check success state → <em>"You're cleared to trade."</em></li>
      </ol>
    `,
    side: 'right',
    askFeedback: true,
  },
  {
    id: 'trader.trade.chrome',
    route: '/trade',
    section: 'Trading dashboard — chrome',
    element: '.topbar, [data-tour="topbar"], header',
    title: 'App chrome — topbar + sidebar',
    body: `
      <p>The dark trading shell. Two persistent surfaces wrap every in-app page:</p>
      <ul>
        <li><strong>Topbar</strong>: instrument selector (EAI-IDX default), live mid-price, USD balance pill, 🔔 bell with unread badge, avatar.</li>
        <li><strong>Sidebar</strong>: 8 chord-navigable icons — <kbd>G T</kbd> Trade · <kbd>G M</kbd> Markets · <kbd>G I</kbd> Index · <kbd>G P</kbd> Portfolio · <kbd>G H</kbd> History · <kbd>G W</kbd> Wallet · Compute · Inference.</li>
      </ul>
      <p>Press <kbd>⌘ K</kbd> any time for the command palette.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'trader.trade.grid',
    route: '/trade',
    section: 'Trading dashboard — grid',
    element: '.tv-chart, .lwc, [data-tour="chart"], main',
    title: 'Chart · book · tape · form · positions',
    body: `
      <p>Bloomberg-density grid:</p>
      <ul>
        <li><strong>Chart</strong> (left ⅔): TradingView-style candles + volume. Timeframe tabs 1m / 5m / 15m / 1h / 1d. Crosshair shows OHLC + the tape print at that bar.</li>
        <li><strong>Order book</strong> (top-right): 10-level L2 depth, cumulative size bars, aggregation toggle 1× / 5× / 10× ticks.</li>
        <li><strong>Tape</strong> (mid-right): time-and-sales rolling. Click a print to seed the order form size.</li>
        <li><strong>Order form</strong> (bottom-right): market / limit / stop, size in credits or USD-notional, margin + buying power inline. Hotkeys <kbd>B</kbd> / <kbd>S</kbd> flip side.</li>
        <li><strong>Positions strip</strong> (bottom): live P&amp;L for every open position. Row click opens the position drawer.</li>
      </ul>
    `,
    side: 'top',
    askFeedback: true,
  },
  {
    id: 'trader.portfolio',
    route: '/portfolio',
    section: 'Portfolio',
    element: 'h1, .page-head, main',
    title: 'Portfolio — performance vs the index',
    body: `
      <p>Performance attribution is built in — you always know whether you're <em>beating beta</em> or just riding it.</p>
      <ul>
        <li><strong>Hero KPI card</strong>: total P&amp;L, today's delta, benchmark-relative bar vs AI Index.</li>
        <li><strong>Performance chart</strong>: range tabs <kbd>1D</kbd> · <kbd>1W</kbd> · <kbd>1M</kbd> · <kbd>3M</kbd> · <kbd>YTD</kbd> · <kbd>ALL</kbd>. Dashed AI-INDEX overlay toggleable.</li>
        <li><strong>Positions table</strong>: sortable, 5 rows in mock; row → position drawer (close / reduce / convert-to-limit / set stop).</li>
        <li><strong>Allocation donut</strong> and <strong>asset-class bar</strong> on the right rail.</li>
        <li><strong>Recent activity feed</strong>: fills, alerts, deposits, in reverse chrono.</li>
      </ul>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'trader.notifications',
    route: '/portfolio',
    section: 'Notifications drawer',
    element: '.topbar, [data-tour="topbar"], header',
    title: 'Bell → notifications drawer',
    body: `
      <p>Click the 🔔 bell in the topbar. 400 px right drawer slides in with the unread feed:</p>
      <ul>
        <li>Order filled · Price alert · Budget alert · API key created · Maintenance scheduled · Welcome.</li>
        <li>Filter chips: <em>All / Unread / Trades / Account / Alerts</em>.</li>
        <li><em>Mark all as read</em> clears the topbar badge.</li>
      </ul>
      <p>Esc closes. Everything that <em>could</em> have been email lands here instead.</p>
    `,
    side: 'left',
    askFeedback: true,
  },
  {
    id: 'trader.wallet',
    route: '/wallet',
    section: 'Wallet',
    element: 'h1, .page-head, main',
    title: 'Wallet — cash on one side, credits on the other',
    body: `
      <p>Three balance cards at the top:</p>
      <ul>
        <li><strong>Cash (USD)</strong> — withdrawable to a verified bank.</li>
        <li><strong>AI-IDX credits</strong> — the index instrument.</li>
        <li><strong>Sub-credits</strong> — text, image, video, GPU-hours, broken out.</li>
      </ul>
      <p>Below: ledger of recent in/out movements. Primary action: <strong>Buy credits</strong>. Secondary: <em>Convert</em> between any two credit families at the venue rate.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'trader.wallet.buy',
    route: '/wallet/buy',
    section: 'Buy credits funnel',
    element: 'form, h1, main',
    title: 'Buy credits — 3-step funnel',
    body: `
      <ol>
        <li><strong>Amount</strong> — USD or credits, live FX shown.</li>
        <li><strong>Payment method</strong> — card (instant), wire (T+1), ACH (T+3). Tax/fee preview line-itemed.</li>
        <li><strong>Confirm</strong> — receipt panel updates in real time; <em>Save &amp; exit</em> returns to <code>/wallet</code>.</li>
      </ol>
      <p>Card buys settle to your trading account in &lt; 60 s. No idle balance — credits start earning the moment they arrive.</p>
    `,
    side: 'right',
    askFeedback: true,
  },
  {
    id: 'trader.benchmark',
    route: '/benchmark',
    section: 'Methodology whitepaper',
    element: 'h1, .toc, main',
    title: 'AI-Index methodology — published artifact',
    body: `
      <p>Every constituent, weight, and audit pointer is public. Open in a fresh tab on demos.</p>
      <ul>
        <li><strong>Sticky TOC</strong> left rail: definitions → weighting → rebalance → audit.</li>
        <li><strong>Formula block</strong> with monospace formatted equations.</li>
        <li><strong>Historical chart</strong> with rebase-to-100 toggle.</li>
        <li><strong>Audit pointer</strong>: signed daily hashes published to a public Merkle log.</li>
      </ul>
    `,
    side: 'bottom',
    askFeedback: true,
  },
]

// ────────────────────────────────────────────────────────────────────────────
// Persona 2 — Maya Chen
// ────────────────────────────────────────────────────────────────────────────
const enterpriseTour: TourStep[] = [
  {
    id: 'ent.intro',
    route: '/',
    section: 'Marketing landing',
    title: 'Meet Maya — VP Engineering',
    body: `
      <p>Maya procures compute at scale. Cares about <strong>budget control · team seats · audit · uptime</strong> — not chart patterns.</p>
      <p>This is the sales-assisted activation flow: she lands on marketing, hands off to a CSM, then runs the activation checklist herself. About 5 minutes, ten stops.</p>
    `,
    side: 'over',
    askFeedback: false,
  },
  {
    id: 'ent.home.audience',
    route: '/',
    section: 'Audience split section',
    element: '.audiences, section',
    title: '"For AI companies" pitch',
    body: `
      <p>Scroll to the three-column audience split: <strong>For traders · For AI companies · For datacenters</strong>. The middle column is Maya's.</p>
      <p>The <em>Talk to enterprise sales →</em> link triggers a CSM-assist flow; in this demo we'll deep-link straight to the activation page.</p>
    `,
    side: 'top',
    askFeedback: true,
  },
  {
    id: 'ent.teams',
    route: '/enterprise/teams',
    section: 'Teams & sub-accounts',
    element: 'h1, .page-head, table',
    title: 'Teams, budgets, scopes',
    body: `
      <p>Finance keeps the lid on, engineering still ships.</p>
      <ul>
        <li><strong>Member roster</strong>: 8 mock seats with role chips — Owner · Admin · Trader · Viewer.</li>
        <li><strong>Per-member budget bars</strong>: month-to-date spend vs cap, color-graded.</li>
        <li><strong>Org spend tile</strong>: aggregate burn + projection.</li>
        <li><strong>Invite member</strong> modal: email + role + per-seat budget + scope checkboxes (Trading / Compute / Inference / Admin).</li>
      </ul>
      <p>Click a row → sub-account detail page (members, attached resources, audit slice).</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'ent.sso',
    route: '/enterprise/sso',
    section: 'SSO & SAML wizard',
    element: 'h1, .page-head, main',
    title: 'Federate sign-in — 3-step wizard',
    body: `
      <p>Maya's IT lead handles this. Three steps with a sticky stepper:</p>
      <ol>
        <li><strong>Pick IdP</strong> — Okta · Microsoft Entra ID · Google Workspace · custom SAML 2.0.</li>
        <li><strong>Exchange metadata</strong> — download 1Trade SP XML, paste IdP metadata URL or XML.</li>
        <li><strong>Test handshake</strong> — real sign-in against a sandbox user, attribute-mapping preview, signature verification.</li>
      </ol>
      <p>On enable: break-glass recovery code generated, SCIM provisioning toggle, nightly sync indicator.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'ent.compute',
    route: '/compute',
    section: 'Compute instances list',
    element: 'h1, .page-head, table',
    title: 'Every instance in one view',
    body: `
      <ul>
        <li><strong>8 instances</strong> across US-EAST · US-WEST · EU-CENTRAL · AP-SOUTH. Status column: <em>RUNNING · PROVISIONING · STOPPED</em>.</li>
        <li><strong>Per-row utilization</strong>: 8 GPU mini-bars showing live util per device.</li>
        <li><strong>Org spend tile</strong> + <strong>headroom tile</strong> in chrome.</li>
        <li><strong>+ New instance</strong> primary CTA.</li>
        <li>Row click → instance detail (overview · logs · snapshots · events · cost).</li>
      </ul>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'ent.compute.new',
    route: '/compute/new',
    section: 'Provision new instance',
    element: 'form, h1, main',
    title: 'Launch wizard with live price calculator',
    body: `
      <p>Linear funnel. Every change updates the price tile in the top-right.</p>
      <ul>
        <li><strong>GPU type</strong> — H100 SXM5 80GB · H200 141GB</li>
        <li><strong>Count</strong> — 1 · 2 · 4 · 8 (NVLink-bonded above 1).</li>
        <li><strong>Region</strong> — US-EAST · US-WEST · EU-CENTRAL · AP-SOUTH.</li>
        <li><strong>Image</strong> — CUDA + PyTorch / vLLM / Triton / custom.</li>
        <li><strong>SSH key</strong> — pulled from <code>/settings#ssh</code>.</li>
        <li><strong>Budget cap</strong> — hard $ cap, instance auto-stops on breach.</li>
      </ul>
      <p>Sidebar tip: <em>"Reserve capacity via the index for ~12 % lower rate."</em></p>
    `,
    side: 'right',
    askFeedback: true,
  },
  {
    id: 'ent.inference',
    route: '/inference',
    section: 'Inference playground',
    element: 'h1, .page-head, main',
    title: 'Catalog + playground + code switcher',
    body: `
      <ul>
        <li><strong>Catalog</strong> (left rail): 10 models tagged by family — text · speech · image · video · embed. Each card shows $/1M-tok, p50/p99 latency, throughput.</li>
        <li><strong>Playground</strong> (center): system prompt + user prompt + streaming output with a typewriter cursor.</li>
        <li><strong>Metadata sidebar</strong> (right): tokens, cost, latency, full model card.</li>
        <li><strong>Top tabs</strong>: <em>Playground · Code</em>. Code tab shows Python / cURL / JS hitting the same endpoint with a real API key.</li>
      </ul>
      <p>Same credit, two doors. Buy on the trading floor or burn through the API — the meter is identical.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'ent.settings.api',
    route: '/settings',
    section: 'Settings → API keys',
    element: 'h1, .page-head, main',
    title: 'API keys — scoped, expiry-bound, audited',
    body: `
      <ul>
        <li><strong>Keys table</strong>: 5 mock keys with color-coded scopes (Trade · Compute · Inference · Read-only).</li>
        <li><strong>+ Create key</strong> → modal: name · scope chips · expiry (1d / 7d / 30d / never) · per-key budget cap.</li>
        <li><strong>Reveal-once flow</strong>: generated state shows the <code>sk_live_…</code> string with a copy button; closing the modal masks it permanently.</li>
        <li><strong>Audit pointer</strong> per key: last-used IP, last-used route.</li>
      </ul>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'ent.audit',
    route: '/enterprise/audit',
    section: 'Audit log',
    element: 'h1, .page-head, main',
    title: 'Audit log — the enterprise pitch',
    body: `
      <p>Every privileged action. Compliance is <em>the</em> enterprise pitch.</p>
      <ul>
        <li><strong>Columns</strong>: timestamp · actor · IP · action · target · before/after diff hash.</li>
        <li><strong>Filters</strong>: actor · action type · date range · resource.</li>
        <li><strong>Full-text search</strong> across action bodies.</li>
        <li><strong>Export</strong>: CSV for spreadsheets, JSON for SIEM ingestion.</li>
        <li><strong>Immutable</strong>: each row has a Merkle hash; tampering is detectable.</li>
      </ul>
    `,
    side: 'bottom',
    askFeedback: true,
  },
]

// ────────────────────────────────────────────────────────────────────────────
// Persona 3 — Tom Reyes
// ────────────────────────────────────────────────────────────────────────────
const datacenterTour: TourStep[] = [
  {
    id: 'dc.intro',
    route: '/',
    section: 'Marketing landing',
    title: 'Meet Tom — datacenter ops',
    body: `
      <p>Tom is on the supply side. Sells GPU-hours into the venue. Cares about <strong>fill rate · dispatch · getting paid</strong>.</p>
      <p>The partner portal is <em>not</em> in the marketing nav or the trader sidebar — it's a different audience reached by direct URL or CSM hand-off. About 3 minutes, seven stops.</p>
    `,
    side: 'over',
    askFeedback: false,
  },
  {
    id: 'dc.home.audience',
    route: '/',
    section: 'Audience split — datacenter card',
    element: '.audiences, section',
    title: '"For datacenters" — supply-side pitch',
    body: `
      <p>In the three-column audience split, the right card is for partners. Tagline: <em>"Sell capacity into the index."</em></p>
      <p>The link is currently a placeholder — partners onboard via CSM. In this tour we deep-link to the partner portal at <code>/datacenter</code>.</p>
    `,
    side: 'top',
    askFeedback: true,
  },
  {
    id: 'dc.dashboard.header',
    route: '/datacenter',
    section: 'Partner portal — header',
    element: 'h1, .page-head, header',
    title: 'Partner identity + CSM',
    body: `
      <p>Custom partner-portal layout — light theme but distinct chrome from the marketing or app layouts.</p>
      <ul>
        <li><strong>Partner identity</strong>: e.g. "Northstar DC · Reno NV · partner ID <code>pdc_7c2a…</code>".</li>
        <li><strong>CSM contact strip</strong>: email · Slack · phone.</li>
        <li><strong>Contract terms link</strong>: capacity SLA, payout schedule, rev-share.</li>
      </ul>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'dc.dashboard.kpis',
    route: '/datacenter',
    section: 'Headline KPIs',
    element: '.kpi, .kpis, .kpi-grid, main',
    title: 'Capacity sold · fill rate · payout pending',
    body: `
      <p>Four-tile KPI strip:</p>
      <ul>
        <li><strong>Capacity sold MTD</strong> in GPU-hours.</li>
        <li><strong>Fill rate %</strong> — published capacity actually matched.</li>
        <li><strong>Payout pending</strong> — accrued, not yet settled.</li>
        <li><strong>Next settlement date</strong> — T+1 wire window.</li>
      </ul>
      <p>Each tile drills through to the underlying ledger slice.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'dc.dashboard.capacity',
    route: '/datacenter',
    section: 'Capacity grid + recent orders',
    element: 'main, .capacity, table',
    title: 'Capacity grid + matched orders',
    body: `
      <ul>
        <li><strong>Capacity grid</strong>: racks × clusters with status — <em>LIVE · MAINT · OFFLINE</em> — and live utilization bars.</li>
        <li><strong>Recent orders</strong> table: which credit families consumed your capacity (text, image, GPU-hours) with timestamps and realized $/GPU-hr.</li>
        <li><strong>Statements</strong> block: monthly settlement statements, downloadable as signed CSV.</li>
      </ul>
      <p>Primary CTA top-right: <strong>+ Register partner</strong> — kicks off a new-site onboarding.</p>
    `,
    side: 'top',
    askFeedback: true,
  },
  {
    id: 'dc.register',
    route: '/datacenter/register',
    section: 'Register capacity',
    element: 'form, h1, main',
    title: 'Multi-step partner registration',
    body: `
      <p>Six-step funnel. Onboarding a datacenter is not a credit card and a checkbox — it's a documentation pack and a venue review.</p>
      <ol>
        <li><strong>Org details</strong> — legal name, jurisdiction, primary contact.</li>
        <li><strong>Site details</strong> — region, PUE, certifications (SOC 2 · ISO 27001 · Tier III).</li>
        <li><strong>Capacity declaration</strong> — GPU type, count, available windows.</li>
        <li><strong>Pricing floor + auto-throttle rules</strong>.</li>
        <li><strong>Banking / payout</strong> — verified bank for T+1 wires.</li>
        <li><strong>Review</strong> → <em>Submit for review</em>. Venue review SLA: 5 business days.</li>
      </ol>
    `,
    side: 'right',
    askFeedback: true,
  },
  {
    id: 'dc.statements',
    route: '/datacenter',
    section: 'Settlement statement detail',
    element: '.statements, table, main',
    title: 'Statements — every fill is an artifact',
    body: `
      <p>Click any row in the Statements table → <em>"Download May 2026 →"</em>. Each statement is:</p>
      <ul>
        <li>A signed CSV of every fill in the period.</li>
        <li>A summary PDF (counterparty redacted) for your books.</li>
        <li>A hash anchored to the venue's public Merkle log — your auditor can verify without trusting us.</li>
      </ul>
    `,
    side: 'top',
    askFeedback: true,
  },
]

// ────────────────────────────────────────────────────────────────────────────
// Bonus — AI Lab / Researcher (compute + inference deep dive)
// ────────────────────────────────────────────────────────────────────────────
const labTour: TourStep[] = [
  {
    id: 'lab.intro',
    route: '/compute',
    section: 'Compute index',
    title: 'AI Lab — provision + inference deep dive',
    body: `
      <p>A subset of Maya's flow zoomed in on the engineer who actually drives the GPUs. ~2 minutes, seven stops.</p>
      <p>You'll launch an H100 box, tail its logs, watch its cost in real time, then jump to the inference catalog.</p>
    `,
    side: 'over',
    askFeedback: false,
  },
  {
    id: 'lab.compute.list',
    route: '/compute',
    section: 'Compute instances',
    element: 'h1, .page-head, table',
    title: 'Your fleet at a glance',
    body: `
      <p>Same view as enterprise: <strong>8 instances</strong>, region column, status column, per-row 8-GPU utilization bars, hourly burn column.</p>
      <p>Click <em>+ New instance</em> top-right or any row to drill in.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'lab.compute.new',
    route: '/compute/new',
    section: 'Launch wizard',
    element: 'form, h1, main',
    title: 'Launch a GPU box',
    body: `
      <p>Quickest path: <strong>8× H100 SXM5</strong> · <strong>EU-CENTRAL</strong> · <strong>CUDA 12.4 + PyTorch 2.4 image</strong> · <strong>SSH key</strong> · $5k budget cap.</p>
      <p>Live price tile shows $23.92 / hr — the line-item breakdown (GPU · disk · system · egress) is below it.</p>
    `,
    side: 'right',
    askFeedback: true,
  },
  {
    id: 'lab.compute.overview',
    route: `/compute/${DEMO_INSTANCE_ID}`,
    section: 'Instance overview',
    element: 'h1, .head, .grid-3',
    title: 'Per-GPU utilization, live',
    body: `
      <p>Eight H100s, each with:</p>
      <ul>
        <li>Utilization bar (% busy).</li>
        <li>Memory %.</li>
        <li>Temperature (turns amber at 78 °C).</li>
        <li>Watts.</li>
      </ul>
      <p>RAM gauge ring on the right, four throughput sparklines (disk-read · disk-write · net-in · net-out). Refreshes every 1.5 s. Config card below shows image, IPv4, SSH command, tags.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'lab.compute.logs',
    route: `/compute/${DEMO_INSTANCE_ID}`,
    section: 'Instance logs',
    element: '.tabs, [role="tablist"], nav',
    title: 'Logs tab — live tail',
    body: `
      <p>Click the <strong>Logs</strong> tab on the instance detail.</p>
      <ul>
        <li><strong>Follow toggle</strong> — pause / resume auto-scroll.</li>
        <li><strong>Stream filter</strong> — all / stdout / stderr.</li>
        <li><strong>Full-text search</strong>.</li>
        <li><strong>Download archive</strong> — full history (last 200 lines in the live tail, rest in object storage).</li>
      </ul>
      <p>Lines stream every ~900 ms in the demo to simulate a real training loop (epoch / step / loss / tok-per-sec).</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'lab.compute.cost',
    route: `/compute/${DEMO_INSTANCE_ID}`,
    section: 'Instance cost',
    element: '.tabs, [role="tablist"], nav',
    title: 'Cost tab — hourly bars + breakdown',
    body: `
      <ul>
        <li>Four KPI cards: <strong>hourly rate · spend MTD · projected month · budget remaining</strong>.</li>
        <li>48-hour bar chart of per-hour spend.</li>
        <li>Breakdown card: GPU compute · provisioned IOPS storage · system · egress.</li>
      </ul>
      <p>Set a budget cap on the instance and it auto-stops on breach — no surprise bills.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
  {
    id: 'lab.inference',
    route: '/inference',
    section: 'Inference catalog',
    element: 'h1, .page-head, main',
    title: 'Model catalog + playground',
    body: `
      <p>If you'd rather not run your own box, hit a hosted model via the catalog. Each model card has:</p>
      <ul>
        <li>Price per 1 M tokens.</li>
        <li>p50 / p99 latency.</li>
        <li>Throughput tokens/s.</li>
        <li>Context window + params.</li>
      </ul>
      <p>Center playground for prompt iteration; <em>Code</em> tab for the equivalent cURL / Python / JS snippet.</p>
    `,
    side: 'bottom',
    askFeedback: true,
  },
]

// ────────────────────────────────────────────────────────────────────────────
export const tours: Record<PersonaId, TourStep[]> = {
  trader:     traderTour,
  enterprise: enterpriseTour,
  datacenter: datacenterTour,
  lab:        labTour,
}
