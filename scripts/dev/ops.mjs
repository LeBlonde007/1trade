#!/usr/bin/env node
// scripts/dev/ops.mjs — the back-office half of the flows, for LOCAL testing only.
//
// Some flows finish outside the web app: a bank clears an ACH debit (a signed Stripe webhook), the
// finance team matches an incoming wire, a datacenter's agent attests its GPUs, operations set a
// partner's payout terms and close a payout cycle. This script plays those parts against the local
// stack (`make up`: platform-core :8001, compute-control :8086) so every flow can be clicked through.
//
// It needs Node 18+ and nothing else. The service token is read from $SERVICE_TOKEN, else from the
// local cluster's `platform-auth` Secret via kubectl. Run with no arguments for the command list.
import { createHash, createHmac, generateKeyPairSync, createPrivateKey, createPublicKey, sign } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const PC = process.env.PLATFORM_CORE_URL || 'http://localhost:8001'
const CC = process.env.COMPUTE_CONTROL_URL || 'http://localhost:8086'
// Must match STRIPE_WEBHOOK_SECRET in the Tiltfile's local platform-core config.
const WEBHOOK_SECRET = process.env.STRIPE_WEBHOOK_SECRET || 'whsec_local_dev_only'
const KEY_FILE = join(dirname(fileURLToPath(import.meta.url)), '..', '..', '.dev', 'attest.pem')

/** die prints a message and exits non-zero. */
function die(msg) {
  console.error('error: ' + msg)
  process.exit(1)
}

/** serviceToken returns the operations token: $SERVICE_TOKEN, else the local cluster's Secret. */
function serviceToken() {
  if (process.env.SERVICE_TOKEN) return process.env.SERVICE_TOKEN
  try {
    const b64 = execFileSync('kubectl', ['get', 'secret', 'platform-auth', '-o', 'jsonpath={.data.SERVICE_TOKEN}'], { encoding: 'utf8' })
    return Buffer.from(b64, 'base64').toString('utf8')
  } catch {
    return die('set SERVICE_TOKEN, or run against the local cluster (kubectl get secret platform-auth)')
  }
}

/** call sends a JSON request and returns { status, body }. */
async function call(method, url, { token, body, headers = {} } = {}) {
  const h = { 'content-type': 'application/json', ...headers }
  if (token) h.authorization = 'Bearer ' + token
  const res = await fetch(url, { method, headers: h, body: body === undefined ? undefined : (typeof body === 'string' ? body : JSON.stringify(body)) })
  const text = await res.text()
  let parsed = text
  try { parsed = text ? JSON.parse(text) : null } catch { /* keep text */ }
  return { status: res.status, body: parsed }
}

/** must returns the body of a 2xx answer, or exits with the step name and the error. */
function must(step, r) {
  if (r.status < 200 || r.status > 299) die(`${step}: HTTP ${r.status} ${JSON.stringify(r.body)}`)
  return r.body
}

/** login signs a user in and returns their token (the partner's agent authenticates as the partner). */
async function login(email, password) {
  const b = must('login', await call('POST', PC + '/v1/auth/login', { body: { email, password } }))
  if (b.mfa_required) die('that account has 2FA on — use an account without 2FA for the agent')
  return b.token
}

/** keyPair loads the local attestation signing key, generating it on first use (gitignored). */
function keyPair() {
  if (!existsSync(KEY_FILE)) {
    mkdirSync(dirname(KEY_FILE), { recursive: true })
    const { privateKey } = generateKeyPairSync('ed25519')
    writeFileSync(KEY_FILE, privateKey.export({ type: 'pkcs8', format: 'pem' }), { mode: 0o600 })
  }
  const priv = createPrivateKey(readFileSync(KEY_FILE))
  return { priv, pub: createPublicKey(priv).export({ type: 'spki', format: 'der' }).subarray(12).toString('base64') }
}

/** stripeEvent posts a Stripe-signed checkout event for a purchase (the mock session id is cs_mock_<id>). */
async function stripeEvent(type, purchaseId, paymentStatus) {
  const payload = JSON.stringify({ id: `evt_dev_${Date.now()}_${Math.random().toString(36).slice(2)}`, type,
    data: { object: { id: 'cs_mock_' + purchaseId, payment_status: paymentStatus } } })
  const t = Math.floor(Date.now() / 1000)
  const sig = createHmac('sha256', WEBHOOK_SECRET).update(`${t}.${payload}`).digest('hex')
  return call('POST', PC + '/v1/billing/webhook/stripe', { body: payload, headers: { 'Stripe-Signature': `t=${t},v1=${sig}` } })
}

