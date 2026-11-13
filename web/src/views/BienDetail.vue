<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
import MoneyInput from '@/components/MoneyInput.vue'
import ResultCard from '@/components/ResultCard.vue'
import FinancingCard from '@/components/FinancingCard.vue'
import CalcDetail from '@/components/CalcDetail.vue'
import ProfitChart from '@/components/ProfitChart.vue'
import RegimeSection from '@/components/RegimeSection.vue'
import { useDebouncedEval } from '@/composables/useDebouncedEval'
import { UNSAVED_MESSAGE, useUnsavedGuard } from '@/composables/useUnsavedGuard'
import { ApiError } from '@/api/client'
import { listLoans, type LoanSummary } from '@/api/loans'
import {
  createProperty,
  deleteProperty,
  duplicateProperty,
  evaluateProperty,
  getProperty,
  updateProperty,
  type PropertyDetail,
  type PropertyInput,
  type TaxRegime,
} from '@/api/properties'
import { formatPercent } from '@/format'
import { paybackDate, positiveDate } from '@/profitDates'

const DPE_CLASSES = ['A', 'B', 'C', 'D', 'E', 'F', 'G']
const REGIMES: { value: TaxRegime; label: string }[] = [
  { value: 'micro_foncier', label: 'Micro-foncier (location nue)' },
  { value: 'reel_foncier', label: 'Réel foncier (location nue)' },
  { value: 'micro_bic', label: 'Micro-BIC (location meublée)' },
  { value: 'lmnp_reel', label: 'LMNP au réel (location meublée)' },
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
  furnitureIncludedCents: 0 as number | null,
  furnitureAddedCents: 0 as number | null,
  agencyManaged: false,
  taxRegime: 'micro_foncier' as TaxRegime,
  loanId: '',
})

const route = useRoute()
const router = useRouter()

/** The saved property shown, or null for a new one (/biens/nouveau). */
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

// What was last loaded or saved, to detect unsaved changes.
const snapshot = ref(JSON.stringify(form))
const dirty = computed(() => JSON.stringify(form) !== snapshot.value)
useUnsavedGuard(dirty)

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
    furnitureIncludedCents: form.furnitureIncludedCents ?? 0,
    furnitureAddedCents: form.furnitureAddedCents ?? 0,
    agencyManaged: form.agencyManaged,
    taxRegime: form.taxRegime,
    loanId: form.loanId,
  }
})

const { result, fieldErrors, error, evaluateNow } = useDebouncedEval(
  evaluationInput,
  async (input) => (input === null ? null : evaluateProperty(input, loadedId.value ?? undefined)),
)

/** Errors from the live evaluation and from the last save attempt. */
const errors = computed(() => ({ ...fieldErrors.value, ...saveFieldErrors.value }))

function fill(p: PropertyDetail) {
  Object.assign(form, {
    name: p.name,
    priceCents: p.priceCents,
    dpe: p.dpe,
    monthlyRentCents: p.monthlyRentCents,
    monthlyRecoverableChargesCents: p.monthlyRecoverableChargesCents,
    annualPropertyTaxCents: p.annualPropertyTaxCents,
    annualCondoFeesCents: p.annualCondoFeesCents,
    annualLandlordInsuranceCents: p.annualLandlordInsuranceCents,
    furnitureIncludedCents: p.furnitureIncludedCents ?? 0,
    furnitureAddedCents: p.furnitureAddedCents ?? 0,
    agencyManaged: p.agencyManaged,
    taxRegime: p.taxRegime,
    loanId: p.loanId,
  })
  snapshot.value = JSON.stringify(form)
  loadedId.value = p.id
  savedName.value = p.name
}

async function load(id: string | null) {
  if (id !== null && id === loadedId.value) return // just created or copied: already shown
  loadError.value = ''
  status.value = ''
  if (id === null) return
  try {
    fill(await getProperty(id))
    await evaluateNow()
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : String(e)
  }
}

watch(routeId, (id) => void load(id), { immediate: true })
watch(form, () => (saveFieldErrors.value = {}), { deep: true })

function showActionError(e: unknown) {
  if (e instanceof ApiError && Object.keys(e.fields).length > 0) saveFieldErrors.value = e.fields
  else saveError.value = e instanceof ApiError ? e.message : String(e)
}

function clearMessages() {
  saveError.value = ''
  status.value = ''
}

async function save() {
  clearMessages()
  saving.value = true
  const input: PropertyInput = {
    name: form.name,
    priceCents: form.priceCents ?? 0,
    dpe: form.dpe,
    monthlyRentCents: form.monthlyRentCents ?? 0,
    monthlyRecoverableChargesCents: form.monthlyRecoverableChargesCents ?? 0,
    annualPropertyTaxCents: form.annualPropertyTaxCents ?? 0,
    annualCondoFeesCents: form.annualCondoFeesCents ?? 0,
    annualLandlordInsuranceCents: form.annualLandlordInsuranceCents ?? 0,
    furnitureIncludedCents: form.furnitureIncludedCents ?? 0,
    furnitureAddedCents: form.furnitureAddedCents ?? 0,
    agencyManaged: form.agencyManaged,
    taxRegime: form.taxRegime,
    loanId: form.loanId,
  }
  try {
    const created = loadedId.value === null
    const saved = created
      ? await createProperty(input)
      : await updateProperty(loadedId.value as string, input)
    fill(saved)
    status.value = 'Bien enregistré'
    if (created) await router.replace(`/biens/${saved.id}`)
  } catch (e) {
    showActionError(e)
  } finally {
    saving.value = false
  }
}

