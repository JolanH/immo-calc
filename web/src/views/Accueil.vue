<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { ApiError } from '@/api/client'
import { getActiveLoan, listLoans, setActiveLoan, type LoanSummary } from '@/api/loans'
import { formatEuros } from '@/format'
import { formatInputEuros } from '@/parse'

// Home page (SPEC-property.md, Home page): Biens on the left, a simplified
// list of loans with the session-only active loan on the right. Applying the
// active loan to the Biens figures comes with the property module.
const loans = ref<LoanSummary[] | null>(null)
const active = ref('') // '' = « Aucun »
const error = ref('')

/** « 200 000 € · 20 ans » */
function loanTerms(loan: LoanSummary): string {
  const years = loan.durationMonths / 12
  return `${formatInputEuros(loan.principalCents)}\u00a0€ · ${years} ${years > 1 ? 'ans' : 'an'}`
}

function showError(e: unknown) {
  error.value = e instanceof ApiError ? e.message : String(e)
}

onMounted(async () => {
  try {
    const [list, activeId] = await Promise.all([listLoans(), getActiveLoan()])
    loans.value = list
    active.value = activeId ?? ''
  } catch (e) {
    showError(e)
  }
})

async function choose(id: string) {
  const previous = active.value
  active.value = id
  error.value = ''
  try {
    active.value = (await setActiveLoan(id || null)) ?? ''
  } catch (e) {
    active.value = previous
    showError(e)
  }
}
</script>

<template>
  <section>
    <h1>Accueil</h1>
    <div class="home">
      <section class="home__biens card" data-test="home-biens">
        <h2>Biens</h2>
        <p class="placeholder">Fonctionnalité à venir</p>
      </section>

      <aside class="home__prets card" data-test="home-prets">
        <h2>Prêts</h2>
        <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
        <template v-if="loans && loans.length === 0">
          <p class="placeholder">Aucune simulation de prêt.</p>
          <RouterLink to="/prets/nouveau">Nouvelle simulation</RouterLink>
        </template>
        <fieldset v-else-if="loans" class="home__loans">
          <legend>Prêt actif</legend>
          <label
            class="home__loan"
            :class="{ 'home__loan--active': active === '' }"
            data-test="no-loan-row"
          >
            <input
              type="radio"
              name="active-loan"
              value=""
              class="visually-hidden"
              :checked="active === ''"
              @change="choose('')"
            />
            <span class="home__loan-text"><strong>Aucun</strong></span>
          </label>
          <label
            v-for="loan in loans"
            :key="loan.id"
            class="home__loan"
            :class="{ 'home__loan--active': active === loan.id }"
            data-test="loan-row"
          >
            <input
              type="radio"
              name="active-loan"
              :value="loan.id"
              class="visually-hidden"
              :checked="active === loan.id"
              @change="choose(loan.id)"
            />
            <span class="home__loan-text">
              <strong>{{ loan.name }}</strong>
              <span>{{ loanTerms(loan) }}</span>
              <small>{{ formatEuros(loan.monthlyTotalCents) }} / mois</small>
            </span>
            <RouterLink
              :to="`/prets/${loan.id}`"
              class="home__loan-open"
              :aria-label="`Voir le prêt ${loan.name}`"
              title="Voir le prêt"
            >
              ›
            </RouterLink>
          </label>
        </fieldset>
      </aside>
    </div>
  </section>
</template>
