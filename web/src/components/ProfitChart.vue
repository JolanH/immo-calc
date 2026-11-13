<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Timeline } from '@/api/properties'
import { formatEuros, formatMonth, formatMonthDate } from '@/format'
import { niceRange, tickFormatter } from './chartScale'

// « Trésorerie cumulée »: −apport plus every month's cash-flow after tax, with
// the zero line and the two profitability dates marked (SPEC-property.md
// rules 16–20). One series, one axis.
const props = defineProps<{ timeline: Timeline }>()

const WIDTH = 1100
const HEIGHT = 280
const LEFT = 72
const RIGHT = 16
const TOP = 28
const BOTTOM = 28

const points = computed(() => props.timeline.points)
const n = computed(() => points.value.length)
const plotWidth = WIDTH - LEFT - RIGHT
const plotHeight = HEIGHT - TOP - BOTTOM

const ticks = computed(() =>
  niceRange(
    Math.min(...points.value.map((p) => p.cumulativeCents)),
    Math.max(...points.value.map((p) => p.cumulativeCents)),
  ),
)
const format = computed(() => tickFormatter(ticks.value))
const yMin = computed(() => ticks.value[0])
const yMax = computed(() => ticks.value.at(-1) ?? 1)

const x = (i: number) => LEFT + (n.value > 1 ? (i / (n.value - 1)) * plotWidth : plotWidth / 2)
const y = (cents: number) =>
  TOP + plotHeight - ((cents - yMin.value) / (yMax.value - yMin.value)) * plotHeight

const line = computed(() => points.value.map((p, i) => `${x(i)},${y(p.cumulativeCents)}`).join(' '))

/** One label per January, at most ~12 of them. */
const yearLabels = computed(() => {
  const januaries = points.value
    .map((p, i) => ({ i, month: p.month }))
    .filter((p) => p.month.endsWith('-01'))
  const every = Math.ceil(januaries.length / 12)
  return januaries.filter((_, k) => k % every === 0)
})

const indexOf = (month: string) => points.value.findIndex((p) => p.month === month)
const markers = computed(() =>
  [
    {
      key: 'positive',
      label: 'Cash-flow positif',
      month: props.timeline.positiveCashFlowMonth,
    },
    { key: 'payback', label: 'Apport récupéré', month: props.timeline.paybackMonth },
  ]
    .map((m) => ({ ...m, index: indexOf(m.month) }))
    .filter((m) => m.month && m.index >= 0),
)

const summary = computed(() => {
  const t = props.timeline
  const positive = t.positiveCashFlowMonth
    ? `cash-flow mensuel positif le ${formatMonthDate(t.positiveCashFlowMonth)}`
    : 'cash-flow mensuel jamais positif'
  const payback = t.paybackAtStart
    ? "apport récupéré dès l'achat"
    : t.paybackMonth
      ? `apport récupéré le ${formatMonthDate(t.paybackMonth)}`
      : 'apport non récupéré en 50 ans'
  return `Trésorerie cumulée depuis ${formatMonth(t.startMonth, 'long')} : ${positive}, ${payback}.`
})

// Hover / keyboard: one active month.
const active = ref<number | null>(null)
const activePoint = computed(() => (active.value === null ? null : points.value[active.value]))

function onPointer(event: PointerEvent) {
  const svg = (event.currentTarget as SVGElement).closest('svg')
  if (!svg) return
  const box = svg.getBoundingClientRect()
  const vx = ((event.clientX - box.left) / box.width) * WIDTH
  const i = Math.round(((vx - LEFT) / plotWidth) * (n.value - 1))
  active.value = Math.min(n.value - 1, Math.max(0, i))
}

function onKey(event: KeyboardEvent) {
  const last = n.value - 1
  const current = active.value ?? 0
  const next: Record<string, number> = {
    ArrowRight: Math.min(last, current + 12),
    ArrowLeft: Math.max(0, current - 12),
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
  <section v-if="n > 0" class="card profit-chart">
    <h2>Trésorerie cumulée</h2>
    <p class="placeholder">
      Apport de départ, puis chaque mois : revenus après impôts − mensualité du prêt.
    </p>
    <div
      class="loan-chart__frame profit-chart__frame"
      tabindex="0"
      data-test="profit-chart"
      aria-describedby="profit-chart-help"
      @focus="active = active ?? 0"
      @blur="active = null"
      @keydown="onKey"
    >
      <p id="profit-chart-help" class="visually-hidden">
        Flèches gauche et droite pour avancer ou reculer d'un an.
      </p>
      <svg
        :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
        role="img"
        :aria-label="summary"
        class="loan-chart__svg"
      >
        <title>{{ summary }}</title>
        <g class="loan-chart__grid">
          <g v-for="t in ticks" :key="t">
            <line :x1="LEFT" :x2="WIDTH - RIGHT" :y1="y(t)" :y2="y(t)" />
            <text :x="LEFT - 8" :y="y(t)" text-anchor="end" dominant-baseline="middle">
              {{ format(t) }}
            </text>
          </g>
        </g>
        <line
          class="profit-chart__zero"
          data-test="zero-line"
          :x1="LEFT"
          :x2="WIDTH - RIGHT"
          :y1="y(0)"
          :y2="y(0)"
        />
        <g v-for="m in markers" :key="m.key" :data-test="`marker-${m.key}`">
          <line
            class="profit-chart__marker-line"
            :x1="x(m.index)"
            :x2="x(m.index)"
            :y1="TOP"
            :y2="HEIGHT - BOTTOM"
          />
          <text
            class="profit-chart__marker-label"
            :x="x(m.index)"
            :y="m.key === 'positive' ? TOP - 14 : TOP - 2"
            :text-anchor="x(m.index) > WIDTH - 200 ? 'end' : 'start'"
            :dx="x(m.index) > WIDTH - 200 ? -4 : 4"
          >
            {{ m.label }} · {{ formatMonthDate(m.month) }}
          </text>
        </g>
        <polyline class="profit-chart__line" :points="line" data-test="cumulative-line" />
        <circle
          v-for="m in markers.filter((m) => m.key === 'payback')"
          :key="`dot-${m.key}`"
          class="profit-chart__dot"
          :cx="x(m.index)"
          :cy="y(points[m.index].cumulativeCents)"
          r="5"
        />
        <template v-if="activePoint && active !== null">
          <line
            class="loan-chart__crosshair"
            :x1="x(active)"
            :x2="x(active)"
            :y1="TOP"
            :y2="HEIGHT - BOTTOM"
          />
          <circle
            class="profit-chart__dot"
            :cx="x(active)"
            :cy="y(activePoint.cumulativeCents)"
            r="4"
          />
        </template>
        <text
          v-for="l in yearLabels"
          :key="l.month"
          class="loan-chart__year"
          :x="x(l.i)"
          :y="HEIGHT - 8"
          text-anchor="middle"
        >
          {{ l.month.slice(0, 4) }}
        </text>
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
        v-if="activePoint && active !== null"
        role="tooltip"
        class="loan-chart__tooltip"
        :style="{ left: `${(x(active) / WIDTH) * 100}%` }"
      >
        <p class="loan-chart__tooltip-title">{{ formatMonth(activePoint.month) }}</p>
        <p>
          <span class="loan-chart__line-key loan-chart__line-key--capital" />
          <strong>{{ formatEuros(activePoint.cumulativeCents) }}</strong> Trésorerie cumulée
        </p>
        <p>
          <strong>{{ formatEuros(activePoint.cashFlowCents) }}</strong> Cash-flow du mois
        </p>
      </div>
    </div>
  </section>
</template>
