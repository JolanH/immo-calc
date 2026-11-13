<script setup lang="ts">
import { computed, ref } from 'vue'
import type { YearRow } from '@/api/loans'
import { formatEuros } from '@/format'
import { niceTicks, tickFormatter } from './chartScale'

// Two charts sharing the same year bands (SPEC-loan.md): stacked yearly bars
// (capital + interest) on top, remaining principal as a line below. Two
// charts rather than one with a second y-axis, so heights stay comparable.
const props = defineProps<{ yearly: YearRow[] }>()

const WIDTH = 1100
const LEFT = 64
const RIGHT = 16
const TOP = 12
const BOTTOM = 28
const BARS_HEIGHT = 300
const LINE_HEIGHT = 200
const MAX_BAR = 24
const GAP = 2
const RADIUS = 4

const plotWidth = WIDTH - LEFT - RIGHT
const n = computed(() => props.yearly.length)
const band = computed(() => plotWidth / Math.max(n.value, 1))
const barWidth = computed(() => Math.min(MAX_BAR, band.value * 0.6))
const center = (i: number) => LEFT + band.value * (i + 0.5)

/** Show at most ~12 year labels. */
const labelEvery = computed(() => Math.ceil(n.value / 12))

function axis(max: number, height: number) {
  const ticks = niceTicks(max)
  const top = ticks.at(-1) ?? 1
  const plotHeight = height - TOP - BOTTOM
  return {
    ticks,
    format: tickFormatter(ticks),
    baseline: height - BOTTOM,
    y: (v: number) => height - BOTTOM - (v / top) * plotHeight,
    scale: (v: number) => (v / top) * plotHeight,
  }
}

const barAxis = computed(() =>
  axis(Math.max(0, ...props.yearly.map((r) => r.principalCents + r.interestCents)), BARS_HEIGHT),
)
const lineAxis = computed(() =>
  axis(Math.max(0, ...props.yearly.map((r) => r.remainingCents)), LINE_HEIGHT),
)

/** Rectangle path with rounded top corners (the data end), square at the baseline. */
function roundedTop(x: number, y: number, w: number, h: number): string {
  const r = Math.min(RADIUS, h, w / 2)
  return `M${x},${y + h}V${y + r}Q${x},${y} ${x + r},${y}H${x + w - r}Q${x + w},${y} ${x + w},${y + r}V${y + h}Z`
}

const bars = computed(() =>
  props.yearly.map((r, i) => {
    const a = barAxis.value
    const w = barWidth.value
    const x = center(i) - w / 2
    const hPrincipal = a.scale(r.principalCents)
    const hInterest = a.scale(r.interestCents)
    const principalTop = a.baseline - hPrincipal
    // The top segment carries the rounded end; the 2px surface gap is taken
    // from it so the principal segment keeps its exact height.
    const interestDrawn = Math.max(0, hInterest - GAP)
    return {
      key: r.year,
      hPrincipal,
      hInterest,
      principalPath:
        hInterest > 0
          ? `M${x},${a.baseline}V${principalTop}H${x + w}V${a.baseline}Z`
          : roundedTop(x, principalTop, w, hPrincipal),
      interestPath:
        interestDrawn > 0 ? roundedTop(x, principalTop - hInterest, w, interestDrawn) : '',
    }
  }),
)

const linePoints = computed(() =>
  props.yearly.map((r, i) => `${center(i)},${lineAxis.value.y(r.remainingCents)}`).join(' '),
)

function yearLabel(r: YearRow): string {
  if (r.installments >= 12) return String(r.year)
  return `${r.year} (${r.installments} ${r.installments > 1 ? 'échéances' : 'échéance'})`
}

const range = computed(() =>
  n.value ? `de ${props.yearly[0].year} à ${props.yearly[n.value - 1].year}` : '',
)

// Hover / keyboard focus: one active year, shown on both charts.
const active = ref<number | null>(null)
const activeRow = computed(() => (active.value === null ? null : props.yearly[active.value]))

function onKey(event: KeyboardEvent) {
  const last = n.value - 1
  const current = active.value ?? 0
  const next: Record<string, number> = {
    ArrowRight: Math.min(last, current + 1),
    ArrowLeft: Math.max(0, current - 1),
    Home: 0,
    End: last,
  }
  if (event.key in next) {
    event.preventDefault()
    active.value = next[event.key]
  }
}
</script>

