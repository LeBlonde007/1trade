/**
 * useTour — persona-driven autoplay product tour (L6).
 *
 * Flow:
 *   1. Open → persona picker (Trader · Enterprise · Datacenter Partner)
 *   2. Auto-play the persona's journey: navigate, spotlight, narrate, advance
 *   3. End → feedback card (rating + comment)
 *
 * Controls: play / pause / next / prev / scrub / skip-to-end.
 *
 *   const t = useTour()
 *   t.open()                   // shows persona picker
 *   t.openAs('trader')         // skip picker, start trader journey directly
 */
export type Persona = 'trader' | 'enterprise' | 'partner'
export type Phase   = 'picker' | 'playing' | 'feedback'

export interface TourStep {
  /** Route to navigate to before the step is shown. Skipped if already there. */
  route: string
  /** CSS selector for the element to spotlight. Comma-separated fallbacks allowed. */
  target: string
  /** Short title shown on the tooltip and in the player chip. */
  title: string
  /** One-sentence narration body. */
  body: string
  /** Dwell time in ms before auto-advance. */
  ms: number
  /** Optional tooltip placement override. */
  placement?: 'right' | 'left' | 'top' | 'bottom'
}

export interface PersonaScript {
  id: Persona
  name: string
  who: string
  runtime: string
  color: string
  steps: TourStep[]
}

