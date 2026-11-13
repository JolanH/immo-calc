<script setup lang="ts">
import { computed } from 'vue'
import type { PropertyResult } from '@/api/properties'
import { formatEuros } from '@/format'

// « Détail du calcul »: every intermediate figure, so the results can be
// checked by hand. Inputs echoed here come from the form.
const props = defineProps<{
  result: PropertyResult
  propertyTaxCents: number
  condoFeesCents: number
  insuranceCents: number
}>()

const depreciation = computed((): [string, number][] => {
  const building = props.result.buildingDepreciationCents ?? 0
  const furniture = props.result.furnitureDepreciationCents ?? 0
  if (building === 0 && furniture === 0) return []
  return [
    ['Amortissement du bâti (LMNP)', building],
    ['Amortissement du mobilier (LMNP)', furniture],
  ]
})

const rows = computed((): [string, number][] => [
  ["Coût d'acquisition (prix + notaire)", props.result.acquisitionCostCents],
  ['Loyers encaissés (vacance déduite)', props.result.collectedRentCents],
  ['Taxe foncière', props.propertyTaxCents],
  ['Charges de copropriété non récupérables', props.condoFeesCents],
  ['Assurance PNO', props.insuranceCents],
  ['Frais de gestion locative', props.result.agencyFeesCents],
  ['Charges non récupérées pendant la vacance', props.result.vacancyChargesCents],
  ['Charges du propriétaire', props.result.ownerChargesCents],
  ['Revenu annuel avant impôts', props.result.incomeBeforeTaxCents],
  ...depreciation.value,
  ['Revenu imposable', props.result.taxableIncomeCents],
  ['Impôt annuel (IR + prélèvements sociaux)', props.result.annualTaxCents],
  ['Revenu annuel net', props.result.netAnnualIncomeCents],
])
</script>

<template>
  <details class="card calc-detail" data-test="calc-detail">
    <summary>Détail du calcul</summary>
    <dl class="figures">
      <template v-for="[label, cents] in rows" :key="label">
        <dt>{{ label }}</dt>
        <dd>{{ formatEuros(cents) }}</dd>
      </template>
    </dl>
  </details>
</template>
