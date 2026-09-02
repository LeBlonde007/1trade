<script setup lang="ts">
/**
 * Base/HelpDot — small "!" affordance next to any input / select / chart.
 * Click → popover with WHAT IT DOES + WHY IT MATTERS + a free-text feedback
 * box. Submission persists via useFieldHelp() and triggers a toast.
 *
 * Usage anywhere:
 *   <label>
 *     Limit price
 *     <BaseHelpDot
 *       title="Limit price"
 *       what="The price you're willing to pay (buy) or accept (sell)."
 *       why="Use a limit when the spread is wide or you don't want to chase price."
 *       field="trade.order.limit-px"
 *     />
 *   </label>
 *
 * Props:
 *   title      — short label, also used as the toast subject
 *   what       — required, one or two sentences in plain English
 *   why        — optional, explains why a user would care / when to use it
 *   field      — optional unique id (recommended) so feedback maps to a place
 *   placement  — 'bottom' (default) | 'top' | 'left' | 'right'
 *   tone       — visual tone: 'mute' (default) | 'on-dark' | 'on-accent'
 */
import { Info, MessageSquare, X, CheckCircle2 } from 'lucide-vue-next'

interface Props {
  title: string
  what: string
  why?: string
  field?: string
  placement?: 'bottom' | 'top' | 'left' | 'right'
  tone?: 'mute' | 'on-dark' | 'on-accent'
}

const props = withDefaults(defineProps<Props>(), {
  placement: 'bottom',
  tone: 'mute',
})

const fh = useFieldHelp()

const open       = ref(false)
const feedback   = ref('')
const submitting = ref(false)
const submitted  = ref(false)
const dotRef     = ref<HTMLElement | null>(null)
const popRef     = ref<HTMLElement | null>(null)

function toggle() {
  open.value = !open.value
  if (open.value) submitted.value = false
}
function close() { open.value = false }

function submit() {
  if (!feedback.value.trim() || submitting.value) return
  submitting.value = true
  fh.submit(props.title, feedback.value, props.field)
  feedback.value = ''
  submitted.value = true
  submitting.value = false
  setTimeout(() => { if (submitted.value) close() }, 1400)
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && open.value) close()
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey) && open.value) submit()
}
function onDocMouseDown(e: MouseEvent) {
  if (!open.value) return
  const t = e.target as Node | null
  if (t && (dotRef.value?.contains(t) || popRef.value?.contains(t))) return
  close()
}

onMounted(() => {
  document.addEventListener('keydown', onKey)
  document.addEventListener('mousedown', onDocMouseDown)
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKey)
  document.removeEventListener('mousedown', onDocMouseDown)
})
</script>

<template>
  <span class="help" :class="['tone-' + tone]">
    <button
      ref="dotRef"
      type="button"
      class="dot"
      :aria-label="'What is ' + title + '? Click for details and to leave feedback'"
      :aria-expanded="open"
      :title="title + ' — click for help'"
      @click.stop="toggle"
    >
      <Info :size="11" :stroke-width="2" />
    </button>

    <Transition name="pop">
      <div
        v-if="open"
        ref="popRef"
        class="pop"
        :class="['p-' + placement]"
        role="dialog"
        :aria-label="title"
        @click.stop
      >
        <header class="head">
          <span class="title">{{ title }}</span>
          <button type="button" class="x" aria-label="Close" @click="close">
            <X :size="13" />
          </button>
        </header>

        <section class="sec">
          <div class="eye">WHAT IT DOES</div>
          <p class="body">{{ what }}</p>
        </section>

        <section v-if="why" class="sec">
          <div class="eye">WHY IT MATTERS</div>
          <p class="body">{{ why }}</p>
        </section>

        <section class="fb">
          <div class="eye"><MessageSquare :size="10" :stroke-width="2" /> LEAVE FEEDBACK</div>
          <textarea
            v-model="feedback"
            rows="3"
            class="textarea"
            placeholder="What's confusing? What would help? (text only, optional)"
            @keydown.stop
          />
          <div class="fb-foot">
            <span v-if="submitted" class="ok">
              <CheckCircle2 :size="12" /> Sent — thanks
            </span>
            <span v-else-if="feedback.trim().length" class="hint mono">
              ⌘↵ to send
            </span>
            <span v-else />
            <button
              type="button"
              class="btn"
              :disabled="!feedback.trim() || submitting"
              @click="submit"
            >
              Submit
            </button>
          </div>
        </section>
      </div>
    </Transition>
  </span>