export const TOUR_PERSONAS: PersonaScript[] = [
  {
    id: 'trader',
    name: 'Jordan Park',
    who: 'Independent quant · retail trader',
    runtime: '~2:00',
    color: 'var(--brand)',
    steps: [
      { route: '/',         target: '.brand, header .brand, .container-x',                                 title: 'Landing',          body: 'A new visitor lands on the public marketing page and sees the live AI Index ticker in the footer.',          ms: 5500 },
      { route: '/signup',   target: 'h1, .lf-title, .auth-title',                                          title: 'Open an account',  body: 'Standard email + password. Strength meter, terms checkbox — low-friction onboarding.',                        ms: 6000 },
      { route: '/onboarding/verify', target: '.card, .verify-card',                                        title: 'Verify email',     body: 'Verification link sent. Resend timer prevents abuse; live AI Index footer keeps the feel of a real market.', ms: 6000 },
      { route: '/onboarding/welcome', target: '.hero, .welcome-wrap, h1',                                  title: 'Welcome',          body: 'Light warmth. Paper-trading account ready with $10,000. No consumer cheer.',                                 ms: 6500 },
      { route: '/trade',    target: '.market-pill',                                                        title: 'Live ticker',      body: 'AI Index price + spread visible everywhere in the app. The venue heartbeat.',                                ms: 5500 },
      { route: '/trade',    target: '.sidebar',                                                            title: 'Navigation',       body: 'Trade · Markets · Index · Portfolio · History · Wallet · Compute · Inference.',                              ms: 5500, placement: 'right' },
      { route: '/trade',    target: '.tv-chart, .chart-wrap, .candle, canvas',                             title: 'Candle chart',     body: 'Real-time OHLC with adjustable timeframes. Crosshair shows price + volume.',                                 ms: 6000 },
      { route: '/trade',    target: '.order-book, .book',                                                  title: 'Order book',       body: 'Bid / ask depth, ticking live. Click any level to pre-fill the order form.',                                 ms: 6000, placement: 'left' },
      { route: '/trade',    target: '.order-form',                                                         title: 'Place an order',   body: 'Market or limit, choose side, hit Submit. Fills appear in the tape + positions panel.',                      ms: 6000, placement: 'left' },
      { route: '/portfolio',target: '.hero, .portfolio-hero, .perf-card, h1',                              title: 'Portfolio',        body: 'P&L against AI-INDEX is built-in. You always know whether you beat beta or rode it.',                        ms: 6500 },
      { route: '/wallet',   target: '.wallet-balances, .balances, .balance-card, h1',                       title: 'Wallet',           body: 'Cash + AI-IDX + sub-credits. Conversion is one click at the venue rate.',                                    ms: 6000 },
      { route: '/benchmark',target: 'h1, .meth-title, .doc-title',                                         title: 'Methodology',      body: 'Every AI-INDEX constituent, weight, and audit pointer is public.',                                            ms: 6500 },
    ],
  },
  {
    id: 'enterprise',
    name: 'Maya Chen',
    who: 'VP Engineering · enterprise buyer',
    runtime: '~2:15',
    color: 'var(--info)',
    steps: [
      { route: '/',                            target: '.audiences, .for-ai, .container-x',                title: 'Audience split',   body: 'Enterprise procurement is sales-assisted, not self-serve. Maya was handed an onboarding URL.',               ms: 5500 },
      { route: '/enterprise/onboarding',       target: '.csm-card, h1, .pc-title',                         title: 'Sales-assist',     body: 'Dedicated CSM with email, Slack, and phone. MSA + pricing schedule on the right rail.',                       ms: 7000 },
      { route: '/enterprise/onboarding',       target: '.checklist, .checklist-items',                     title: 'Activation steps', body: '6-step checklist: MSA · credit · billing entity · IdP · invite team · first instance. Every check unlocks usage.', ms: 7000 },
      { route: '/enterprise/teams',            target: 'h1, .teams-head',                                  title: 'Team & sub-accounts', body: '8 members across 4 sub-accounts. Per-seat budgets so finance keeps the lid on.',                          ms: 6500 },
      { route: '/enterprise/teams',            target: '.budget-bars, .budgets, .sub-account-budgets',     title: 'Per-seat budgets', body: 'AI Research at $50K/mo, Customer Service AI at $30K. Hard caps + auto-stop.',                                ms: 6500 },
      { route: '/compute',                     target: 'h1, .compute-head',                                title: 'Compute',          body: '8 instances across 4 regions. Real-time utilization + spend per row.',                                       ms: 6500 },
      { route: '/compute/new',                 target: 'h1, .new-instance-head, .gpu-pick',                 title: 'Provision GPUs',   body: 'H100 / H200 picker, region, image, SSH key, budget cap. Live price calculator.',                              ms: 7000 },
      { route: '/inference',                   target: '.model-catalog, .catalog, .inference-head, h1',     title: 'Inference catalog',body: '10 models across text / speech / image / video / embed. Same credit funds either trading or inference.',     ms: 7000 },
      { route: '/settings#api',                target: '.api-card, .api-table-wrap, .api-table',           title: 'API keys',         body: 'Scoped, expiry-bound, one-time-reveal. Every issuance lands on the audit chain.',                            ms: 6500 },
      { route: '/enterprise/audit',            target: 'h1, .audit-page h1, .integrity',                   title: 'Audit log',        body: 'Immutable, hash-chained event log. Compliance officer can export signed CSV / JSON for regulators.',          ms: 7000 },
      { route: '/enterprise/billing',          target: '.kpi-strip, .stats, .billing-head, h1',            title: 'Billing',          body: 'MTD spend, projected month-end, budget cap, status. Cost alerts wired to Slack.',                            ms: 6500 },
    ],
  },
  {
    id: 'partner',
    name: 'Tom Reyes',
    who: 'Capacity Ops · datacenter partner',
    runtime: '~1:30',
    color: 'var(--accent)',
    steps: [
      { route: '/',                       target: '.for-datacenters, .audiences, .container-x',            title: 'Supply side',      body: 'On the other side of every trade is a datacenter shipping electrons. Here is what they see.',                ms: 5500 },
      { route: '/datacenter',             target: 'h1, .top-cta, .partner-head',                           title: 'Partner portal',   body: 'Northstar DC · Reno NV. Capacity sold, fill rate, payout pending, next settlement.',                         ms: 7000 },
      { route: '/datacenter',             target: '.kpi-strip, .capacity-grid, .stat',                     title: 'Capacity grid',    body: 'Rack × cluster utilization in real time. Green normal, amber heat, red full.',                              ms: 6500 },
      { route: '/datacenter',             target: '.orders-table, .statements, .statements-card, table',   title: 'Orders + settlements', body: 'Which credit families consumed your capacity. Monthly statements downloadable as CSV.',                  ms: 7000 },
      { route: '/datacenter/register',    target: 'h1, .reg-head, .form-wrap',                             title: 'Register capacity',body: 'Multi-step intake: org, site (SOC 2 / Tier III), capacity declaration, pricing floor, banking.',             ms: 7500 },
    ],
  },
]

