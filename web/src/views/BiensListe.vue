<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
import PropertiesTable from '@/components/PropertiesTable.vue'
import { ApiError } from '@/api/client'
import { listProperties, type PropertySummary } from '@/api/properties'

const router = useRouter()
const properties = ref<PropertySummary[] | null>(null)
/** Biens ticked for the comparison (2 to 4). */
const selected = ref<string[]>([])

function compare() {
  void router.push({ path: '/biens/comparer', query: { biens: selected.value.join(',') } })
}
const error = ref('')

onMounted(async () => {
  try {
    properties.value = await listProperties()
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : String(e)
  }
})
</script>

<template>
  <section>
    <AppBreadcrumb :items="[{ label: 'Accueil', to: '/' }, { label: 'Biens' }]" />
    <div class="page-header">
      <h1>Biens</h1>
      <RouterLink
        v-if="properties?.length"
        to="/biens/nouveau"
        class="button button--primary"
        data-test="new-property"
      >
        Nouveau bien
      </RouterLink>
    </div>
    <p v-if="error" role="alert" class="alert alert--error">{{ error }}</p>
    <div v-if="properties && properties.length === 0" class="card empty-state">
      <p>Aucun bien enregistré.</p>
      <RouterLink to="/biens/nouveau" class="button button--primary" data-test="new-property">
        Nouveau bien
      </RouterLink>
    </div>
    <template v-else-if="properties">
      <div class="compare-bar">
        <span class="placeholder">Cochez de 2 à 4 biens pour les comparer.</span>
        <button
          type="button"
          class="button"
          data-test="compare"
          :disabled="selected.length < 2"
          @click="compare"
        >
          {{ selected.length ? `Comparer (${selected.length})` : 'Comparer' }}
        </button>
      </div>
      <PropertiesTable v-model:selected="selected" :properties="properties" selectable />
    </template>
  </section>
</template>