const commands = {
  /** pubkey prints the base64 Ed25519 key compute-control must trust (the Tiltfile calls this). */
  async pubkey() {
    process.stdout.write(keyPair().pub)
  },

  /** ach-clear <purchase-id> — the bank clears an ACH debit: credits are booked. */
  async 'ach-clear'(id) {
    if (!id) die('usage: ach-clear <purchase-id>')
    must('checkout completed (debit started)', await stripeEvent('checkout.session.completed', id, 'unpaid'))
    must('debit cleared', await stripeEvent('checkout.session.async_payment_succeeded', id, 'paid'))
    console.log(`ACH ${id} cleared — refresh the wallet.`)
  },

  /** ach-fail <purchase-id> — the debit is returned: nothing is booked. */
  async 'ach-fail'(id) {
    if (!id) die('usage: ach-fail <purchase-id>')
    must('checkout completed (debit started)', await stripeEvent('checkout.session.completed', id, 'unpaid'))
    must('debit returned', await stripeEvent('checkout.session.async_payment_failed', id, 'unpaid'))
    console.log(`ACH ${id} returned — the purchase is failed, no credits booked.`)
  },

  /** wire-received <purchase-id> <amount-usd> — finance records the incoming wire (exact amount books). */
  async 'wire-received'(id, amount) {
    if (!id || !amount) die('usage: wire-received <purchase-id> <amount-usd>   (e.g. 2990.00)')
    const r = await call('POST', `${PC}/v1/billing/wires/${id}/received`, { token: serviceToken(),
      body: { amount_usd: amount, bank_reference: 'DEV' + Date.now() } })
    must('wire received', r)
    console.log(`wire ${id}: ${r.body?.status ?? 'recorded'} — refresh the wallet.`)
  },

  /** attest <source-id> <partner-email> <partner-password> — pass KYB, bond, GPU identity and the
   *  timed challenge for a registered source; it activates and starts taking work. */
  async attest(id, email, password) {
    if (!id || !email || !password) die('usage: attest <source-id> <partner-email> <partner-password>')
    const ops = serviceToken(), agent = await login(email, password), { priv } = keyPair()
    const src = `${CC}/v1/supply/sources/${id}`
    const info = must('read source', await call('GET', src, { token: ops }))
    must('KYB', await call('POST', src + '/attestation/kyb', { token: ops, body: { state: 'pass', evidence: { ref: 'dev-kyb' } } }))
    must('bond', await call('POST', src + '/attestation/bond', { token: ops, body: { state: 'pass', evidence: { amount_usd: '50000.00', custody: 'dev-escrow' } } }))
    must('heartbeat', await call('POST', src + '/heartbeat', { token: agent, body: { gpus_healthy: info.gpu_count, utilization_pct: 20 } }))
    const model = { gpu_h200: 'NVIDIA H200 141GB HBM3e' }[info.gpu_type] || 'NVIDIA H100 80GB HBM3'
    let c = must('hardware challenge', await call('POST', src + '/challenges', { token: agent, body: { kind: 'hardware' } }))
    const report = Buffer.from(JSON.stringify({ source_id: id, nonce: c.nonce,
      gpus: Array.from({ length: info.gpu_count }, (_, i) => ({ uuid: `GPU-dev-${id.slice(0, 8)}-${i}`, model })) }))
    must('hardware report', await call('POST', `${src}/challenges/${c.id}`, { token: agent,
      body: { report: report.toString('base64'), signature: sign(null, report, priv).toString('base64') } }))
    c = must('timed challenge', await call('POST', src + '/challenges', { token: agent, body: { kind: 'challenge' } }))
    let h = createHash('sha256').update(Buffer.from(c.nonce, 'hex')).digest()
    for (let i = 1; i < c.iterations; i++) h = createHash('sha256').update(h).digest()
    const st = must('challenge answer', await call('POST', `${src}/challenges/${c.id}`, { token: agent, body: { result: h.toString('hex') } }))
    const after = must('read source', await call('GET', src, { token: ops }))
    console.log(`attestation complete: ${st.complete}; source state: ${after.state}`)
    if (!st.complete) console.log('hint: compute-control must trust this key — `node scripts/dev/ops.mjs pubkey` (Tilt sets it for you)')
  },

  /** payout-terms <partner-email> <partner-password> — operations set the partner's payout terms. */
  async 'payout-terms'(email, password) {
    if (!email || !password) die('usage: payout-terms <partner-email> <partner-password>')
    const me = must('who am I', await call('GET', PC + '/v1/auth/me', { token: await login(email, password) }))
    const tenant = me.tenant_id || me.tenant?.id
    must('terms', await call('PUT', `${CC}/v1/supply/partners/${tenant}/agreement`, { token: serviceToken(),
      body: { rates: { gpu_h100: '1.800000', gpu_h200: '2.600000' }, fee_percent: '10', holdback_percent: '15', dispute_days: 14 } }))
    console.log(`payout terms set for ${tenant}: H100 $1.80/GPU-h, H200 $2.60/GPU-h, 10% fee, 15% holdback.`)
  },

  /** payout-cycle [hours] — close a payout cycle over the last N hours (default 24). */
  async 'payout-cycle'(hours = '24') {
    const end = Date.now() - 1000, start = end - Number(hours) * 3600e3
    const b = must('cycle', await call('POST', CC + '/v1/supply/payouts/cycles', { token: serviceToken(),
      body: { period_start: new Date(start).toISOString(), period_end: new Date(end).toISOString() } }))
    console.log(`${b.data?.length ?? 0} statement(s) issued — see /datacenter → Payouts.`)
  },
}

const [cmd, ...args] = process.argv.slice(2)
if (!commands[cmd]) {
  console.log(`usage: node scripts/dev/ops.mjs <command>

  ach-clear <purchase-id>                      bank clears an ACH debit (credits booked)
  ach-fail <purchase-id>                       bank returns an ACH debit (nothing booked)
  wire-received <purchase-id> <amount-usd>     finance records an incoming wire
  attest <source-id> <email> <password>        a datacenter source passes attestation and activates
  payout-terms <email> <password>              operations set a partner's payout terms
  payout-cycle [hours]                         close a payout cycle (default: last 24 h)
  pubkey                                       the attestation key compute-control trusts

Purchase ids are on /enterprise/billing; source ids on /datacenter. Local only.`)
  process.exit(cmd ? 1 : 0)
}
await commands[cmd](...args)
