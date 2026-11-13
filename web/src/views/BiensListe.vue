<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
import DpeBadge from '@/components/DpeBadge.vue'
import { ApiError } from '@/api/client'
import { listProperties, type PropertySummary } from '@/api/properties'
import { formatEuros, formatPercent } from '@/format'

const router = useRouter()
const properties = ref<PropertySummary[] | null>(null)
const error = ref('')

/** The cash-flow column only makes sense once a property has a loan. */
const showCashFlow = computed(() =>
  (properties.value ?? []).some((p) => p.cashFlowAfterTaxCents !== null),
)

onMounted(async () => {
  try {
    properties.value = await listProperties()
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : String(e)
  }
})
</script>

<template>
  <section>
    <AppBreadcrumb :items="[{ label: 'Accueil', to: '/' }, { label: 'Biens' }]" />
    <div class="page-header">
      <h1>Biens</h1>
      <RouterLink
        v-if="properties?.length"
        to="/biens/nouveau"
        class="button button--primary"
        data-test="new-property"
      >
        Nouveau bien
      </RouterLink>
    </div>
    <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
    <div v-if="properties && properties.length === 0" class="card empty-state">
      <p>Aucun bien enregistré.</p>
      <RouterLink to="/biens/nouveau" class="button button--primary" data-test="new-property">
        Nouveau bien
      </RouterLink>
    </div>
    <div v-else-if="properties" class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th scope="col">Nom</th>
            <th scope="col" class="num">Prix</th>
            <th scope="col" class="num">Loyer</th>
            <th scope="col">DPE</th>
            <th scope="col" class="num">Rentabilité nette après impôts</th>
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
            <td>
              <RouterLink :to="`/biens/${p.id}`" @click.stop>{{ p.name }}</RouterLink>
            </td>
            <td class="num">{{ formatEuros(p.priceCents) }}</td>
            <td class="num">{{ formatEuros(p.monthlyRentCents) }}</td>
            <td><DpeBadge :dpe="p.dpe" /></td>
            <td class="num">{{ formatPercent(p.netAfterTaxYieldPct) }}</td>
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
  </section>
</template>
