<script setup lang="ts">
import { REGIME_LABELS, type RegimeComparison, type TaxRegime } from '@/api/properties'
import { formatEuros, formatPercent } from '@/format'

// « Régime fiscal » (shown in a modal): how each regime taxes the rent, and the four of them
// applied to this bien with the same settings and loan (SPEC-property.md,
// UI; docs/tax-regimes.md).
defineProps<{ regimes: RegimeComparison[]; financed?: boolean }>()

const EXPLANATIONS: Record<TaxRegime, { kind: string; text: string }> = {
  micro_foncier: {
    kind: 'Location nue',
    text: "Imposable : 70 % des recettes, loyers encaissés et charges remboursées par le locataire (abattement forfaitaire de 30 %), sans déduire les charges réelles ni les intérêts. Réservé aux loyers nus jusqu'à 15 000 € par an.",
  },
  reel_foncier: {
    kind: 'Location nue',
    text: "Imposable : loyers − charges réelles (taxe foncière, copropriété non récupérable, assurance, gestion) − intérêts, assurance et frais de dossier et de garantie de l'emprunt de la 1re année. Plus avantageux dès que ces déductions dépassent 30 % des loyers.",
  },
  micro_bic: {
    kind: 'Location meublée',
    text: "Imposable : 50 % des recettes, loyers encaissés et charges remboursées par le locataire (abattement forfaitaire de 50 %). Réservé aux recettes meublées jusqu'à 83 600 € par an.",
  },
  lmnp_reel: {
    kind: 'Location meublée',
    text: "Imposable : loyers − charges réelles − intérêts, assurance et frais de l'emprunt − amortissements (bâti hors 15 % de terrain sur 30 ans, mobilier sur 7 ans). Les amortissements ne créent jamais de déficit.",
  },
}
</script>

<template>
  <div class="regime-section" data-test="regime-section">
    <p class="placeholder">
      Les régimes « micro » sont une simplification : un abattement forfaitaire remplace les charges
      réelles. Le régime réel est plus avantageux quand les charges{{
        financed ? ' et les intérêts' : ''
      }}
      dépassent cet abattement. Impôt = revenu imposable × (TMI + prélèvements sociaux : 17,2 % en
      location nue, 18,6 % en meublé, modifiables dans Paramètres).
    </p>
    <dl class="regime-section__list">
      <template v-for="r in regimes" :key="r.regime">
        <dt>
          {{ REGIME_LABELS[r.regime] }}
          <span class="regime-section__kind">· {{ EXPLANATIONS[r.regime].kind }}</span>
        </dt>
        <dd>{{ EXPLANATIONS[r.regime].text }}</dd>
      </template>
    </dl>
    <div class="table-wrap">
      <table class="table regime-section__table" data-test="regime-table">
        <caption class="visually-hidden">
          Comparaison des régimes fiscaux sur ce bien
        </caption>
        <thead>
          <tr>
            <th scope="col">Régime</th>
            <th scope="col" class="num">Revenu imposable</th>
            <th scope="col" class="num">Impôt annuel</th>
            <th scope="col" class="num">Revenu annuel net</th>
            <th scope="col" class="num">Rentabilité après impôts</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="r in regimes"
            :key="r.regime"
            :class="{ 'regime-section__current': r.current }"
            :data-test="`regime-${r.regime}`"
          >
            <th scope="row">
              {{ REGIME_LABELS[r.regime] }}
              <span v-if="r.current" class="tag" data-test="tag-current">actuel</span>
              <span v-if="r.best" class="tag tag--best" data-test="tag-best">
                le plus avantageux en {{ r.furnished ? 'meublé' : 'location nue' }}
              </span>
              <span v-if="!r.eligible" class="tag" data-test="tag-ineligible">
                plafond dépassé
              </span>
            </th>
            <td class="num">{{ formatEuros(r.taxableIncomeCents) }}</td>
            <td class="num">{{ formatEuros(r.annualTaxCents) }}</td>
            <td class="num">{{ formatEuros(r.netAnnualIncomeCents) }}</td>
            <td class="num">{{ formatPercent(r.netAfterTaxYieldPct) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p class="placeholder">
      Location nue et meublée ne sont pas interchangeables : le meublé suppose un logement équipé et
      se loue souvent à un autre loyer, donc le régime le plus avantageux est désigné séparément
      pour chacune. Un régime micro dont le plafond est dépassé ne peut pas être choisi. Impôt de la
      1re année ; il évolue ensuite avec les intérêts de l'emprunt (dates de rentabilité).
    </p>
  </div>
</template>
