<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import NumberInput from './NumberInput.vue'
import RateInput from './RateInput.vue'
import { ApiError } from '@/api/client'
import { getSettings, updateSettings, type Settings } from '@/api/settings'
import { formatInputRate, parseRate } from '@/parse'
import {
  applyAccent,
  applyThemeMode,
  DEFAULT_ACCENT,
  DEFAULT_THEME_MODE,
  THEME_MODES,
  type ThemeMode,
} from '@/theme'

const TAX_BRACKETS = [0, 11, 30, 41, 45]
const REQUIRED = 'Ce champ est obligatoire'

type Form = { [K in Exclude<keyof Settings, 'accentColor' | 'themeMode'>]: number | null }

const form = reactive<Form>({
  marginalTaxRatePct: null,
  socialChargesUnfurnishedPct: null,
  socialChargesFurnishedPct: null,
  notaryFeePct: null,
  vacancyMonthsPerYear: null,
  agencyFeePct: null,
})
/** « Couleur principale »: previewed live, kept only when saved. */
const accent = ref(DEFAULT_ACCENT)
const savedAccent = ref(DEFAULT_ACCENT)
watch(accent, (c) => applyAccent(c))
/** « Thème »: previewed live too. */
const themeMode = ref<ThemeMode>(DEFAULT_THEME_MODE)
const savedThemeMode = ref<ThemeMode>(DEFAULT_THEME_MODE)
watch(themeMode, (m) => applyThemeMode(m))
// Leaving without saving puts the saved colour and theme back.
onBeforeUnmount(() => {
  applyAccent(savedAccent.value)
  applyThemeMode(savedThemeMode.value)
})

function takeSettings(s: Settings) {
  const { accentColor, themeMode: mode, ...numbers } = s
  Object.assign(form, numbers)
  accent.value = accentColor || DEFAULT_ACCENT
  savedAccent.value = accent.value
  themeMode.value = mode || DEFAULT_THEME_MODE
  savedThemeMode.value = themeMode.value
}

const loaded = ref(false)
const errors = ref<Record<string, string>>({})
const error = ref('')
const status = ref('')
const saving = ref(false)

function showError(e: unknown) {
  error.value = e instanceof ApiError ? e.message : String(e)
}

onMounted(async () => {
  try {
    takeSettings(await getSettings())
    loaded.value = true
  } catch (e) {
    showError(e)
  }
})

async function save() {
  error.value = ''
  status.value = ''
  const missing = Object.fromEntries(
    Object.entries(form)
      .filter(([, v]) => v === null)
      .map(([k]) => [k, REQUIRED]),
  )
  errors.value = missing
  if (Object.keys(missing).length > 0) return

  saving.value = true
  try {
    takeSettings(
      await updateSettings({
        ...(form as Omit<Settings, 'accentColor' | 'themeMode'>),
        accentColor: accent.value,
        themeMode: themeMode.value,
      }),
    )
    status.value = 'Paramètres enregistrés'
  } catch (e) {
    if (e instanceof ApiError && Object.keys(e.fields).length > 0) errors.value = e.fields
    else showError(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="card">
    <h2>Hypothèses et fiscalité</h2>
    <p class="placeholder">
      Ces hypothèses s'appliquent au calcul de rentabilité de tous les biens.
    </p>
    <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
    <form v-if="loaded" class="settings-form" novalidate @submit.prevent="save">
      <div class="field">
        <label for="param-tmi">Tranche marginale d'imposition (TMI)</label>
        <div class="field__control">
          <select
            id="param-tmi"
            class="input input--short"
            :value="String(form.marginalTaxRatePct)"
            :aria-invalid="errors.marginalTaxRatePct ? 'true' : undefined"
            @change="form.marginalTaxRatePct = Number(($event.target as HTMLSelectElement).value)"
          >
            <option v-for="b in TAX_BRACKETS" :key="b" :value="String(b)">{{ b }} %</option>
          </select>
        </div>
        <p v-if="errors.marginalTaxRatePct" role="alert" class="field-error">
          {{ errors.marginalTaxRatePct }}
        </p>
      </div>
      <RateInput
        id="param-ps-nue"
        v-model="form.socialChargesUnfurnishedPct"
        label="Prélèvements sociaux — location nue"
        :error="errors.socialChargesUnfurnishedPct"
      />
      <RateInput
        id="param-ps-meublee"
        v-model="form.socialChargesFurnishedPct"
        label="Prélèvements sociaux — location meublée"
        :error="errors.socialChargesFurnishedPct"
      />
      <RateInput
        id="param-notaire"
        v-model="form.notaryFeePct"
        label="Frais de notaire (du prix)"
        :error="errors.notaryFeePct"
      />
      <NumberInput
        id="param-vacance"
        v-model="form.vacancyMonthsPerYear"
        label="Vacance locative (par an)"
        :parse="parseRate"
        :format="formatInputRate"
        suffix="mois"
        :error="errors.vacancyMonthsPerYear"
      />
      <RateInput
        id="param-gestion"
        v-model="form.agencyFeePct"
        label="Frais de gestion locative (des loyers encaissés)"
        :error="errors.agencyFeePct"
      />
      <fieldset class="field segmented" data-test="theme-mode">
        <legend>Thème</legend>
        <div class="segmented__options">
          <label v-for="m in THEME_MODES" :key="m.value" class="segmented__option">
            <input v-model="themeMode" type="radio" name="param-theme" :value="m.value" />
            <span>{{ m.label }}</span>
          </label>
        </div>
        <p v-if="errors.themeMode" role="alert" class="field-error">{{ errors.themeMode }}</p>
      </fieldset>
      <div class="field">
        <label for="param-couleur">Couleur principale de l'interface</label>
        <div class="field__control">
          <input id="param-couleur" v-model="accent" type="color" class="colour-input" />
          <code>{{ accent }}</code>
          <button
            type="button"
            class="button"
            data-test="reset-colour"
            @click="accent = DEFAULT_ACCENT"
          >
            Rouge foncé par défaut
          </button>
        </div>
        <p v-if="errors.accentColor" role="alert" class="field-error">{{ errors.accentColor }}</p>
      </div>
      <p v-if="status" role="status" class="alert alert--success">{{ status }}</p>
      <div class="actions">
        <button type="submit" class="button button--primary" :disabled="saving">Enregistrer</button>
      </div>
    </form>
  </section>
</template>
