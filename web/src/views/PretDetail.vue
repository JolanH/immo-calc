<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MoneyInput from '@/components/MoneyInput.vue'
import RateInput from '@/components/RateInput.vue'
import MonthPicker from '@/components/MonthPicker.vue'
import ResultCard from '@/components/ResultCard.vue'
import AmortizationTable from '@/components/AmortizationTable.vue'
import { useDebouncedEval } from '@/composables/useDebouncedEval'
import { UNSAVED_MESSAGE, useUnsavedGuard } from '@/composables/useUnsavedGuard'
import { ApiError } from '@/api/client'
import {
  createLoan,
  deleteLoan,
  duplicateLoan,
  getLoan,
  listLoans,
  simulateLoan,
  updateLoan,
  type LoanDetail,
  type LoanInput,
} from '@/api/loans'

const DEFAULT_YEARS = 20

/** The month after today, as "YYYY-MM". */
function nextMonth(today = new Date()): string {
  const d = new Date(today.getFullYear(), today.getMonth() + 1, 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

const route = useRoute()
const router = useRouter()

/** The saved loan shown, or null for a new simulation. */
const routeId = computed(() => {
  const id = route.params.id
  return typeof id === 'string' && id !== 'nouveau' ? id : null
})
const loadedId = ref<string | null>(null)
const savedName = ref('')
const loadError = ref('')
const saveError = ref('')
const saveFieldErrors = ref<Record<string, string>>({})
const status = ref('')
const saving = ref(false)

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

// What was last loaded or saved, to detect unsaved changes.
const snapshot = ref(JSON.stringify(form))
const dirty = computed(() => JSON.stringify(form) !== snapshot.value)
useUnsavedGuard(dirty)

function durationMonths(): number {
  const years = Number(form.durationYears)
  return Number.isFinite(years) ? Math.round(years * 12) : 0
}

/** What the simulation receives; null while the required numbers are missing. */
const simulationInput = computed((): Omit<LoanInput, 'name'> | null => {
  if (form.principalCents === null || form.annualRatePct === null) return null
  return {
    principalCents: form.principalCents,
    annualRatePct: form.annualRatePct,
    durationMonths: durationMonths(),
    insuranceRatePct: form.insuranceRatePct ?? 0,
    applicationFeeCents: form.applicationFeeCents ?? 0,
    guaranteeFeeCents: form.guaranteeFeeCents ?? 0,
    firstPaymentMonth: form.firstPaymentMonth,
  }
})

const { result, fieldErrors, error, evaluateNow } = useDebouncedEval(
  simulationInput,
  async (input) => (input === null ? null : simulateLoan(input)),
)

/** Errors from the live simulation and from the last save attempt. */
const errors = computed(() => ({ ...fieldErrors.value, ...saveFieldErrors.value }))

function fill(loan: LoanDetail) {
  Object.assign(form, {
    name: loan.name,
    principalCents: loan.principalCents,
    annualRatePct: loan.annualRatePct,
    durationYears: loan.durationMonths / 12,
    insuranceRatePct: loan.insuranceRatePct,
    applicationFeeCents: loan.applicationFeeCents,
    guaranteeFeeCents: loan.guaranteeFeeCents,
    firstPaymentMonth: loan.firstPaymentMonth,
  })
  snapshot.value = JSON.stringify(form)
  loadedId.value = loan.id
  savedName.value = loan.name
}

async function load(id: string | null) {
  if (id !== null && id === loadedId.value) return // just created: already shown
  loadError.value = ''
  status.value = ''
  if (id === null) {
    // New simulation: propose « Prêt N ».
    try {
      const loans = await listLoans()
      form.name = `Prêt ${(Array.isArray(loans) ? loans.length : 0) + 1}`
    } catch {
      form.name = 'Prêt 1'
    }
    snapshot.value = JSON.stringify(form)
    return
  }
  try {
    fill(await getLoan(id))
    await evaluateNow()
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : String(e)
  }
}

watch(routeId, (id) => void load(id), { immediate: true })
watch(form, () => (saveFieldErrors.value = {}), { deep: true })

async function save() {
  saving.value = true
  saveError.value = ''
  status.value = ''
  const input: LoanInput = {
    name: form.name,
    principalCents: form.principalCents ?? 0,
    annualRatePct: form.annualRatePct ?? 0,
    durationMonths: durationMonths(),
    insuranceRatePct: form.insuranceRatePct ?? 0,
    applicationFeeCents: form.applicationFeeCents ?? 0,
    guaranteeFeeCents: form.guaranteeFeeCents ?? 0,
    firstPaymentMonth: form.firstPaymentMonth,
  }
  try {
    const saved = loadedId.value ? await updateLoan(loadedId.value, input) : await createLoan(input)
    const created = loadedId.value === null
    fill(saved)
    status.value = 'Prêt enregistré'
    if (created) await router.replace(`/prets/${saved.id}`)
  } catch (e) {
    if (e instanceof ApiError && Object.keys(e.fields).length > 0) {
      saveFieldErrors.value = e.fields
    } else {
      saveError.value = e instanceof ApiError ? e.message : String(e)
    }
  } finally {
    saving.value = false
  }
}

function showActionError(e: unknown) {
  saveError.value = e instanceof ApiError ? e.message : String(e)
}

async function duplicate() {
  if (!loadedId.value) return
  if (dirty.value && !window.confirm(UNSAVED_MESSAGE)) return
  saveError.value = ''
  status.value = ''
  try {
    const copy = await duplicateLoan(loadedId.value)
    fill(copy) // shown right away; the route change below won't reload it
    await router.push(`/prets/${copy.id}`)
    await evaluateNow()
    status.value = 'Copie créée'
  } catch (e) {
    showActionError(e)
  }
}

async function remove() {
  if (!loadedId.value) return
  if (!window.confirm(`Supprimer le prêt « ${savedName.value} » ? Cette action est définitive.`))
    return
  saveError.value = ''
  status.value = ''
  try {
    await deleteLoan(loadedId.value)
    snapshot.value = JSON.stringify(form) // deleted: nothing left to save
    await router.push('/prets')
  } catch (e) {
    showActionError(e)
  }
}
</script>

<template>
  <section class="detail">
    <h1>{{ loadedId ? savedName : 'Nouvelle simulation de prêt' }}</h1>
    <p v-if="loadError" role="alert" class="alert alert--error">{{ loadError }}</p>
    <div v-else class="detail__layout">
      <form class="card detail__form" novalidate @submit.prevent="save">
        <div class="field">
          <label for="pret-nom">Nom</label>
          <input
            id="pret-nom"
            v-model="form.name"
            type="text"
            maxlength="80"
            class="input"
            placeholder="Appart Lyon — Crédit Mutuel"
            :aria-invalid="errors.name ? 'true' : undefined"
            :aria-describedby="errors.name ? 'pret-nom-error' : undefined"
          />
          <p v-if="errors.name" id="pret-nom-error" role="alert" class="field-error">
            {{ errors.name }}
          </p>
        </div>
        <MoneyInput
          id="pret-montant"
          v-model="form.principalCents"
          label="Montant"
          placeholder="200 000"
          :error="errors.principalCents"
        />
        <RateInput
          id="pret-taux"
          v-model="form.annualRatePct"
          label="Taux nominal"
          placeholder="3,5"
          :error="errors.annualRatePct"
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
              :aria-invalid="errors.durationMonths ? 'true' : undefined"
              :aria-describedby="errors.durationMonths ? 'pret-duree-error' : undefined"
            />
            <span class="field__suffix">ans</span>
          </div>
          <p v-if="errors.durationMonths" id="pret-duree-error" role="alert" class="field-error">
            {{ errors.durationMonths }}
          </p>
        </div>
        <MonthPicker
          id="pret-echeance"
          v-model="form.firstPaymentMonth"
          label="Première échéance"
          :error="errors.firstPaymentMonth"
        />
        <details class="detail__more">
          <summary>Assurance et frais</summary>
          <RateInput
            id="pret-assurance"
            v-model="form.insuranceRatePct"
            label="Taux assurance (du capital initial, par an)"
            :error="errors.insuranceRatePct"
          />
          <MoneyInput
            id="pret-frais-dossier"
            v-model="form.applicationFeeCents"
            label="Frais de dossier"
            :error="errors.applicationFeeCents"
          />
          <MoneyInput
            id="pret-frais-garantie"
            v-model="form.guaranteeFeeCents"
            label="Frais de garantie"
            :error="errors.guaranteeFeeCents"
          />
        </details>
        <p v-if="status" role="status" class="alert alert--success">{{ status }}</p>
        <p v-if="saveError" role="alert" class="alert alert--error">{{ saveError }}</p>
        <div class="actions">
          <button type="submit" class="button button--primary" data-test="save" :disabled="saving">
            Enregistrer
          </button>
          <template v-if="loadedId">
            <button type="button" class="button" data-test="duplicate" @click="duplicate">
              Dupliquer
            </button>
            <button type="button" class="button button--danger" data-test="delete" @click="remove">
              Supprimer
            </button>
          </template>
        </div>
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
    <AmortizationTable
      v-if="!loadError && result"
      :schedule="result.schedule"
      :yearly="result.yearly"
    />
  </section>
</template>
