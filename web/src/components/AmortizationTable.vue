<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Installment, YearRow } from '@/api/loans'
import { formatEuros, formatMonth } from '@/format'

const props = defineProps<{ schedule: Installment[]; yearly: YearRow[] }>()

type View = 'annual' | 'monthly'
const view = ref<View>('annual')

interface Row {
  key: string
  label: string
  // What the borrower pays: principal + interest + insurance, so the
  // columns of a row add up.
  dueCents: number
  interestCents: number
  principalCents: number
  insuranceCents: number
  remainingCents: number
}

function yearLabel(row: YearRow): string {
  if (row.installments >= 12) return String(row.year)
  const word = row.installments > 1 ? 'échéances' : 'échéance'
  return `${row.year} (${row.installments} ${word})`
}

const rows = computed((): Row[] =>
  view.value === 'annual'
    ? props.yearly.map((r) => ({
        key: String(r.year),
        label: yearLabel(r),
        dueCents: r.paymentCents + r.insuranceCents,
        interestCents: r.interestCents,
        principalCents: r.principalCents,
        insuranceCents: r.insuranceCents,
        remainingCents: r.remainingCents,
      }))
    : props.schedule.map((r) => ({
        key: String(r.index),
        label: formatMonth(r.month),
        dueCents: r.paymentCents + r.insuranceCents,
        interestCents: r.interestCents,
        principalCents: r.principalCents,
        insuranceCents: r.insuranceCents,
        remainingCents: r.remainingCents,
      })),
)
</script>

<template>
  <section class="amortization">
    <div class="page-header">
      <h2>Tableau d'amortissement</h2>
      <div class="segmented" role="group" aria-label="Affichage du tableau">
        <button
          type="button"
          class="segmented__option"
          data-test="view-annual"
          :aria-pressed="view === 'annual' ? 'true' : 'false'"
          @click="view = 'annual'"
        >
          Annuel
        </button>
        <button
          type="button"
          class="segmented__option"
          data-test="view-monthly"
          :aria-pressed="view === 'monthly' ? 'true' : 'false'"
          @click="view = 'monthly'"
        >
          Mensuel
        </button>
      </div>
    </div>
    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th scope="col">{{ view === 'annual' ? 'Année' : 'Mois' }}</th>
            <th scope="col" class="num">Échéance</th>
            <th scope="col" class="num">Intérêts</th>
            <th scope="col" class="num">Capital remboursé</th>
            <th scope="col" class="num">Assurance</th>
            <th scope="col" class="num">Capital restant dû</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.key">
            <th scope="row">{{ row.label }}</th>
            <td class="num">{{ formatEuros(row.dueCents) }}</td>
            <td class="num">{{ formatEuros(row.interestCents) }}</td>
            <td class="num">{{ formatEuros(row.principalCents) }}</td>
            <td class="num">{{ formatEuros(row.insuranceCents) }}</td>
            <td class="num">{{ formatEuros(row.remainingCents) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
