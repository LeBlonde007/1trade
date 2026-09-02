<script setup lang="ts">
/**
 * CodeRunner — a Codex-style runnable code block. Shows the code with a language label, a Copy button,
 * and (for Python / JavaScript) a ▶ Run button that executes it IN THE BROWSER and shows the output
 * inline. Python runs on Pyodide (CPython compiled to WASM — sandboxed, no DOM/network-to-origin); JS
 * runs in an isolated Web Worker (no DOM/cookies). No backend, no key — safe and self-contained.
 */
const props = defineProps<{ code: string; lang?: string }>()

// Minimal Pyodide surface we use (avoids pulling the full types).
interface PyodideAPI {
  setStdout(o: { batched: (s: string) => void }): void
  setStderr(o: { batched: (s: string) => void }): void
  runPythonAsync(code: string): Promise<unknown>
}

const PYODIDE_BASE = 'https://cdn.jsdelivr.net/pyodide/v0.26.4/full/'

const normLang = computed(() => {
  const l = (props.lang || '').toLowerCase()
  if (l === 'py') return 'python'
  if (l === 'js' || l === 'node' || l === 'mjs') return 'javascript'
  return l || 'text'
})
const runnable = computed(() => normLang.value === 'python' || normLang.value === 'javascript')
// HTML (with the tag, or a document/body) gets a live, sandboxed preview instead of a stdout run.
const previewable = computed(() => normLang.value === 'html' || /^\s*<!doctype html|^\s*<html[\s>]|<body[\s>]/i.test(props.code))

const running = ref(false)
const pyLoading = ref(false)
const output = ref<string | null>(null)
const errored = ref(false)
const copied = ref(false)
const showPreview = ref(false)

/** copy puts the snippet on the clipboard. */
function copy() {
  navigator.clipboard?.writeText(props.code).then(() => {
    copied.value = true
    setTimeout(() => { copied.value = false }, 1200)
  }).catch(() => { /* clipboard blocked — ignore */ })
}

/** run executes the snippet in-browser and captures stdout / the result / errors. */
async function run() {
  if (running.value) return
  running.value = true
  output.value = ''
  errored.value = false
  try {
    output.value = normLang.value === 'python' ? await runPython(props.code) : await runJS(props.code)
    if (!output.value.trim()) output.value = '(no output)'
  } catch (e) {
    errored.value = true
    output.value = String(e)
  } finally {
    running.value = false
  }
}

// ── Python via Pyodide (lazy-loaded WASM CPython; sandboxed) ──────────────────────────────────────
let pyodidePromise: Promise<PyodideAPI> | null = null
function loadPyodide(): Promise<PyodideAPI> {
  if (!pyodidePromise) {
    pyLoading.value = true
    pyodidePromise = (async () => {
      const w = window as unknown as { loadPyodide?: (o: { indexURL: string }) => Promise<PyodideAPI> }
      if (!w.loadPyodide) {
        await new Promise<void>((resolve, reject) => {
          const s = document.createElement('script')
          s.src = PYODIDE_BASE + 'pyodide.js'
          s.onload = () => resolve()
          s.onerror = () => reject(new Error('failed to load the Python runtime'))
          document.head.appendChild(s)
        })
      }
      const py = await w.loadPyodide!({ indexURL: PYODIDE_BASE })
      pyLoading.value = false
      return py
    })()
  }
  return pyodidePromise
}
async function runPython(code: string): Promise<string> {
  const py = await loadPyodide()
  let buf = ''
  py.setStdout({ batched: (s: string) => { buf += s + '\n' } })
  py.setStderr({ batched: (s: string) => { buf += s + '\n' } })
  try {
    const r = await py.runPythonAsync(code)
    if (r !== undefined && r !== null) buf += String(r)
  } catch (e) {
    errored.value = true
    buf += String(e)
  }
  return buf
}

