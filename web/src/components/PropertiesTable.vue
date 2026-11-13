<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import DpeBadge from './DpeBadge.vue'
import { REGIME_LABELS, type PropertySummary } from '@/api/properties'
import { formatEuros, formatPercent } from '@/format'

// The Biens table, shared by /biens and the home page. With `selectable`,
// a first column of checkboxes picks up to MAX_COMPARED biens to compare.
const MAX_COMPARED = 4

const props = defineProps<{
  properties: PropertySummary[]
  selectable?: boolean
  selected?: string[]
}>()
const emit = defineEmits<{ 'update:selected': [ids: string[]] }>()
const router = useRouter()

const chosen = computed(() => props.selected ?? [])

function toggle(id: string, on: boolean) {
  const rest = chosen.value.filter((x) => x !== id)
  emit('update:selected', on ? [...rest, id] : rest)
}

/** The cash-flow column only makes sense once a property has a loan. */
const showCashFlow = computed(() => props.properties.some((p) => p.cashFlowAfterTaxCents !== null))
</script>

<template>
  <div class="table-wrap">
    <table class="table">
      <thead>
        <tr>
          <th v-if="selectable" scope="col" class="check-col">
            <span class="visually-hidden">Comparer</span>
          </th>
          <th scope="col">Nom</th>
          <th scope="col" class="num">Prix</th>
          <th scope="col" class="num">Loyer</th>
          <th scope="col">DPE</th>
          <th scope="col">Régime fiscal</th>
          <th scope="col" class="num">Rentabilité nette après impôts</th>
          <th scope="col" class="num kpi">Rentabilité nette après crédit</th>
          <th scope="col" class="num">Revenus mensuels après impôts</th>
          <th v-if="showCashFlow" scope="col" class="num">Cash-flow après impôts</th>
          <th scope="col">Prêt</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="p in properties"
          :key="p.id"
          class="table__row--link"
          @click="router.push(`/biens/${p.id}`)"
        >
          <td v-if="selectable" class="check-col" @click.stop>
            <input
              type="checkbox"
              :checked="chosen.includes(p.id)"
              :disabled="!chosen.includes(p.id) && chosen.length >= MAX_COMPARED"
              :aria-label="`Comparer « ${p.name} »`"
              @change="toggle(p.id, ($event.target as HTMLInputElement).checked)"
            />
          </td>
          <td>
            <RouterLink :to="`/biens/${p.id}`" @click.stop>{{ p.name }}</RouterLink>
          </td>
          <td class="num">{{ formatEuros(p.priceCents) }}</td>
          <td class="num">{{ formatEuros(p.monthlyRentCents) }}</td>
          <td><DpeBadge :dpe="p.dpe" /></td>
          <td>{{ REGIME_LABELS[p.taxRegime] ?? '—' }}</td>
          <td class="num">{{ formatPercent(p.netAfterTaxYieldPct) }}</td>
          <td class="num kpi">{{ formatPercent(p.netAfterCreditYieldPct) }}</td>
          <td class="num">{{ formatEuros(p.monthlyNetIncomeCents) }}</td>
          <td v-if="showCashFlow" class="num">
            {{ p.cashFlowAfterTaxCents === null ? '—' : formatEuros(p.cashFlowAfterTaxCents) }}
          </td>
          <td>
            <RouterLink v-if="p.loanId" :to="`/prets/${p.loanId}`" @click.stop>
              {{ p.loanName }}
            </RouterLink>
            <span v-else>—</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