async function duplicate() {
  if (!loadedId.value) return
  if (dirty.value && !window.confirm(UNSAVED_MESSAGE)) return
  clearMessages()
  try {
    const copy = await duplicateProperty(loadedId.value)
    fill(copy) // shown right away; the route change below won't reload it
    await router.push(`/biens/${copy.id}`)
    await evaluateNow()
    status.value = 'Copie créée'
  } catch (e) {
    showActionError(e)
  }
}

/** Opens a loan form prefilled for this property; saving it links them. */
async function createLoanForProperty() {
  if (!loadedId.value) return
  await router.push({ path: '/prets/nouveau', query: { bien: loadedId.value } })
}

async function remove() {
  if (!loadedId.value) return
  if (!window.confirm(`Supprimer le bien « ${savedName.value} » ? Cette action est définitive.`)) {
    return
  }
  clearMessages()
  try {
    await deleteProperty(loadedId.value)
    snapshot.value = JSON.stringify(form) // deleted: nothing left to save
    await router.push('/biens')
  } catch (e) {
    showActionError(e)
  }
}
</script>

<template>
  <section class="detail">
    <AppBreadcrumb
      :items="[
        { label: 'Accueil', to: '/' },
        { label: 'Biens', to: '/biens' },
        { label: loadedId ? savedName : 'Nouveau bien' },
      ]"
    />
    <h1>{{ loadedId ? savedName : 'Nouveau bien' }}</h1>
    <p v-if="loadError" role="alert" class="alert alert--error">{{ loadError }}</p>
    <div v-else class="detail__layout">
      <form id="bien-form" class="card detail__form" novalidate @submit.prevent="save">
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
          label="Charges de copropriété non récupérables (par an)"
          hint="La part des appels du syndic qui reste à votre charge. La part refacturée au locataire va dans « Charges récupérables »."
          :error="errors.annualCondoFeesCents"
        />
        <MoneyInput
          id="bien-pno"
          v-model="form.annualLandlordInsuranceCents"
          label="Assurance PNO (par an)"
          :error="errors.annualLandlordInsuranceCents"
        />
        <MoneyInput
          id="bien-mobilier-compris"
          v-model="form.furnitureIncludedCents"
          label="Mobilier compris dans le prix"
          hint="Meubles listés dans l'acte de vente : exclus des frais de notaire, amortis en LMNP au réel."
          :error="errors.furnitureIncludedCents"
        />
        <MoneyInput
          id="bien-mobilier-achete"
          v-model="form.furnitureAddedCents"
          label="Mobilier acheté en plus"
          hint="Ajouté au coût d'acquisition, amorti en LMNP au réel."
          :error="errors.furnitureAddedCents"
        />
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
            <button
              type="button"
              class="button"
              data-test="create-loan"
              @click="createLoanForProperty"
            >
              Créer un prêt pour ce bien
            </button>
            <button type="button" class="button button--danger" data-test="delete" @click="remove">
              Supprimer
            </button>
          </template>
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
              <component
                :is="result.financing ? 'span' : 'strong'"
                :class="{ 'result-card__value result-card__value--accent': !result.financing }"
              >
                Après impôts {{ formatPercent(result.netAfterTaxYieldPct) }}
              </component>
              <strong
                v-if="result.financing"
                class="result-card__value result-card__value--accent"
                data-test="after-credit-yield"
              >
                Après crédit {{ formatPercent(result.netAfterCreditYieldPct) }}
              </strong>
            </div>
            <div class="result-card" data-test="profit-dates">
              <span class="result-card__label">Date de rentabilité</span>
              <template v-if="result.timeline">
                <span>Cash-flow mensuel positif : {{ positiveDate(result.timeline) }}</span>
                <strong class="result-card__value--accent">
                  Apport récupéré : {{ paybackDate(result.timeline) }}
                </strong>
              </template>
              <span v-else class="placeholder">—</span>
            </div>
          </template>
          <p v-else-if="!error" class="placeholder">
            Saisissez le prix, le loyer, la taxe foncière et la classe DPE pour voir les résultats.
          </p>
        </div>
        <section
          class="card bien-options"
          data-test="bien-options"
          aria-labelledby="bien-options-title"
        >
          <h2 id="bien-options-title">Gestion, fiscalité et prêt</h2>
          <label class="checkbox">
            <input id="bien-agence" v-model="form.agencyManaged" type="checkbox" form="bien-form" />
            Gestion par une agence
          </label>
          <div class="field">
            <label for="bien-regime">Régime fiscal</label>
            <select id="bien-regime" v-model="form.taxRegime" class="input" form="bien-form">
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
              form="bien-form"
              :aria-invalid="errors.loanId ? 'true' : undefined"
            >
              <option value="">Aucun</option>
              <option v-for="l in loans" :key="l.id" :value="l.id">{{ l.name }}</option>
            </select>
            <p v-if="errors.loanId" role="alert" class="field-error">{{ errors.loanId }}</p>
            <RouterLink
              v-if="form.loanId"
              :to="`/prets/${form.loanId}`"
              class="field__link"
              data-test="open-loan"
            >
              Voir le prêt →
            </RouterLink>
          </div>
        </section>
        <FinancingCard v-if="result?.financing" :financing="result.financing" />
        <CalcDetail
          v-if="result"
          :result="result"
          :property-tax-cents="form.annualPropertyTaxCents ?? 0"
          :condo-fees-cents="form.annualCondoFeesCents ?? 0"
          :insurance-cents="form.annualLandlordInsuranceCents ?? 0"
        />
        <RegimeSection
          v-if="result?.regimes?.length"
          :regimes="result.regimes"
          :financed="!!result.financing"
        />
        <ProfitChart
          v-if="result?.timeline"
          :timeline="result.timeline"
          :financed="!!result.financing"
        />
      </div>
    </div>
  </section>
</template>
