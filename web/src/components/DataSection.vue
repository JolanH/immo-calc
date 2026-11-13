<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ApiError } from '@/api/client'
import { getDataInfo, type DataInfo } from '@/api/data'

const info = ref<DataInfo | null>(null)
const error = ref('')
const warningDismissed = ref(false)

function plural(n: number, singular: string, pluralForm: string): string {
  return `${n} ${n > 1 ? pluralForm : singular}`
}

async function load() {
  error.value = ''
  try {
    info.value = await getDataInfo()
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : String(e)
  }
}

onMounted(load)
</script>

<template>
  <section class="card">
    <h2>Données</h2>
    <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
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
      <dd>
        {{ plural(info.counts.properties, 'bien', 'biens') }},
        {{ plural(info.counts.loans, 'prêt', 'prêts') }}
      </dd>
    </dl>
  </section>
</template>