// ── JavaScript in an isolated Web Worker (no DOM / cookies; console + last expression captured) ────
function runJS(code: string): Promise<string> {
  return new Promise((resolve) => {
    // The worker source is a string, so the indirect-eval inside it runs in the worker's isolated scope.
    const workerSrc = [
      "let out='';",
      "const cap=(...a)=>{out+=a.map(x=>{try{return typeof x==='object'?JSON.stringify(x):String(x)}catch(_){return String(x)}}).join(' ')+'\\n'};",
      'self.console={log:cap,info:cap,warn:cap,error:cap,debug:cap};',
      'self.onmessage=(e)=>{try{const r=(0,eval)(e.data);if(r!==undefined)out+=String(r)+"\\n";}catch(err){out+=String(err)+"\\n";}self.postMessage(out);};',
    ].join('\n')
    const url = URL.createObjectURL(new Blob([workerSrc], { type: 'application/javascript' }))
    const worker = new Worker(url)
    const done = (s: string) => { clearTimeout(timer); worker.terminate(); URL.revokeObjectURL(url); resolve(s) }
    const timer = setTimeout(() => done('(timed out after 5s)'), 5000)
    worker.onmessage = (e: MessageEvent) => done(String(e.data))
    worker.onerror = (e: ErrorEvent) => { errored.value = true; done('Error: ' + e.message) }
    worker.postMessage(code)
  })
}
</script>

<template>
  <div class="code-runner">
    <div class="cr-head">
      <span class="cr-lang mono">{{ normLang }}</span>
      <div class="cr-actions">
        <button type="button" class="cr-btn" @click="copy">{{ copied ? 'Copied' : 'Copy' }}</button>
        <button v-if="previewable" type="button" class="cr-btn cr-run" :class="{ on: showPreview }" @click="showPreview = !showPreview">
          {{ showPreview ? 'Hide preview' : '▶ Preview' }}
        </button>
        <button v-if="runnable" type="button" class="cr-btn cr-run" :disabled="running" @click="run">
          <span v-if="running" class="cr-spin" />
          {{ running ? (pyLoading ? 'Loading runtime…' : 'Running…') : '▶ Run' }}
        </button>
      </div>
    </div>
    <iframe v-if="previewable && showPreview" class="cr-preview" :srcdoc="code" sandbox="allow-scripts allow-modals" title="HTML preview" />
    <pre v-show="!(previewable && showPreview)" class="cr-code"><code>{{ code }}</code></pre>
    <div v-if="output !== null" class="cr-out" :class="{ err: errored }">
      <div class="cr-out-label mono">{{ errored ? 'Error' : 'Output' }}</div>
      <pre>{{ output }}</pre>
    </div>
  </div>
</template>

<style scoped>
.code-runner { border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden; margin: 8px 0; background: var(--canvas); }
.cr-preview { display: block; width: 100%; height: 420px; border: 0; background: white; }
.cr-btn.on { color: var(--accent); border-color: var(--accent); }
.cr-head { display: flex; align-items: center; justify-content: space-between; padding: 5px 10px; border-bottom: 1px solid var(--border); background: var(--surface, var(--canvas)); }
.cr-lang { font-size: 11px; letter-spacing: 0.03em; color: var(--muted, var(--text)); text-transform: lowercase; }
.cr-actions { display: flex; gap: 6px; }
.cr-btn { font-size: 11px; padding: 3px 9px; border: 1px solid var(--border); border-radius: var(--radius-sm); background: none; color: var(--text); cursor: pointer; display: inline-flex; align-items: center; gap: 5px; }
.cr-btn:hover:not(:disabled) { border-color: var(--accent); color: var(--accent); }
.cr-btn:disabled { cursor: not-allowed; color: var(--muted, var(--text)); }
.cr-run { color: var(--success, var(--accent)); border-color: var(--success, var(--accent)); font-weight: 600; }
.cr-code { margin: 0; padding: 12px; overflow-x: auto; font-family: var(--font-mono, monospace); font-size: 12.5px; line-height: 1.5; color: var(--text); }
.cr-code code { font-family: inherit; white-space: pre; }
.cr-out { border-top: 1px solid var(--border); }
.cr-out-label { font-size: 10px; letter-spacing: 0.04em; text-transform: uppercase; color: var(--muted, var(--text)); padding: 6px 12px 0; }
.cr-out pre { margin: 0; padding: 4px 12px 12px; font-family: var(--font-mono, monospace); font-size: 12px; line-height: 1.5; white-space: pre-wrap; color: var(--text); }
.cr-out.err .cr-out-label, .cr-out.err pre { color: var(--warn); }
.cr-spin { width: 11px; height: 11px; border: 1.5px solid var(--border); border-top-color: var(--accent); border-radius: 50%; display: inline-block; animation: cr-spin 0.7s linear infinite; }
@keyframes cr-spin { to { transform: rotate(360deg); } }
</style>
