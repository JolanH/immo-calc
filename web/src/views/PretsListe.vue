<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { ApiError } from '@/api/client'
import { listLoans, type LoanSummary } from '@/api/loans'
import { formatEuros, formatPercent } from '@/format'

const router = useRouter()
const loans = ref<LoanSummary[] | null>(null)
const error = ref('')

function years(months: number): string {
  const n = months / 12
  return `${n} ${n > 1 ? 'ans' : 'an'}`
}

onMounted(async () => {
  try {
    loans.value = await listLoans()
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : String(e)
  }
})
</script>

<template>
  <section>
    <div class="page-header">
      <h1>Prêts</h1>
      <RouterLink
        v-if="loans?.length"
        to="/prets/nouveau"
        class="button button--primary"
        data-test="new-loan"
      >
        Nouvelle simulation
      </RouterLink>
    </div>
    <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
    <div v-if="loans && loans.length === 0" class="card empty-state">
      <p>Aucune simulation de prêt.</p>
      <RouterLink to="/prets/nouveau" class="button button--primary" data-test="new-loan">
        Nouvelle simulation
      </RouterLink>
    </div>
    <div v-else-if="loans" class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th scope="col">Nom</th>
            <th scope="col" class="num">Montant</th>
            <th scope="col" class="num">Taux</th>
            <th scope="col" class="num">Durée</th>
            <th scope="col" class="num">Mensualité (assurance incluse)</th>
            <th scope="col" class="num">Coût total du crédit</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="loan in loans"
            :key="loan.id"
            class="table__row--link"
            @click="router.push(`/prets/${loan.id}`)"
          >
            <td>
              <RouterLink :to="`/prets/${loan.id}`" @click.stop>{{ loan.name }}</RouterLink>
            </td>
            <td class="num">{{ formatEuros(loan.principalCents) }}</td>
            <td class="num">{{ formatPercent(loan.annualRatePct) }}</td>
            <td class="num">{{ years(loan.durationMonths) }}</td>
            <td class="num">{{ formatEuros(loan.monthlyTotalCents) }}</td>
            <td class="num">{{ formatEuros(loan.totalCostCents) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
