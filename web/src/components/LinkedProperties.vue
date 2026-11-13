<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { listProperties, type PropertySummary } from '@/api/properties'

// « Biens liés »: the properties that use a loan.
const props = defineProps<{ loanId: string }>()

const properties = ref<PropertySummary[] | null>(null)

onMounted(async () => {
  try {
    properties.value = await listProperties({ loanId: props.loanId })
  } catch {
    properties.value = null // informational: nothing shown if it can't load
  }
})
</script>

<template>
  <section v-if="properties" class="card linked-properties" data-test="linked-properties">
    <h2>Biens liés</h2>
    <ul v-if="properties.length">
      <li v-for="p in properties" :key="p.id">
        <RouterLink :to="`/biens/${p.id}`">{{ p.name }}</RouterLink>
      </li>
    </ul>
    <p v-else class="placeholder">Aucun bien n'utilise ce prêt.</p>
  </section>
</template>
