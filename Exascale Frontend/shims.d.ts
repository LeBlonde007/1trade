/**
 * Ambient shims for dependencies that ship no type declarations.
 *
 * chartjs-adapter-date-fns registers a date adapter on Chart.js as a side effect and
 * exports nothing, but it has no `types` entry, so `await import(...)` on it raised
 * TS7016 under `vue-tsc`. Declaring the module keeps the import type-safe without
 * silencing anything else.
 */
declare module 'chartjs-adapter-date-fns'
