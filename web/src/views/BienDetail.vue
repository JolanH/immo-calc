<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
import MoneyInput from '@/components/MoneyInput.vue'
import ResultCard from '@/components/ResultCard.vue'
import FinancingCard from '@/components/FinancingCard.vue'
import CalcDetail from '@/components/CalcDetail.vue'
import { useDebouncedEval } from '@/composables/useDebouncedEval'
import { listLoans, type LoanSummary } from '@/api/loans'
import { evaluateProperty, type PropertyInput, type TaxRegime } from '@/api/properties'
import { formatPercent } from '@/format'

const DPE_CLASSES = ['A', 'B', 'C', 'D', 'E', 'F', 'G']
const REGIMES: { value: TaxRegime; label: string }[] = [
  { value: 'micro_foncier', label: 'Micro-foncier (location nue)' },
  { value: 'reel_foncier', label: 'Réel foncier (location nue)' },
  { value: 'micro_bic', label: 'Micro-BIC (location meublée)' },
]

// Form state: null means "not a number yet" (empty or being typed).
const form = reactive({
  name: '',
  priceCents: null as number | null,
  dpe: '',
  monthlyRentCents: null as number | null,
  monthlyRecoverableChargesCents: 0 as number | null,
  annualPropertyTaxCents: null as number | null,
  annualCondoFeesCents: 0 as number | null,
  annualLandlordInsuranceCents: 0 as number | null,
  agencyManaged: false,
  taxRegime: 'micro_foncier' as TaxRegime,
  loanId: '',
})

const loans = ref<LoanSummary[]>([])
onMounted(async () => {
  try {
    loans.value = await listLoans()
  } catch {
    loans.value = [] // the loan list is optional for evaluating
  }
})

/** What the API receives; null while the required fields are missing. */
const evaluationInput = computed((): Omit<PropertyInput, 'name'> | null => {
  if (
    form.priceCents === null ||
    form.monthlyRentCents === null ||
    form.annualPropertyTaxCents === null ||
    form.dpe === ''
  ) {
    return null
  }
  return {
    priceCents: form.priceCents,
    dpe: form.dpe,
    monthlyRentCents: form.monthlyRentCents,
    monthlyRecoverableChargesCents: form.monthlyRecoverableChargesCents ?? 0,
    annualPropertyTaxCents: form.annualPropertyTaxCents,
    annualCondoFeesCents: form.annualCondoFeesCents ?? 0,
    annualLandlordInsuranceCents: form.annualLandlordInsuranceCents ?? 0,
    agencyManaged: form.agencyManaged,
    taxRegime: form.taxRegime,
    loanId: form.loanId,
  }
})

const {
  result,
  fieldErrors: errors,
  error,
} = useDebouncedEval(evaluationInput, async (input) =>
  input === null ? null : evaluateProperty(input),
)
</script>

