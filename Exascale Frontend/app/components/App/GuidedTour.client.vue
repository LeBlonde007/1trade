<script setup lang="ts">
/**
 * App/GuidedTour.client.vue — driver.js runner for persona-driven tours.
 *
 * Mounted once per layout (app + marketing). Watches:
 *   - useGuidedTour().currentStep        → re-arms driver.js
 *   - useRoute().fullPath                 → triggers a re-arm if route just changed
 *
 * Cross-route handling:
 *   When currentStep changes and step.route !== current route, we router.push()
 *   the new route. After navigation completes we wait for the target selector
 *   to appear (briefly polled), then arm driver.js. If the selector never
 *   resolves, we fall back to a centered popover.
 *
 * All popover DOM is built via createElement + textContent (no innerHTML).
 * The one place author-supplied HTML enters is `step.body`, which comes from
 * the static module ~/data/tour-scripts.ts (build-time, trusted). We parse it
 * through DOMParser and append the resulting nodes — still no innerHTML write.
 */
import { driver, type Driver } from 'driver.js'
import 'driver.js/dist/driver.css'
import type { TourStep } from '~/data/tour-scripts'

const tour = useGuidedTour()
const route = useRoute()
const router = useRouter()
const toasts = useToasts?.()

let instance: Driver | null = null
let polling: ReturnType<typeof setInterval> | null = null

function teardown() {
  if (instance) { instance.destroy(); instance = null }
  if (polling) { clearInterval(polling); polling = null }
  document.body.classList.remove('gt-active')
}

function findElement(selector?: string): Element | null {
  if (!selector) return null
  for (const part of selector.split(',').map(s => s.trim()).filter(Boolean)) {
    const el = document.querySelector(part)
    if (el) return el
  }
  return null
}

function arm() {
  if (!tour.active.value) return
  const step = tour.currentStep.value
  if (!step) return

  if (step.route !== route.path) {
    router.push(step.route)
    return
  }

  const start = performance.now()
  if (polling) clearInterval(polling)
  polling = setInterval(() => {
    const found = step.element ? findElement(step.element) : null
    const stepHasNoElement = !step.element
    const timedOut = performance.now() - start > 2500
    if (found || stepHasNoElement || timedOut) {
      clearInterval(polling!); polling = null
      mount(step, found)
    }
  }, 80)
}

function mount(step: TourStep, anchor: Element | null) {
  teardown()
  document.body.classList.add('gt-active')

  instance = driver({
    showProgress: false,
    showButtons: [],
    allowClose: false,
    overlayOpacity: 0.55,
    popoverClass: `gt-popover gt-side-${step.side ?? 'auto'}`,
    onDestroyStarted: () => {
      tour.stop()
    },
    onPopoverRender: (popover) => {
      // Replace the default description element with our DOM-built content.
      // driver.js v1 stringifies `description`, so we can't pass an HTMLElement
      // through the steps config. Instead we mutate the rendered popover here.
      const wrapper = (popover as { wrapper: HTMLElement }).wrapper
      const descEl  = wrapper.querySelector('.driver-popover-description') as HTMLElement | null
      if (!descEl) return
      while (descEl.firstChild) descEl.removeChild(descEl.firstChild)
      descEl.appendChild(buildPopoverContent(step))
    },
    steps: [
      {
        element: anchor ?? undefined,
        popover: {
          title:       step.title,
          description: ' ',  // placeholder; replaced in onPopoverRender
          side:        (step.side && step.side !== 'over') ? step.side : 'bottom',
          align:       'start',
        },
      },
    ],
  })

  instance.drive()
}

function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  opts: { class?: string; text?: string; attrs?: Record<string, string> } = {},
  ...children: Node[]
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag)
  if (opts.class) node.className = opts.class
  if (opts.text !== undefined) node.textContent = opts.text
  if (opts.attrs) for (const [k, v] of Object.entries(opts.attrs)) node.setAttribute(k, v)
  for (const c of children) node.appendChild(c)
  return node
}

