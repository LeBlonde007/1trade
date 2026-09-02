<script setup lang="ts">
/**
 * TradeCandleChart — TradingView lightweight-charts candle + volume.
 * Client-only (suffix .client.vue) because lightweight-charts touches the DOM.
 *
 * Generates Brownian-motion OHLC data on mount and updates every 3 seconds.
 */
import { createChart, ColorType, CrosshairMode } from 'lightweight-charts'

const props = withDefaults(
  defineProps<{
    /** Symbol label shown in the header. */
    symbol?: string
    height?: number
  }>(),
  {
    symbol: 'EAI-IDX',
    height: 380,
  },
)

const container = ref<HTMLDivElement>()
let chart: ReturnType<typeof createChart> | null = null
let candleSeries: ReturnType<NonNullable<typeof chart>['addCandlestickSeries']> | null = null
let volumeSeries: ReturnType<NonNullable<typeof chart>['addHistogramSeries']> | null = null
let lastTime = Math.floor(Date.now() / 1000)
let lastPrice = 0.001005
let tickInterval: ReturnType<typeof setInterval> | null = null

const seedCandles = (count: number) => {
  const candles = []
  let t = lastTime - count * 60
  let p = 0.000998
  for (let i = 0; i < count; i++) {
    const drift = (Math.random() - 0.5) * 0.000004
    const open = p
    const high = open + Math.random() * 0.000003
    const low = open - Math.random() * 0.000003
    const close = Math.max(low, Math.min(high, open + drift))
    candles.push({ time: t as any, open, high, low, close })
    p = close
    t += 60
  }
  lastTime = t
  lastPrice = p
  return candles
}

const seedVolume = (candles: { time: number; close: number; open: number }[]) =>
  candles.map(c => ({
    time: c.time as any,
    value: 800 + Math.random() * 4000,
    color: c.close >= c.open ? 'rgba(25,195,125,0.4)' : 'rgba(239,68,68,0.4)',
  }))

onMounted(() => {
  if (!container.value) return

  // lightweight-charts wants literal color strings, not CSS vars.
  // Read the current theme tokens from the live document so the chart
  // automatically tracks any theme/token change without hardcoding.
  // ─── DESIGN-CONTRACT EXCEPTION ───────────────────────────────
  // Hex fallbacks below are NOT design values — they only apply if the
  // CSS variable fails to resolve (e.g. during SSR snapshot). The runtime
  // value always comes from tokens.css. Do not edit colors here; edit tokens.css.
  const root = getComputedStyle(document.documentElement)
  const v = (name: string, fallback = '') => (root.getPropertyValue(name).trim() || fallback)
  const POS    = v('--pos',    '#19C37D')
  const NEG    = v('--neg',    '#EF4444')
  const TXT2   = v('--text-2', '#A8A196')
  const CANVAS = v('--canvas', '#0A0A0A')
  const BRAND  = v('--brand',  '#D4AF37')
  // Match border-on-dark / grid color tokens
  const GRID   = v('--border', 'rgba(255,255,255,0.08)')

  chart = createChart(container.value, {
    width: container.value.clientWidth,
    height: props.height,
    layout: {
      background: { type: ColorType.Solid, color: 'transparent' },
      textColor: TXT2,
      fontFamily: 'JetBrains Mono, ui-monospace, monospace',
      fontSize: 11,
    },
    grid: {
      vertLines: { color: GRID },
      horzLines: { color: GRID },
    },
    rightPriceScale: { borderColor: GRID },
    timeScale: { borderColor: GRID, timeVisible: true, secondsVisible: false },
    crosshair: {
      mode: CrosshairMode.Normal,
      vertLine: { color: `${BRAND}80`, labelBackgroundColor: CANVAS },
      horzLine: { color: `${BRAND}80`, labelBackgroundColor: CANVAS },
    },
  })

  candleSeries = chart.addCandlestickSeries({
    upColor: POS,
    downColor: NEG,
    borderUpColor: POS,
    borderDownColor: NEG,
    wickUpColor: POS,
    wickDownColor: NEG,
    priceFormat: { type: 'price', precision: 6, minMove: 0.000001 },
  })

  volumeSeries = chart.addHistogramSeries({
    priceFormat: { type: 'volume' },
    priceScaleId: '',
    color: 'rgba(154, 154, 149, 0.3)',
  })

  // Position the volume series at the bottom 25% of the chart
  // @ts-expect-error — lightweight-charts API is loosely typed for priceScale config
  volumeSeries.priceScale().applyOptions({ scaleMargins: { top: 0.75, bottom: 0 } })

  const candles = seedCandles(200)
  candleSeries.setData(candles as any)
  // Apply soft volume colors derived from semantic tokens
  const POS_VOL = `${POS}66`  // hex + alpha ≈ 0.4
  const NEG_VOL = `${NEG}66`
  volumeSeries.setData(seedVolume(candles).map(v => ({
    ...v,
    color: v.color.startsWith('rgba(25') ? POS_VOL : NEG_VOL,
  })) as any)
  chart.timeScale().fitContent()

  // Live tick every 3 seconds — extend the last candle or add a new one
  tickInterval = setInterval(() => {
    const drift = (Math.random() - 0.5) * 0.000003
    const newClose = Math.max(0.00098, Math.min(0.00103, lastPrice + drift))
    const now = Math.floor(Date.now() / 1000)
    const candleTime = Math.floor(now / 60) * 60
    if (candleTime > lastTime) {
      candleSeries?.update({ time: candleTime as any, open: lastPrice, high: Math.max(lastPrice, newClose), low: Math.min(lastPrice, newClose), close: newClose })
      volumeSeries?.update({ time: candleTime as any, value: 800 + Math.random() * 4000, color: newClose >= lastPrice ? POS_VOL : NEG_VOL } as any)
      lastTime = candleTime
    } else {
      candleSeries?.update({ time: candleTime as any, open: lastPrice, high: Math.max(lastPrice, newClose), low: Math.min(lastPrice, newClose), close: newClose })
    }
    lastPrice = newClose
  }, 3000)

  // Resize on viewport changes
  const ro = new ResizeObserver(() => {
    if (container.value && chart) chart.applyOptions({ width: container.value.clientWidth })
  })
  ro.observe(container.value)

  onUnmounted(() => {
    ro.disconnect()
    if (tickInterval) clearInterval(tickInterval)
    chart?.remove()
    chart = null
  })
})
</script>

<template>
  <div ref="container" class="chart" :style="{ height: `${height}px` }" />
</template>

<style scoped>
.chart {
  width: 100%;
}
</style>
