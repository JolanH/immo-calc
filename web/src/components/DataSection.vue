<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ApiError } from '@/api/client'
import { getDataInfo, importData, type DataInfo } from '@/api/data'

const info = ref<DataInfo | null>(null)
const error = ref('')
const status = ref('')
const warningDismissed = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

function plural(n: number, singular: string, pluralForm: string): string {
  return `${n} ${n > 1 ? pluralForm : singular}`
}

function countsText(counts: DataInfo['counts']): string {
  return `${plural(counts.properties, 'bien', 'biens')}, ${plural(counts.loans, 'prêt', 'prêts')}`
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

  error.value = ''
  status.value = ''
  try {
    info.value = await importData(await file.text())
    status.value = 'Import terminé'
  } catch (e) {
    showError(e)
  }
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
  </section>
</template>