/**
 * Parse a trusted, author-supplied HTML snippet from tour-scripts.ts into
 * a real DOM fragment. Uses DOMParser, then strips any <script>/<iframe>/event
 * handlers as a belt-and-braces guard even though the source is build-time.
 */
function parseTrustedHtml(html: string): DocumentFragment {
  const frag = document.createDocumentFragment()
  const parsed = new DOMParser().parseFromString(`<div>${html}</div>`, 'text/html')
  const root = parsed.body.firstElementChild
  if (!root) return frag

  const banned = new Set(['SCRIPT', 'IFRAME', 'OBJECT', 'EMBED', 'STYLE'])
  const walk = (node: Element) => {
    Array.from(node.children).forEach(walk)
    if (banned.has(node.tagName)) { node.remove(); return }
    for (const attr of Array.from(node.attributes)) {
      if (attr.name.toLowerCase().startsWith('on')) node.removeAttribute(attr.name)
      if (attr.name.toLowerCase() === 'href' && attr.value.trim().toLowerCase().startsWith('javascript:')) {
        node.removeAttribute('href')
      }
    }
  }
  walk(root)

  while (root.firstChild) frag.appendChild(root.firstChild)
  return frag
}

function buildPopoverContent(step: TourStep): HTMLElement {
  const personaName = tour.persona.value ?? ''
  const progress = `${tour.index.value + 1} / ${tour.total.value}`

  // ── Meta row ──
  const meta = el('div', { class: 'gt-meta' },
    el('span', { class: 'gt-meta-tag',      text: personaName }),
    el('span', { class: 'gt-meta-section',  text: step.section }),
    el('span', { class: 'gt-meta-progress', text: progress }),
  )

  // ── Body (trusted HTML → real nodes) ──
  const body = el('div', { class: 'gt-body' })
  body.appendChild(parseTrustedHtml(step.body))

  // ── Optional feedback form ──
  const askFeedback = step.askFeedback !== false
  const isFirst = tour.index.value === 0
  const isLast  = tour.isLast.value

  let textarea: HTMLTextAreaElement | null = null

  const fb = el('div', { class: 'gt-fb' })
  if (askFeedback) {
    fb.appendChild(el('div', { class: 'gt-fb-label', text: 'Feedback on this step (optional)' }))
    textarea = el('textarea', {
      class: 'gt-fb-text',
      attrs: { rows: '2', placeholder: 'What worked, what confused you, what\'s missing?' },
    })
    fb.appendChild(textarea)
  }

  // ── Footer buttons ──
  const foot = el('div', { class: 'gt-foot' })

  const endBtn = el('button', { class: 'gt-btn gt-btn-ghost', text: 'End tour', attrs: { type: 'button' } })
  endBtn.addEventListener('click', () => tour.stop())
  foot.appendChild(endBtn)

  foot.appendChild(el('span', { class: 'gt-foot-spacer' }))

  const prevBtn = el('button', { class: 'gt-btn gt-btn-ghost', text: 'Back', attrs: { type: 'button' } })
  if (isFirst) prevBtn.setAttribute('disabled', '')
  prevBtn.addEventListener('click', () => tour.prev())
  foot.appendChild(prevBtn)

  if (askFeedback) {
    const submitBtn = el('button', {
      class: 'gt-btn gt-btn-primary',
      text:  isLast ? 'Finish' : 'Next',
      attrs: { type: 'button' },
    })
    submitBtn.addEventListener('click', async () => {
      const body = textarea?.value ?? ''
      if (body.trim()) {
        submitBtn.disabled = true
        submitBtn.textContent = 'Saving…'
        const ok = await tour.submitFeedback(body)
        if (ok) {
          toasts?.push({ tone: 'pos', title: 'Feedback saved', body: 'Logged to the tour transcript.' })
        } else {
          toasts?.push({ tone: 'warn', title: 'Feedback didn\'t save', body: 'We\'ll still move on. Check console.' })
        }
      }
      tour.next()
    })
    foot.appendChild(submitBtn)
  } else {
    const nextBtn = el('button', {
      class: 'gt-btn gt-btn-primary',
      text:  isLast ? 'Finish' : 'Next',
      attrs: { type: 'button' },
    })
    nextBtn.addEventListener('click', () => tour.next())
    foot.appendChild(nextBtn)
  }

  // Compose
  const root = el('div', { class: 'gt-content' })
  root.appendChild(meta)
  root.appendChild(body)
  if (askFeedback) root.appendChild(fb)
  root.appendChild(foot)
  return root
}