<template>
  <section class="detail">
    <AppBreadcrumb
      :items="[
        { label: 'Accueil', to: '/' },
        { label: 'Biens', to: '/biens' },
        { label: 'Nouveau bien' },
      ]"
    />
    <h1>Nouveau bien</h1>
    <div class="detail__layout">
      <form class="card detail__form" novalidate @submit.prevent>
        <div class="field">
          <label for="bien-nom">Nom</label>
          <input
            id="bien-nom"
            v-model="form.name"
            type="text"
            maxlength="80"
            class="input"
            placeholder="T2 Lyon 7e"
            :aria-invalid="errors.name ? 'true' : undefined"
            :aria-describedby="errors.name ? 'bien-nom-error' : undefined"
          />
          <p v-if="errors.name" id="bien-nom-error" role="alert" class="field-error">
            {{ errors.name }}
          </p>
        </div>
        <MoneyInput
          id="bien-prix"
          v-model="form.priceCents"
          label="Prix (net vendeur)"
          placeholder="150 000"
          :error="errors.priceCents"
        />
        <div class="field">
          <label for="bien-dpe">DPE</label>
          <select
            id="bien-dpe"
            v-model="form.dpe"
            class="input input--short"
            :aria-invalid="errors.dpe ? 'true' : undefined"
            :aria-describedby="errors.dpe ? 'bien-dpe-error' : undefined"
          >
            <option value="">Choisir…</option>
            <option v-for="c in DPE_CLASSES" :key="c" :value="c">{{ c }}</option>
          </select>
          <p v-if="errors.dpe" id="bien-dpe-error" role="alert" class="field-error">
            {{ errors.dpe }}
          </p>
        </div>
        <MoneyInput
          id="bien-loyer"
          v-model="form.monthlyRentCents"
          label="Loyer (hors charges, par mois)"
          placeholder="700"
          :error="errors.monthlyRentCents"
        />
        <MoneyInput
          id="bien-charges"
          v-model="form.monthlyRecoverableChargesCents"
          label="Charges récupérables (par mois)"
          :error="errors.monthlyRecoverableChargesCents"
        />
        <MoneyInput
          id="bien-taxe-fonciere"
          v-model="form.annualPropertyTaxCents"
          label="Taxe foncière (par an)"
          placeholder="900"
          :error="errors.annualPropertyTaxCents"
        />
        <MoneyInput
          id="bien-copro"
          v-model="form.annualCondoFeesCents"
          label="Charges de copropriété (par an)"
          :error="errors.annualCondoFeesCents"
        />
        <MoneyInput
          id="bien-pno"
          v-model="form.annualLandlordInsuranceCents"
          label="Assurance PNO (par an)"
          :error="errors.annualLandlordInsuranceCents"
        />
        <label class="checkbox">
          <input id="bien-agence" v-model="form.agencyManaged" type="checkbox" />
          Gestion par une agence
        </label>
        <div class="field">
          <label for="bien-regime">Régime fiscal</label>
          <select id="bien-regime" v-model="form.taxRegime" class="input">
            <option v-for="r in REGIMES" :key="r.value" :value="r.value">{{ r.label }}</option>
          </select>
          <p v-if="errors.taxRegime" role="alert" class="field-error">{{ errors.taxRegime }}</p>
        </div>
        <div class="field">
          <label for="bien-pret">Prêt</label>
          <select
            id="bien-pret"
            v-model="form.loanId"
            class="input"
            :aria-invalid="errors.loanId ? 'true' : undefined"
          >
            <option value="">Aucun</option>
            <option v-for="l in loans" :key="l.id" :value="l.id">{{ l.name }}</option>
          </select>
          <p v-if="errors.loanId" role="alert" class="field-error">{{ errors.loanId }}</p>
        </div>
      </form>

      <div class="bien-results">
        <div v-if="result?.warnings.length" data-test="warnings" class="warnings">
          <p
            v-for="w in result.warnings"
            :key="w.message"
            role="alert"
            class="warning"
            :class="`warning--${w.severity}`"
          >
            {{ w.message }}
          </p>
        </div>
        <div class="detail__results" data-test="results" aria-live="polite">
          <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
          <template v-if="result">
            <ResultCard label="Revenus mensuels bruts" :cents="result.monthlyGrossIncomeCents" />
            <ResultCard
              label="Revenus mensuels après impôts"
              :cents="result.monthlyNetIncomeCents"
              highlight
            />
            <ResultCard label="Revenu annuel net" :cents="result.netAnnualIncomeCents" />
            <div class="result-card" data-test="yields">
              <span class="result-card__label">Rentabilité</span>
              <span>Brute {{ formatPercent(result.grossYieldPct) }}</span>
              <span>Nette {{ formatPercent(result.netYieldPct) }}</span>
              <strong class="result-card__value result-card__value--accent">
                Après impôts {{ formatPercent(result.netAfterTaxYieldPct) }}
              </strong>
            </div>
          </template>
          <p v-else-if="!error" class="placeholder">
            Saisissez le prix, le loyer, la taxe foncière et la classe DPE pour voir les résultats.
          </p>
        </div>
        <FinancingCard v-if="result?.financing" :financing="result.financing" />
        <CalcDetail
          v-if="result"
          :result="result"
          :property-tax-cents="form.annualPropertyTaxCents ?? 0"
          :condo-fees-cents="form.annualCondoFeesCents ?? 0"
          :insurance-cents="form.annualLandlordInsuranceCents ?? 0"
        />
      </div>
    </div>
  </section>
</template>