</template>

<style scoped>
.help {
  position: relative;
  display: inline-flex;
  align-items: center;
  margin-left: 4px;
  vertical-align: middle;
}

.dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 1px solid var(--border-strong);
  background: transparent;
  color: var(--text-3);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  transition: color 120ms ease, border-color 120ms ease, background-color 120ms ease;
}
.dot:hover  { color: var(--text); border-color: var(--text); }
.dot:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

/* Tones */
.tone-on-dark   .dot { color: rgba(232,230,224,0.55); border-color: rgba(232,230,224,0.25); }
.tone-on-dark   .dot:hover { color: rgba(232,230,224,0.95); border-color: rgba(232,230,224,0.6); }
.tone-on-accent .dot { color: rgba(10,11,14,0.55); border-color: rgba(10,11,14,0.32); }
.tone-on-accent .dot:hover { color: var(--text); border-color: var(--text); }

/* Popover */
.pop {
  position: absolute;
  z-index: 200;
  width: 300px;
  background: var(--overlay, var(--elevated));
  color: var(--text);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  box-shadow: 0 16px 36px rgba(0,0,0,0.45);
  padding: 12px 14px 12px;
  font-family: var(--font-sans);
  text-align: left;
  cursor: auto;
}
.p-bottom { top: calc(100% + 8px); left: 50%; transform: translateX(-50%); }
.p-top    { bottom: calc(100% + 8px); left: 50%; transform: translateX(-50%); }
.p-right  { left: calc(100% + 8px); top: 50%; transform: translateY(-50%); }
.p-left   { right: calc(100% + 8px); top: 50%; transform: translateY(-50%); }

.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.title {
  font-weight: 600;
  font-size: 13px;
  color: var(--text);
  letter-spacing: -0.005em;
}
.x {
  width: 22px; height: 22px;
  background: transparent; color: var(--text-3);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.x:hover { color: var(--text); border-color: var(--text); }

.sec  { padding: 6px 0 8px; border-bottom: 1px solid var(--border); }
.sec:last-of-type { border-bottom: none; }
.eye {
  font-family: var(--font-mono);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.16em;
  color: var(--text-3);
  margin-bottom: 5px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.body {
  margin: 0;
  font-size: 12px;
  color: var(--text-2);
  line-height: 1.55;
}

.fb {
  padding-top: 10px;
  margin-top: 4px;
  border-top: 1px solid var(--border);
}
.textarea {
  width: 100%;
  background: var(--canvas);
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 10px;
  font-family: var(--font-sans);
  font-size: 12px;
  line-height: 1.5;
  resize: vertical;
  outline: none;
  min-height: 56px;
}
.textarea:focus { border-color: var(--accent); }

.fb-foot {
  display: flex; align-items: center; justify-content: space-between;
  margin-top: 8px;
  min-height: 24px;
  gap: 8px;
}
.ok   { color: var(--pos); font-size: 11.5px; display: inline-flex; gap: 4px; align-items: center; }
.hint { color: var(--text-3); font-size: 10.5px; }
.btn {
  background: var(--brand);
  color: var(--text-on-accent);
  border: 1px solid var(--brand);
  border-radius: var(--radius-sm);
  padding: 6px 12px;
  font-size: 11.5px;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
  transition: background-color 120ms ease, border-color 120ms ease;
}
.btn:hover:not(:disabled) { background: var(--brand-hov); border-color: var(--brand-hov); }
.btn:disabled { opacity: 0.4; cursor: default; }

/* Transitions */
.pop-enter-active, .pop-leave-active {
  transition: opacity 140ms ease, transform 140ms ease;
}
.p-bottom.pop-enter-from, .p-bottom.pop-leave-to {
  opacity: 0; transform: translateX(-50%) translateY(-4px);
}
.p-top.pop-enter-from, .p-top.pop-leave-to {
  opacity: 0; transform: translateX(-50%) translateY(4px);
}
.p-right.pop-enter-from, .p-right.pop-leave-to {
  opacity: 0; transform: translateY(-50%) translateX(-4px);
}
.p-left.pop-enter-from, .p-left.pop-leave-to {
  opacity: 0; transform: translateY(-50%) translateX(4px);
}

.mono { font-family: var(--font-mono); font-variant-numeric: tabular-nums; }
</style>