watch(
  () => [tour.active.value, tour.index.value, tour.persona.value, route.fullPath].join('|'),
  () => {
    if (!tour.active.value) { teardown(); return }
    arm()
  },
  { immediate: true }
)

onBeforeUnmount(() => teardown())
</script>

<template>
  <span class="gt-mount-root" aria-hidden="true" />
</template>

<style>
/* GLOBAL — must not be scoped: driver.js renders outside this component's tree. */

.gt-mount-root { display: none; }

.driver-popover.gt-popover {
  background: var(--surface-overlay, #1C1F26);
  color: var(--text-primary, #E8E6E0);
  border: 1px solid rgba(255,255,255,0.16);
  border-radius: 6px;
  padding: 14px 16px 12px;
  font-family: 'Inter', system-ui, sans-serif;
  font-feature-settings: 'tnum';
  max-width: 420px;
  box-shadow: 0 12px 32px rgba(0,0,0,0.4);
}
[data-theme='light'] .driver-popover.gt-popover {
  background: #FFFFFF;
  color: #0A0B0E;
  border: 1px solid rgba(0,0,0,0.12);
  box-shadow: 0 12px 32px rgba(0,0,0,0.12);
}

.driver-popover.gt-popover .driver-popover-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 6px;
  letter-spacing: -0.005em;
}

.driver-popover.gt-popover .driver-popover-footer,
.driver-popover.gt-popover .driver-popover-progress-text,
.driver-popover.gt-popover .driver-popover-close-btn {
  display: none !important;
}

.driver-popover.gt-popover .gt-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
.driver-popover.gt-popover .gt-meta-tag {
  padding: 2px 6px;
  border-radius: 2px;
  background: rgba(200,242,92,0.16);
  color: #C8F25C;
}
[data-theme='light'] .driver-popover.gt-popover .gt-meta-tag {
  background: rgba(74,144,226,0.12);
  color: #2A5FA6;
}
.driver-popover.gt-popover .gt-meta-section {
  color: var(--text-tertiary, #5F5F5C);
  flex: 1;
  font-weight: 500;
  letter-spacing: 0;
  text-transform: none;
  font-size: 11px;
}
.driver-popover.gt-popover .gt-meta-progress {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-weight: 500;
  color: var(--text-tertiary, #5F5F5C);
  letter-spacing: 0;
  text-transform: none;
}

.driver-popover.gt-popover .gt-body {
  font-size: 13px;
  line-height: 1.55;
  color: var(--text-secondary, #9A9A95);
  margin-bottom: 12px;
}
[data-theme='light'] .driver-popover.gt-popover .gt-body {
  color: #5F5F5C;
}
.driver-popover.gt-popover .gt-body strong { color: var(--text-primary, #E8E6E0); font-weight: 600; }
[data-theme='light'] .driver-popover.gt-popover .gt-body strong { color: #0A0B0E; }
.driver-popover.gt-popover .gt-body kbd {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 11px;
  padding: 1px 5px;
  border-radius: 3px;
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.12);
}
[data-theme='light'] .driver-popover.gt-popover .gt-body kbd {
  background: rgba(0,0,0,0.04);
  border-color: rgba(0,0,0,0.12);
}
.driver-popover.gt-popover .gt-body a { color: #4A90E2; }

.driver-popover.gt-popover .gt-fb {
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 4px;
  padding: 10px;
  margin-bottom: 12px;
}
[data-theme='light'] .driver-popover.gt-popover .gt-fb {
  background: rgba(0,0,0,0.02);
  border-color: rgba(0,0,0,0.06);
}
.driver-popover.gt-popover .gt-fb-label {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-tertiary, #5F5F5C);
  margin-bottom: 6px;
}
.driver-popover.gt-popover .gt-fb-rating {
  display: flex;
  gap: 4px;
  margin-bottom: 8px;
}
.driver-popover.gt-popover .gt-fb-rate {
  background: none;
  border: none;
  color: rgba(255,255,255,0.18);
  font-size: 16px;
  line-height: 1;
  padding: 2px;
  cursor: pointer;
  transition: color 100ms ease;
}
[data-theme='light'] .driver-popover.gt-popover .gt-fb-rate { color: rgba(0,0,0,0.18); }
.driver-popover.gt-popover .gt-fb-rate:hover,
.driver-popover.gt-popover .gt-fb-rate.on { color: #F59E0B; }

.driver-popover.gt-popover .gt-fb-text {
  width: 100%;
  background: rgba(0,0,0,0.32);
  border: 1px solid rgba(255,255,255,0.10);
  border-radius: 3px;
  color: var(--text-primary, #E8E6E0);
  font: inherit;
  font-size: 12px;
  padding: 7px 8px;
  resize: vertical;
  min-height: 44px;
  font-family: inherit;
}
[data-theme='light'] .driver-popover.gt-popover .gt-fb-text {
  background: #FFFFFF;
  border-color: rgba(0,0,0,0.16);
  color: #0A0B0E;
}
.driver-popover.gt-popover .gt-fb-text:focus {
  outline: none;
  border-color: #4A90E2;
  box-shadow: 0 0 0 3px rgba(74,144,226,0.18);
}

.driver-popover.gt-popover .gt-foot {
  display: flex;
  align-items: center;
  gap: 6px;
}
.driver-popover.gt-popover .gt-foot-spacer { flex: 1; }

.driver-popover.gt-popover .gt-btn {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 500;
  border-radius: 3px;
  cursor: pointer;
  font-family: inherit;
  transition: background 100ms ease, border-color 100ms ease;
}
.driver-popover.gt-popover .gt-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.driver-popover.gt-popover .gt-btn-ghost {
  background: transparent;
  color: var(--text-secondary, #9A9A95);
  border: 1px solid rgba(255,255,255,0.10);
}
.driver-popover.gt-popover .gt-btn-ghost:hover:not(:disabled) {
  background: rgba(255,255,255,0.06);
  color: var(--text-primary, #E8E6E0);
}
[data-theme='light'] .driver-popover.gt-popover .gt-btn-ghost {
  color: #5F5F5C;
  border-color: rgba(0,0,0,0.12);
}
[data-theme='light'] .driver-popover.gt-popover .gt-btn-ghost:hover:not(:disabled) {
  background: rgba(0,0,0,0.04);
  color: #0A0B0E;
}

.driver-popover.gt-popover .gt-btn-primary {
  background: var(--brand-primary, #C8F25C);
  color: #0A0B0E;
  border: 1px solid var(--brand-primary, #C8F25C);
  font-weight: 600;
}
.driver-popover.gt-popover .gt-btn-primary:hover:not(:disabled) {
  background: var(--brand-primary-hover, #B8E548);
}

.driver-overlay { transition: opacity 180ms ease; }
body.gt-active { overflow-x: hidden; }

.driver-popover.gt-side-over {
  position: fixed !important;
  top: 50% !important;
  left: 50% !important;
  transform: translate(-50%, -50%);
}
</style>