export function useTour() {
  const isOpen     = useState<boolean>('tour-open',     () => false)
  const phase      = useState<Phase>('tour-phase',      () => 'picker')
  const personaId  = useState<Persona | null>('tour-persona', () => null)
  const stepIdx    = useState<number>('tour-step',      () => 0)
  const playing    = useState<boolean>('tour-playing',  () => false)
  /** 0..1, current step playback progress. */
  const progress   = useState<number>('tour-progress',  () => 0)
  /** Feedback values */
  const fbRating   = useState<number>('tour-fb-rating', () => 0)
  const fbComment  = useState<string>('tour-fb-comment',() => '')

  const persona = computed<PersonaScript | null>(() => {
    if (!personaId.value) return null
    return TOUR_PERSONAS.find((p) => p.id === personaId.value) ?? null
  })
  const currentStep = computed<TourStep | null>(() => {
    if (!persona.value) return null
    return persona.value.steps[stepIdx.value] ?? null
  })
  const totalSteps = computed(() => persona.value?.steps.length ?? 0)

  function open() {
    phase.value     = 'picker'
    personaId.value = null
    stepIdx.value   = 0
    progress.value  = 0
    playing.value   = false
    fbRating.value  = 0
    fbComment.value = ''
    isOpen.value    = true
  }

  function openAs(id: Persona) {
    open()
    selectPersona(id)
  }

  function selectPersona(id: Persona) {
    personaId.value = id
    stepIdx.value   = 0
    progress.value  = 0
    phase.value     = 'playing'
    playing.value   = true
  }

  function play()  { playing.value = true }
  function pause() { playing.value = false }
  function toggle(){ playing.value = !playing.value }

  function next() {
    if (!persona.value) return
    if (stepIdx.value >= persona.value.steps.length - 1) {
      finish()
      return
    }
    stepIdx.value++
    progress.value = 0
  }
  function prev() {
    if (stepIdx.value <= 0) return
    stepIdx.value--
    progress.value = 0
  }
  function goto(n: number) {
    if (!persona.value) return
    stepIdx.value = Math.max(0, Math.min(persona.value.steps.length - 1, n))
    progress.value = 0
  }
  function finish() {
    playing.value = false
    phase.value   = 'feedback'
  }
  function close() {
    isOpen.value  = false
    playing.value = false
  }

  function submitFeedback() {
    // In a real app this would POST to /api/tour-feedback.
    // For demo we toast and close.
    try {
      const t = useToasts()
      t.push({
        tone: 'pos',
        title: 'Thanks for the feedback',
        body: 'Rating: ' + fbRating.value + '/5' + (fbComment.value ? ' · "' + fbComment.value.slice(0, 60) + '"' : ''),
      })
    } catch {
      // useToasts may not be mounted yet in some contexts
    }
    close()
  }

  return {
    isOpen,
    phase,
    personaId,
    persona,
    stepIdx,
    currentStep,
    totalSteps,
    playing,
    progress,
    fbRating,
    fbComment,
    open,
    openAs,
    selectPersona,
    play,
    pause,
    toggle,
    next,
    prev,
    goto,
    finish,
    close,
    submitFeedback,
  }
}
