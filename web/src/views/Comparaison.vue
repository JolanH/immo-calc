<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
import DpeBadge from '@/components/DpeBadge.vue'
import { ApiError } from '@/api/client'
import { listLoans, type LoanSummary } from '@/api/loans'
import {
  evaluateProperty,
  getProperty,
  listProperties,
  type PropertyDetail,
  type PropertyResult,
  type PropertySummary,
  REGIME_LABELS,
} from '@/api/properties'
import { formatEuros, formatPercent } from '@/format'
import { paybackDate, positiveDate } from '@/profitDates'

// Up to 4 biens side by side (SPEC-property.md, Comparison). Each column can
// use its own loan, as saved, for the comparison only: nothing is written on
// the bien; biens and loans live in the URL so a reload keeps them.
const MAX = 4
const ID = /^[0-9a-f]{16}$/

interface Column {
  id: string
  detail: PropertyDetail | null
  error: string
  loanId: string
  result: PropertyResult | null
}

const route = useRoute()
const router = useRouter()
const columns = reactive<Column[]>([])
const loans = ref<LoanSummary[]>([])
const allBiens = ref<PropertySummary[]>([])
const addChoice = ref('')

function queryList(key: string): string[] | null {
  const v = route.query[key]
  return typeof v === 'string' ? v.split(',') : null
}

/** Keeps the URL in step with the columns (biens and their loans). */
function syncUrl() {
  const query: Record<string, string> = {}
  if (columns.length) {
    query.biens = columns.map((c) => c.id).join(',')
    query.prets = columns.map((c) => c.loanId).join(',')
  }
  void router.replace({ path: route.path, query })
}

function message(e: unknown): string {
  return e instanceof ApiError ? e.message : String(e)
}

async function evaluate(col: Column) {
  if (!col.detail) return
  const d = col.detail
  try {
    col.result = await evaluateProperty(
      {
        priceCents: d.priceCents,
        dpe: d.dpe,
        monthlyRentCents: d.monthlyRentCents,
        monthlyRecoverableChargesCents: d.monthlyRecoverableChargesCents,
        annualPropertyTaxCents: d.annualPropertyTaxCents,
        annualCondoFeesCents: d.annualCondoFeesCents,
        annualLandlordInsuranceCents: d.annualLandlordInsuranceCents,
        furnitureIncludedCents: d.furnitureIncludedCents,
        furnitureAddedCents: d.furnitureAddedCents,
        agencyManaged: d.agencyManaged,
        taxRegime: d.taxRegime,
        loanId: col.loanId,
      },
      col.id,
    )
    col.error = ''
  } catch (e) {
    col.result = null
    col.error = message(e)
  }
}

/** Adds a column; loanId undefined means « the bien's own loan ». */
async function addColumn(id: string, loanId?: string) {
  const col = reactive<Column>({ id, detail: null, error: '', loanId: loanId ?? '', result: null })
  columns.push(col)
  try {
    col.detail = await getProperty(id)
    if (loanId === undefined) col.loanId = col.detail.loanId
    await evaluate(col)
  } catch (e) {
    col.error = message(e)
  }
}

onMounted(async () => {
  const ids = [...new Set((queryList('biens') ?? []).filter((id) => ID.test(id)))].slice(0, MAX)
  const chosenLoans = queryList('prets')
  const [loanList, biens] = await Promise.all([
    listLoans().catch(() => []),
    listProperties().catch(() => []),
    ...ids.map((id, i) => addColumn(id, chosenLoans ? (chosenLoans[i] ?? '') : undefined)),
  ])
  loans.value = loanList
  allBiens.value = biens
})

async function changeLoan(col: Column, loanId: string) {
  col.loanId = loanId
  syncUrl()
  await evaluate(col)
}

function removeColumn(i: number) {
  columns.splice(i, 1)
  syncUrl()
}

const addable = computed(() => allBiens.value.filter((b) => !columns.some((c) => c.id === b.id)))

async function addBien() {
  const id = addChoice.value
  addChoice.value = ''
  if (!id || columns.length >= MAX) return
  await addColumn(id)
  syncUrl()
}

const dash = '—'
const money = (v: number | undefined) => (v === undefined ? dash : formatEuros(v))
const pct = (v: number | undefined) => (v === undefined ? dash : formatPercent(v))

