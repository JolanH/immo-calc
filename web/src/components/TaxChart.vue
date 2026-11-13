<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { YearTax } from '@/api/properties'
import { formatEuros } from '@/format'
import { niceRange, tickFormatter } from './chartScale'

// « Impôt au fil des années »: the annual tax and the net income after tax of
// each year (year 1 = the loan's first year, or the first year without a
// loan). The tax moves with the loan's interest and, under LMNP au réel, with
// the end of the depreciations. Two lines, one euro axis.
const props = defineProps<{ years: YearTax[] }>()

// The drawing takes its container's width so that text keeps its real size.
const width = ref(1100)
const frame = ref<HTMLElement | null>(null)
let observer: ResizeObserver | undefined

onMounted(() => {
  if (typeof ResizeObserver === 'undefined' || !frame.value) return
  observer = new ResizeObserver((entries) => {
    const w = Math.round(entries[0]?.contentRect.width ?? 0)
    if (w > 0) width.value = w
  })
  observer.observe(frame.value)
})
onBeforeUnmount(() => observer?.disconnect())

const HEIGHT = 260
const LEFT = 72
const RIGHT = 16
const TOP = 16
const BOTTOM = 28

const n = computed(() => props.years.length)
const plotWidth = computed(() => width.value - LEFT - RIGHT)
const plotHeight = HEIGHT - TOP - BOTTOM

const ticks = computed(() =>
  niceRange(
    Math.min(...props.years.map((p) => Math.min(p.annualTaxCents, p.netAnnualIncomeCents))),
    Math.max(...props.years.map((p) => Math.max(p.annualTaxCents, p.netAnnualIncomeCents))),
  ),
)
const format = computed(() => tickFormatter(ticks.value))
const yMin = computed(() => ticks.value[0])
const yMax = computed(() => ticks.value.at(-1) ?? 1)

const x = (i: number) =>
  LEFT + (n.value > 1 ? (i / (n.value - 1)) * plotWidth.value : plotWidth.value / 2)
const y = (cents: number) =>
  TOP + plotHeight - ((cents - yMin.value) / (yMax.value - yMin.value)) * plotHeight

const taxLine = computed(() =>
  props.years.map((p, i) => `${x(i)},${y(p.annualTaxCents)}`).join(' '),
)
const netLine = computed(() =>
  props.years.map((p, i) => `${x(i)},${y(p.netAnnualIncomeCents)}`).join(' '),
)

/** At most ~12 year labels. */
const yearLabels = computed(() => {
  const every = Math.ceil(n.value / 12)
  return props.years.map((p, i) => ({ i, year: p.year })).filter((_, i) => i % every === 0)
})

const first = computed(() => props.years[0])
const last = computed(() => props.years[n.value - 1])
const summary = computed(
  () =>
    `Impôt annuel : ${formatEuros(first.value.annualTaxCents)} la 1re année, ` +
    `${formatEuros(last.value.annualTaxCents)} la ${last.value.year}e. ` +
    `Revenu net après impôts : ${formatEuros(first.value.netAnnualIncomeCents)} puis ` +
    `${formatEuros(last.value.netAnnualIncomeCents)}.`,
)

const active = ref<number | null>(null)
const activeYear = computed(() => (active.value === null ? null : props.years[active.value]))

function onPointer(event: PointerEvent) {
  const svg = (event.currentTarget as SVGElement).closest('svg')
  if (!svg) return
  const box = svg.getBoundingClientRect()
  const vx = ((event.clientX - box.left) / box.width) * width.value
  const i = Math.round(((vx - LEFT) / plotWidth.value) * (n.value - 1))
  active.value = Math.min(n.value - 1, Math.max(0, i))
}

function onKey(event: KeyboardEvent) {
  const lastIndex = n.value - 1
  const current = active.value ?? 0
  const next: Record<string, number> = {
    ArrowRight: Math.min(lastIndex, current + 1),
    ArrowLeft: Math.max(0, current - 1),
    Home: 0,
    End: lastIndex,
  }
  if (event.key in next) {
    event.preventDefault()
    active.value = next[event.key]
  }
}
</script>

<template>
  <section v-if="n > 1" class="card tax-chart">
    <h2>Impôt au fil des années</h2>
    <p class="placeholder">
      Impôt annuel et revenu net après impôts, année par année (loyers et charges non indexés). Il
      varie avec les intérêts de l'emprunt et, en LMNP au réel, avec la fin des amortissements.
    </p>
    <ul class="loan-chart__legend" data-test="tax-chart-legend">
      <li><span class="loan-chart__key tax-chart__key--net" />Revenu net après impôts</li>
      <li><span class="loan-chart__key tax-chart__key--tax" />Impôt annuel</li>
    </ul>
    <div
      ref="frame"
      class="loan-chart__frame profit-chart__frame"
      tabindex="0"
      data-test="tax-chart"
      aria-describedby="tax-chart-help"
      @focus="active = active ?? 0"
      @blur="active = null"
      @keydown="onKey"
    >
      <p id="tax-chart-help" class="visually-hidden">
        Flèches gauche et droite pour avancer ou reculer d'un an.
      </p>
      <svg
        :viewBox="`0 0 ${width} ${HEIGHT}`"
        role="img"
        :aria-label="summary"
        class="loan-chart__svg"
      >
        <title>{{ summary }}</title>
        <g class="loan-chart__grid">
          <g v-for="t in ticks" :key="t">
            <line :x1="LEFT" :x2="width - RIGHT" :y1="y(t)" :y2="y(t)" />
            <text :x="LEFT - 8" :y="y(t)" text-anchor="end" dominant-baseline="middle">
              {{ format(t) }}
            </text>
          </g>
        </g>
        <polyline class="tax-chart__net" :points="netLine" data-test="net-line" />
        <polyline class="tax-chart__tax" :points="taxLine" data-test="tax-line" />
        <text
          v-for="l in yearLabels"
          :key="l.year"
          class="loan-chart__year"
          :x="x(l.i)"
          :y="HEIGHT - 8"
          text-anchor="middle"
        >
          {{ l.year }}
        </text>
        <template v-if="activeYear && active !== null">
          <line
            class="loan-chart__crosshair"
            :x1="x(active)"
            :x2="x(active)"
            :y1="TOP"
            :y2="HEIGHT - BOTTOM"
          />
          <circle
            class="tax-chart__dot tax-chart__dot--net"
            :cx="x(active)"
            :cy="y(activeYear.netAnnualIncomeCents)"
            r="4"
          />
          <circle
            class="tax-chart__dot tax-chart__dot--tax"
            :cx="x(active)"
            :cy="y(activeYear.annualTaxCents)"
            r="4"
          />
        </template>
        <rect
          class="loan-chart__hit"
          :x="LEFT"
          :y="TOP"
          :width="plotWidth"
          :height="plotHeight"
          @pointermove="onPointer"
          @pointerleave="active = null"
        />
      </svg>
      <div
        v-if="activeYear && active !== null"
        role="tooltip"
        class="loan-chart__tooltip"
        :style="{ left: `${(x(active) / width) * 100}%` }"
      >
        <p class="loan-chart__tooltip-title">Année {{ activeYear.year }}</p>
        <p>
          <span class="loan-chart__line-key tax-chart__key--net" />
          <strong>{{ formatEuros(activeYear.netAnnualIncomeCents) }}</strong> Revenu net
        </p>
        <p>
          <span class="loan-chart__line-key tax-chart__key--tax" />
          <strong>{{ formatEuros(activeYear.annualTaxCents) }}</strong> Impôt
        </p>
      </div>
    </div>
  </section>
</template>
