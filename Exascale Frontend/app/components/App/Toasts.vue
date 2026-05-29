<script setup lang="ts">
/**
 * App/Toasts — global ephemeral toast region (N3).
 * Mount once in layouts/app.vue (and marketing.vue if needed).
 * Position: bottom-right stack, 12px gap. Auto-dismiss 4.5s.
 */
import { CheckCircle2, XCircle, AlertTriangle, Info, X } from 'lucide-vue-next'
import type { Toast, ToastTone } from '#imports'

const t = useToasts()

const TIMEOUT_MS = 4500
const timers = new Map<number, ReturnType<typeof setTimeout>>()
const paused = ref<Set<number>>(new Set())

function arm(toast: Toast) {
  if (timers.has(toast.id)) return
  const tm = setTimeout(() => {
    if (!paused.value.has(toast.id)) t.dismiss(toast.id)
  }, TIMEOUT_MS)
  timers.set(toast.id, tm)
}

function disarm(id: number) {
  const tm = timers.get(id)
  if (tm) { clearTimeout(tm); timers.delete(id) }
}

function pause(id: number)  { paused.value = new Set([...paused.value, id]); disarm(id) }
function resume(id: number) {
  paused.value = new Set([...paused.value].filter((x) => x !== id))
  const tx = t.toasts.value.find((x) => x.id === id)
  if (tx) arm(tx)
}

watch(() => t.toasts.value.map((x) => x.id).join(','), () => {
  t.toasts.value.forEach((toast) => {
    if (!timers.has(toast.id) && !paused.value.has(toast.id)) arm(toast)
  })
  // Clean up timers for dismissed toasts
  const live = new Set(t.toasts.value.map((x) => x.id))
  for (const [id] of timers) {
    if (!live.has(id)) disarm(id)
  }
}, { immediate: true })

onBeforeUnmount(() => {
  for (const tm of timers.values()) clearTimeout(tm)
  timers.clear()
})

function iconFor(tone: ToastTone) {
  if (tone === 'pos')  return CheckCircle2
  if (tone === 'neg')  return XCircle
  if (tone === 'warn') return AlertTriangle
  return Info
}
</script>

<template>
  <Teleport to="body">
    <div class="toast-region" role="region" aria-label="Notifications">
      <TransitionGroup name="toast" tag="div" class="toast-stack">
        <div
          v-for="toast in t.toasts.value"
          :key="toast.id"
          class="toast"
          :class="'tone-' + toast.tone"
          role="status"
          @mouseenter="pause(toast.id)"
          @mouseleave="resume(toast.id)"
        >
          <span class="rail" />
          <span class="icon"><component :is="iconFor(toast.tone)" :size="16" /></span>
          <div class="body">
            <div class="title">{{ toast.title }}</div>
            <div v-if="toast.body" class="text">{{ toast.body }}</div>
            <NuxtLink v-if="toast.href" :to="toast.href" class="cta">
              {{ toast.hrefLabel || 'View →' }}
            </NuxtLink>
          </div>
          <button class="dismiss" type="button" :aria-label="'Dismiss ' + toast.title" @click="t.dismiss(toast.id)">
            <X :size="13" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-region {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 2500;
  pointer-events: none;
}
.toast-stack {
  display: flex; flex-direction: column-reverse;
  gap: 10px;
  pointer-events: none;
}
.toast {
  position: relative;
  display: grid;
  grid-template-columns: 3px 20px 1fr 22px;
  gap: 10px;
  align-items: flex-start;
  width: 360px;
  background: var(--overlay, var(--elevated));
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  padding: 12px 12px 12px 0;
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.45);
  font-family: var(--font-sans);
  color: var(--text);
  pointer-events: auto;
}
.rail { background: var(--text-3); border-radius: 1px; align-self: stretch; }
.toast.tone-pos  .rail { background: var(--pos); }
.toast.tone-neg  .rail { background: var(--neg); }
.toast.tone-warn .rail { background: var(--warn); }
.toast.tone-info .rail { background: var(--info); }

.icon { padding-top: 1px; }
.toast.tone-pos  .icon { color: var(--pos); }
.toast.tone-neg  .icon { color: var(--neg); }
.toast.tone-warn .icon { color: var(--warn); }
.toast.tone-info .icon { color: var(--info); }

.body { min-width: 0; }
.title {
  font-size: 13px; font-weight: 600;
  color: var(--text); line-height: 1.35;
  letter-spacing: -0.005em;
}
.text  { font-size: 11.5px; color: var(--text-2); margin-top: 3px; line-height: 1.5; }
.cta {
  display: inline-block;
  margin-top: 6px;
  font-size: 11.5px; color: var(--accent);
  text-decoration: none;
}
.cta:hover { text-decoration: underline; }

.dismiss {
  width: 20px; height: 20px;
  background: transparent;
  border: none; color: var(--text-3);
  display: inline-flex; align-items: center; justify-content: center;
  cursor: pointer;
  border-radius: var(--radius-sm);
}
.dismiss:hover { color: var(--text); background: rgba(255,255,255,0.06); }

/* Transitions — slide up from right */
.toast-enter-active, .toast-leave-active {
  transition: transform 240ms cubic-bezier(0.16, 1, 0.3, 1), opacity 180ms ease;
}
.toast-enter-from { transform: translateY(20px) scale(0.98); opacity: 0; }
.toast-leave-to   { transform: translateX(20px); opacity: 0; }
</style>