<template>
  <section v-if="n > 0" class="loan-chart">
    <div class="page-header">
      <h2>Remboursement par année</h2>
      <ul class="loan-chart__legend" data-test="legend">
        <li><span class="loan-chart__key loan-chart__key--capital" />Capital remboursé</li>
        <li><span class="loan-chart__key loan-chart__key--interest" />Intérêts</li>
      </ul>
    </div>
    <div
      class="loan-chart__frame"
      tabindex="0"
      data-test="chart"
      aria-describedby="loan-chart-help"
      @focus="active = active ?? 0"
      @blur="active = null"
      @keydown="onKey"
    >
      <p id="loan-chart-help" class="visually-hidden">
        Flèches gauche et droite pour parcourir les années. Le détail complet est dans le tableau
        d'amortissement.
      </p>
      <svg
        :viewBox="`0 0 ${WIDTH} ${BARS_HEIGHT}`"
        role="img"
        :aria-label="`Capital remboursé et intérêts par année, ${range}`"
        class="loan-chart__svg"
      >
        <title>Remboursement par année {{ range }}</title>
        <g class="loan-chart__grid">
          <g v-for="t in barAxis.ticks" :key="t">
            <line :x1="LEFT" :x2="WIDTH - RIGHT" :y1="barAxis.y(t)" :y2="barAxis.y(t)" />
            <text :x="LEFT - 8" :y="barAxis.y(t)" text-anchor="end" dominant-baseline="middle">
              {{ barAxis.format(t) }}
            </text>
          </g>
        </g>
        <g v-for="(b, i) in bars" :key="b.key" data-test="bar">
          <path
            class="loan-chart__capital"
            :d="b.principalPath"
            data-test="segment-principal"
            :data-height="b.hPrincipal"
          />
          <path
            class="loan-chart__interest"
            :d="b.interestPath"
            data-test="segment-interest"
            :data-height="b.hInterest"
          />
          <text
            v-if="i % labelEvery === 0"
            class="loan-chart__year"
            :x="center(i)"
            :y="BARS_HEIGHT - 8"
            text-anchor="middle"
          >
            {{ yearly[i].year }}
          </text>
        </g>
        <line
          class="loan-chart__baseline"
          :x1="LEFT"
          :x2="WIDTH - RIGHT"
          :y1="barAxis.baseline"
          :y2="barAxis.baseline"
        />
        <rect
          v-for="(r, i) in yearly"
          :key="`hit-${r.year}`"
          data-test="hit"
          class="loan-chart__hit"
          :class="{ 'loan-chart__hit--active': active === i }"
          :x="LEFT + band * i"
          :y="TOP"
          :width="band"
          :height="BARS_HEIGHT - TOP - BOTTOM"
          @pointerenter="active = i"
          @pointerleave="active = null"
        />
      </svg>

      <h3 class="loan-chart__subtitle">Capital restant dû</h3>
      <svg
        :viewBox="`0 0 ${WIDTH} ${LINE_HEIGHT}`"
        role="img"
        :aria-label="`Capital restant dû en fin d'année, ${range}`"
        class="loan-chart__svg"
      >
        <title>Capital restant dû {{ range }}</title>
        <g class="loan-chart__grid">
          <g v-for="t in lineAxis.ticks" :key="t">
            <line :x1="LEFT" :x2="WIDTH - RIGHT" :y1="lineAxis.y(t)" :y2="lineAxis.y(t)" />
            <text :x="LEFT - 8" :y="lineAxis.y(t)" text-anchor="end" dominant-baseline="middle">
              {{ lineAxis.format(t) }}
            </text>
          </g>
        </g>
        <line
          class="loan-chart__baseline"
          :x1="LEFT"
          :x2="WIDTH - RIGHT"
          :y1="lineAxis.baseline"
          :y2="lineAxis.baseline"
        />
        <template v-if="active !== null && activeRow">
          <line
            class="loan-chart__crosshair"
            :x1="center(active)"
            :x2="center(active)"
            :y1="TOP"
            :y2="lineAxis.baseline"
          />
        </template>
        <polyline class="loan-chart__remaining" :points="linePoints" data-test="remaining-line" />
        <circle
          v-if="active !== null && activeRow"
          class="loan-chart__marker"
          :cx="center(active)"
          :cy="lineAxis.y(activeRow.remainingCents)"
          r="4"
        />
        <template v-for="(r, i) in yearly" :key="`ly-${r.year}`">
          <text
            v-if="i % labelEvery === 0"
            class="loan-chart__year"
            :x="center(i)"
            :y="LINE_HEIGHT - 8"
            text-anchor="middle"
          >
            {{ r.year }}
          </text>
        </template>
        <rect
          v-for="(r, i) in yearly"
          :key="`lhit-${r.year}`"
          class="loan-chart__hit"
          :x="LEFT + band * i"
          :y="TOP"
          :width="band"
          :height="LINE_HEIGHT - TOP - BOTTOM"
          @pointerenter="active = i"
          @pointerleave="active = null"
        />
      </svg>

      <div
        v-if="active !== null && activeRow"
        role="tooltip"
        class="loan-chart__tooltip"
        :style="{ left: `${(center(active) / WIDTH) * 100}%` }"
      >
        <p class="loan-chart__tooltip-title">{{ yearLabel(activeRow) }}</p>
        <p>
          <span class="loan-chart__line-key loan-chart__line-key--capital" />
          <strong>{{ formatEuros(activeRow.principalCents) }}</strong> Capital remboursé
        </p>
        <p>
          <span class="loan-chart__line-key loan-chart__line-key--interest" />
          <strong>{{ formatEuros(activeRow.interestCents) }}</strong> Intérêts
        </p>
        <p>
          <span class="loan-chart__line-key loan-chart__line-key--remaining" />
          <strong>{{ formatEuros(activeRow.remainingCents) }}</strong> Capital restant dû
        </p>
      </div>
    </div>
  </section>
</template>
