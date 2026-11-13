<script setup lang="ts">
import { computed, reactive } from 'vue'
import MoneyInput from '@/components/MoneyInput.vue'
import RateInput from '@/components/RateInput.vue'
import MonthPicker from '@/components/MonthPicker.vue'
import ResultCard from '@/components/ResultCard.vue'
import { useDebouncedEval } from '@/composables/useDebouncedEval'
import { simulateLoan, type LoanInput } from '@/api/loans'

const DEFAULT_YEARS = 20

/** The month after today, as "YYYY-MM". */
function nextMonth(today = new Date()): string {
  const d = new Date(today.getFullYear(), today.getMonth() + 1, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

// Form state: null means "not a number yet" (empty or being typed).
const form = reactive({
  name: '',
  principalCents: null as number | null,
  annualRatePct: null as number | null,
  durationYears: DEFAULT_YEARS as number | string,
  insuranceRatePct: 0 as number | null,
  applicationFeeCents: 0 as number | null,
  guaranteeFeeCents: 0 as number | null,
  firstPaymentMonth: nextMonth(),
})

/** What the API receives; null while the required numbers are missing. */
const simulationInput = computed((): Omit<LoanInput, 'name'> | null => {
  if (form.principalCents === null || form.annualRatePct === null) return null
  const years = Number(form.durationYears)
  return {
    principalCents: form.principalCents,
    annualRatePct: form.annualRatePct,
    durationMonths: Number.isFinite(years) ? Math.round(years * 12) : 0,
    insuranceRatePct: form.insuranceRatePct ?? 0,
    applicationFeeCents: form.applicationFeeCents ?? 0,
    guaranteeFeeCents: form.guaranteeFeeCents ?? 0,
    firstPaymentMonth: form.firstPaymentMonth,
  }
})

const { result, fieldErrors, error } = useDebouncedEval(simulationInput, async (input) =>
  input === null ? null : simulateLoan(input),
)
</script>

<template>
  <section class="detail">
    <h1>Nouvelle simulation de prêt</h1>
    <div class="detail__layout">
      <form class="card detail__form" novalidate @submit.prevent>
        <div class="field">
          <label for="pret-nom">Nom</label>
          <input
            id="pret-nom"
            v-model="form.name"
            type="text"
            maxlength="80"
            class="input"
            placeholder="Appart Lyon — Crédit Mutuel"
          />
        </div>
        <MoneyInput
          id="pret-montant"
          v-model="form.principalCents"
          label="Montant"
          placeholder="200 000"
          :error="fieldErrors.principalCents"
        />
        <RateInput
          id="pret-taux"
          v-model="form.annualRatePct"
          label="Taux nominal"
          placeholder="3,5"
          :error="fieldErrors.annualRatePct"
        />
        <div class="field">
          <label for="pret-duree">Durée</label>
          <div class="field__control">
            <input
              id="pret-duree"
              v-model="form.durationYears"
              type="number"
              min="1"
              max="30"
              step="1"
              class="input input--short"
              :aria-invalid="fieldErrors.durationMonths ? 'true' : undefined"
              :aria-describedby="fieldErrors.durationMonths ? 'pret-duree-error' : undefined"
            />
            <span class="field__suffix">ans</span>
          </div>
          <p
            v-if="fieldErrors.durationMonths"
            id="pret-duree-error"
            role="alert"
            class="field-error"
          >
            {{ fieldErrors.durationMonths }}
          </p>
        </div>
        <MonthPicker
          id="pret-echeance"
          v-model="form.firstPaymentMonth"
          label="Première échéance"
          :error="fieldErrors.firstPaymentMonth"
        />
        <details class="detail__more">
          <summary>Assurance et frais</summary>
          <RateInput
            id="pret-assurance"
            v-model="form.insuranceRatePct"
            label="Taux assurance (du capital initial, par an)"
            :error="fieldErrors.insuranceRatePct"
          />
          <MoneyInput
            id="pret-frais-dossier"
            v-model="form.applicationFeeCents"
            label="Frais de dossier"
            :error="fieldErrors.applicationFeeCents"
          />
          <MoneyInput
            id="pret-frais-garantie"
            v-model="form.guaranteeFeeCents"
            label="Frais de garantie"
            :error="fieldErrors.guaranteeFeeCents"
          />
        </details>
      </form>

      <div class="detail__results" data-test="results" aria-live="polite">
        <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
        <template v-if="result">
          <ResultCard label="Mensualité hors assurance" :cents="result.monthlyPaymentCents" />
          <ResultCard label="Assurance mensuelle" :cents="result.monthlyInsuranceCents" />
          <ResultCard label="Mensualité totale" :cents="result.monthlyTotalCents" highlight />
          <ResultCard label="Intérêts totaux" :cents="result.totalInterestCents" />
          <ResultCard label="Coût de l'assurance" :cents="result.totalInsuranceCents" />
          <ResultCard label="Frais" :cents="result.totalFeesCents" />
          <ResultCard label="Coût total du crédit" :cents="result.totalCostCents" highlight />
          <ResultCard label="Montant total remboursé" :cents="result.totalRepaidCents" />
        </template>
        <p v-else-if="!error" class="placeholder">
          Saisissez le montant et le taux pour voir les résultats.
        </p>
      </div>
    </div>
  </section>
</template>
