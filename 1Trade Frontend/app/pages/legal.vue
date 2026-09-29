<script setup lang="ts">
/**
 * /legal — where every legal link on the site lands (terms, privacy, trading rules, risk disclosure,
 * compliance). The full documents are being prepared with counsel; until they are published this
 * page says so plainly and states the facts that already hold for the product as it runs today.
 * Public — no auth.
 */
definePageMeta({ layout: 'marketing' })
useHead({ title: 'Legal — 1Trade' })

const sections = [
  {
    id: 'terms', title: 'Terms of service',
    body: [
      'Credits on 1Trade are prepaid service units: each credit is redeemable for a defined amount of AI inference or GPU compute on the platform. They are not a deposit and not a financial product.',
      'Purchases are charged in US dollars. Credits are delivered to your wallet once the payment settles (card at once; ACH when the bank debit clears; wire when the funds arrive and are matched).',
    ],
  },
  {
    id: 'privacy', title: 'Privacy',
    body: [
      'We store your email address, a hash of your password (never the password), your account settings, and the activity you create on the platform — purchases, usage, orders and fills — so we can run the service and keep the audit trail it requires.',
      'We do not sell personal data. Payment card details are handled by our payment processor and never reach our servers.',
    ],
  },
  {
    id: 'trading-rules', title: 'Trading rules',
    body: [
      'Trading is paper-only: orders use paper cash and paper credits, and no real money changes hands. Real-money trading stays closed until exchange licensing is complete.',
      'Orders match by price, then time. Fees are 0.50% for the maker and 1.00% for the taker. Self-trades are prevented, and every trade is recorded on an append-only, hash-chained log.',
      'Paper books are quoted by a clearly labelled paper liquidity account so they are never empty. Surveillance watches every paper market for wash trades, spoofing, layering and similar patterns.',
    ],
  },
  {
    id: 'risk', title: 'Risk disclosure',
    body: [
      'Paper trading results are not indicative of any real-money outcome. Paper prices move with a reference price and paper order flow, and paper balances have no cash value.',
      'The AI Index values shown are provisional until the index service is in production; the methodology and its audit chain are public on the index page.',
    ],
  },
  {
    id: 'compliance', title: 'Compliance',
    body: [
      'Real-money purchases require identity verification (KYC) before they are accepted. Every change to credits, orders and administrative settings is written to an audit log.',
      'Questions about these terms or about your data: contact@1trade.com.',
    ],
  },
]
</script>

<template>
  <div class="legal">
    <header class="head">
      <p class="eyebrow">Legal</p>
      <h1>Terms, privacy and trading rules</h1>
      <p class="lede">
        The full legal documents are being prepared with counsel and will be published before any
        real-money service opens. Until then, this page states how the platform works today.
      </p>
      <nav class="toc" aria-label="Sections">
        <a v-for="s in sections" :key="s.id" :href="'#' + s.id">{{ s.title }}</a>
      </nav>
    </header>
    <section v-for="s in sections" :id="s.id" :key="s.id" class="sec">
      <h2>{{ s.title }}</h2>
      <p v-for="(p, i) in s.body" :key="i">{{ p }}</p>
    </section>
  </div>
</template>

<style scoped>
.legal { max-width: 760px; margin: 0 auto; padding: 72px var(--sp-5) 96px; color: var(--text); }
.eyebrow { font-family: var(--font-mono); font-size: var(--fs-tiny); letter-spacing: 0.16em; text-transform: uppercase; color: var(--text-3); margin: 0; }
h1 { font-size: var(--fs-3xl); letter-spacing: -0.02em; margin: 8px 0 12px; }
.lede { color: var(--text-2); line-height: 1.6; margin: 0 0 var(--sp-4); }
.toc { display: flex; flex-wrap: wrap; gap: var(--sp-2) var(--sp-4); padding-bottom: var(--sp-5); border-bottom: 1px solid var(--border); }
.toc a { color: var(--brand); font-size: var(--fs-sm); text-decoration: none; }
.toc a:hover { text-decoration: underline; }
.sec { padding: var(--sp-5) 0; border-bottom: 1px solid var(--border); scroll-margin-top: 80px; }
.sec h2 { font-size: var(--fs-lg); margin: 0 0 var(--sp-3); }
.sec p { color: var(--text-2); line-height: 1.65; margin: 0 0 var(--sp-3); font-size: var(--fs-sm); }
</style>
