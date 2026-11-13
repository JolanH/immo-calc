<script setup lang="ts">
import type { Financing } from '@/api/properties'
import { formatEuros, formatPercent } from '@/format'

defineProps<{ financing: Financing }>()
</script>

<template>
  <section class="card financing" data-test="financing">
    <h2>Financement</h2>
    <dl class="figures">
      <dt>Mensualité (assurance incluse)</dt>
      <dd>{{ formatEuros(financing.monthlyPaymentCents) }}</dd>
      <dt>Cash-flow avant impôts</dt>
      <dd>{{ formatEuros(financing.cashFlowBeforeTaxCents) }} / mois</dd>
      <dt>Cash-flow après impôts</dt>
      <dd>
        <strong>{{ formatEuros(financing.cashFlowAfterTaxCents) }}</strong> / mois
      </dd>
      <dt>Apport</dt>
      <dd>{{ formatEuros(financing.ownContributionCents) }}</dd>
      <dt>Rendement sur apport</dt>
      <dd>
        {{
          financing.returnOnContributionPct === null
            ? 'n/a'
            : formatPercent(financing.returnOnContributionPct)
        }}
      </dd>
    </dl>
  </section>
</template>
