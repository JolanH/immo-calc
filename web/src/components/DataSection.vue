<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ApiError } from '@/api/client'
import { getDataInfo, importData, openFile, type DataInfo } from '@/api/data'

const info = ref<DataInfo | null>(null)
const error = ref('')
const status = ref('')
const warningDismissed = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

/** Typed-path fallback, shown when the server has no native file dialog. */
const pathForm = ref<{ message: string; value: string; fieldError: string } | null>(null)

function plural(n: number, singular: string, pluralForm: string): string {
  return `${n} ${n > 1 ? pluralForm : singular}`
}

function countsText(counts: DataInfo['counts']): string {
  return `${plural(counts.properties, 'bien', 'biens')}, ${plural(counts.loans, 'prêt', 'prêts')}`
}

function clearMessages() {
  error.value = ''
  status.value = ''
}

function showError(e: unknown) {
  error.value = e instanceof ApiError ? e.message : String(e)
}

async function load() {
  error.value = ''
  try {
    info.value = await getDataInfo()
  } catch (e) {
    showError(e)
  }
}

async function onFileChosen(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // choosing the same file again must trigger a new change
  if (!file || !info.value) return

  const question =
    `Toutes les données actuelles (${countsText(info.value.counts)}) seront remplacées ` +
    'par celles du fichier importé. Une sauvegarde du fichier actuel sera créée. Continuer ?'
  if (!window.confirm(question)) return

  clearMessages()
  try {
    info.value = await importData(await file.text())
    status.value = 'Import terminé'
  } catch (e) {
    showError(e)
  }
}

/** Opens another data file, through the native dialog or a typed path. */
async function open(path?: string) {
  clearMessages()
  try {
    const result = await openFile(path)
    if (!result) return // dialog cancelled
    info.value = result
    pathForm.value = null
    warningDismissed.value = true
    status.value = `Fichier ouvert : ${result.path}`
  } catch (e) {
    if (e instanceof ApiError && e.code === 'picker_unavailable') {
      pathForm.value = { message: e.message, value: '', fieldError: '' }
    } else if (e instanceof ApiError && pathForm.value && e.fields.path) {
      pathForm.value.fieldError = e.fields.path
    } else {
      showError(e)
    }
  }
}

function submitPath() {
  if (!pathForm.value) return
  pathForm.value.fieldError = ''
  void open(pathForm.value.value)
}

onMounted(load)
</script>

<template>
  <section class="card">
    <h2>Données</h2>
    <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
    <p v-if="status" role="status" class="alert alert--success">{{ status }}</p>
    <div
      v-if="info?.warning && !warningDismissed"
      class="alert alert--warning"
      data-test="data-warning"
    >
      <span>{{ info.warning }}</span>
      <button
        type="button"
        class="alert__close"
        aria-label="Fermer"
        @click="warningDismissed = true"
      >
        ×
      </button>
    </div>
    <dl v-if="info" class="data-info">
      <dt>Fichier de données</dt>
      <dd>
        <code data-test="data-path">{{ info.path }}</code>
      </dd>
      <dt>Contenu</dt>
      <dd>{{ countsText(info.counts) }}</dd>
    </dl>
    <div v-if="info" class="actions">
      <button type="button" class="button" data-test="open" @click="open()">
        Ouvrir un autre fichier…
      </button>
      <a class="button" href="/api/data/export" download data-test="export">Exporter</a>
      <button type="button" class="button" @click="fileInput?.click()">Importer…</button>
      <input
        ref="fileInput"
        type="file"
        accept=".yaml,.yml,application/yaml"
        class="visually-hidden"
        tabindex="-1"
        aria-hidden="true"
        @change="onFileChosen"
      />
    </div>
    <form v-if="pathForm" class="path-form" data-test="path-form" @submit.prevent="submitPath">
      <p>{{ pathForm.message }}</p>
      <label for="data-path-input">Chemin du fichier</label>
      <div class="path-form__row">
        <input
          id="data-path-input"
          v-model="pathForm.value"
          type="text"
          class="input"
          placeholder="/home/moi/immo-calc.yaml"
          :aria-invalid="pathForm.fieldError ? 'true' : undefined"
          aria-describedby="data-path-error"
        />
        <button type="submit" class="button">Ouvrir</button>
        <button type="button" class="button" data-test="path-cancel" @click="pathForm = null">
          Annuler
        </button>
      </div>
      <p v-if="pathForm.fieldError" id="data-path-error" class="field-error">
        {{ pathForm.fieldError }}
      </p>
    </form>
  </section>
</template>