/** Text rows: [key, label, value of a column]. */
type Row = { key: string; label: string; value: (c: Column) => string }
const sections: { title: string; rows: (Row | 'dpe' | 'loan' | 'warnings')[] }[] = [
  {
    title: 'Bien',
    rows: [
      { key: 'price', label: 'Prix', value: (c) => money(c.detail?.priceCents) },
      {
        key: 'acquisition-cost',
        label: "Coût d'acquisition",
        value: (c) => money(c.result?.acquisitionCostCents),
      },
      { key: 'rent', label: 'Loyer', value: (c) => money(c.detail?.monthlyRentCents) },
      {
        key: 'regime',
        label: 'Régime fiscal',
        value: (c) => (c.detail ? REGIME_LABELS[c.detail.taxRegime] : dash),
      },
      'dpe',
    ],
  },
  {
    title: 'Résultats',
    rows: [
      {
        key: 'gross-monthly',
        label: 'Revenus mensuels bruts',
        value: (c) => money(c.result?.monthlyGrossIncomeCents),
      },
      {
        key: 'net-monthly',
        label: 'Revenus mensuels après impôts',
        value: (c) => money(c.result?.monthlyNetIncomeCents),
      },
      {
        key: 'net-annual',
        label: 'Revenu annuel net',
        value: (c) => money(c.result?.netAnnualIncomeCents),
      },
      {
        key: 'gross-yield',
        label: 'Rentabilité brute',
        value: (c) => pct(c.result?.grossYieldPct),
      },
      { key: 'net-yield', label: 'Rentabilité nette', value: (c) => pct(c.result?.netYieldPct) },
      {
        key: 'after-tax-yield',
        label: 'Rentabilité nette après impôts',
        value: (c) => pct(c.result?.netAfterTaxYieldPct),
      },
      {
        key: 'after-credit-yield',
        label: 'Rentabilité nette après crédit',
        value: (c) => pct(c.result?.netAfterCreditYieldPct),
      },
    ],
  },
  {
    title: 'Financement',
    rows: [
      'loan',
      {
        key: 'payment',
        label: 'Mensualité',
        value: (c) => money(c.result?.financing?.monthlyPaymentCents),
      },
      {
        key: 'cash-flow-before',
        label: 'Cash-flow avant impôts',
        value: (c) => money(c.result?.financing?.cashFlowBeforeTaxCents),
      },
      {
        key: 'cash-flow-after',
        label: 'Cash-flow après impôts',
        value: (c) => money(c.result?.financing?.cashFlowAfterTaxCents),
      },
      {
        key: 'contribution',
        label: 'Apport',
        value: (c) => money(c.result?.financing?.ownContributionCents),
      },
      {
        key: 'return',
        label: 'Rendement sur apport',
        value: (c) => {
          const f = c.result?.financing
          if (!f) return dash
          return f.returnOnContributionPct === null
            ? 'n/a'
            : formatPercent(f.returnOnContributionPct)
        },
      },
    ],
  },
  {
    title: 'Date de rentabilité',
    rows: [
      {
        key: 'positive',
        label: 'Cash-flow mensuel positif',
        value: (c) => (c.result?.timeline ? positiveDate(c.result.timeline) : dash),
      },
      {
        key: 'payback',
        label: 'Apport récupéré',
        value: (c) => (c.result?.timeline ? paybackDate(c.result.timeline) : dash),
      },
      'warnings',
    ],
  },
]
</script>

<template>
  <section>
    <AppBreadcrumb
      :items="[
        { label: 'Accueil', to: '/' },
        { label: 'Biens', to: '/biens' },
        { label: 'Comparaison' },
      ]"
    />
    <div class="page-header">
      <h1>Comparaison</h1>
      <select
        v-if="columns.length > 0 && columns.length < MAX && addable.length"
        v-model="addChoice"
        class="input compare-add"
        data-test="add-bien"
        aria-label="Ajouter un bien à la comparaison"
        @change="addBien"
      >
        <option value="">Ajouter un bien…</option>
        <option v-for="b in addable" :key="b.id" :value="b.id">{{ b.name }}</option>
      </select>
    </div>

    <p v-if="columns.length === 0" class="card empty-state">
      Sélectionnez de 2 à 4 biens dans la liste des biens pour les comparer.
      <RouterLink to="/biens">Aller aux biens</RouterLink>
    </p>

    <div v-else class="table-wrap">
      <table class="table compare-table">
        <thead>
          <tr>
            <th scope="col"><span class="visually-hidden">Indicateur</span></th>
            <th v-for="(c, i) in columns" :key="c.id" scope="col" data-test="compare-column">
              <div class="compare-head">
                <RouterLink v-if="c.detail" :to="`/biens/${c.id}`">{{ c.detail.name }}</RouterLink>
                <span v-else-if="c.error" class="field-error">{{ c.error }}</span>
                <button
                  type="button"
                  class="alert__close"
                  data-test="remove-column"
                  :aria-label="`Retirer ${c.detail?.name ?? 'ce bien'} de la comparaison`"
                  @click="removeColumn(i)"
                >
                  ×
                </button>
              </div>
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-for="section in sections" :key="section.title">
            <tr class="compare-section">
              <th scope="colgroup" :colspan="columns.length + 1">{{ section.title }}</th>
            </tr>
            <template v-for="row in section.rows" :key="typeof row === 'string' ? row : row.key">
              <tr v-if="row === 'dpe'" data-test="row-dpe">
                <th scope="row">DPE</th>
                <td v-for="c in columns" :key="c.id">
                  <DpeBadge v-if="c.detail" :dpe="c.detail.dpe" />
                  <span v-else>{{ dash }}</span>
                </td>
              </tr>
              <tr v-else-if="row === 'loan'" data-test="row-loan">
                <th scope="row">Prêt</th>
                <td v-for="c in columns" :key="c.id">
                  <select
                    v-if="c.detail"
                    class="input"
                    :value="c.loanId"
                    :aria-label="`Prêt appliqué à ${c.detail.name}`"
                    @change="changeLoan(c, ($event.target as HTMLSelectElement).value)"
                  >
                    <option value="">Aucun</option>
                    <option v-for="l in loans" :key="l.id" :value="l.id">{{ l.name }}</option>
                  </select>
                </td>
              </tr>
              <tr v-else-if="row === 'warnings'" data-test="row-warnings">
                <th scope="row">Avertissements</th>
                <td v-for="c in columns" :key="c.id" class="compare-warnings">
                  <template v-if="c.result?.warnings.length">
                    <p
                      v-for="w in c.result.warnings"
                      :key="w.message"
                      :class="`warning warning--${w.severity}`"
                    >
                      {{ w.message }}
                    </p>
                  </template>
                  <span v-else>{{ dash }}</span>
                </td>
              </tr>
              <tr
                v-else
                :data-test="`row-${row.key}`"
                :class="{ 'compare-kpi': row.key === 'after-credit-yield' }"
              >
                <th scope="row">{{ row.label }}</th>
                <td v-for="c in columns" :key="c.id" class="num">{{ row.value(c) }}</td>
              </tr>
            </template>
          </template>
        </tbody>
      </table>
    </div>
  </section>
</template>
