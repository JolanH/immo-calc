<script setup lang="ts">
import { useRouter } from 'vue-router'

// « Comparer (n) »: opens the comparison of the ticked biens (2 to 4).
// With loanId (the home page's active loan), every column starts on it.
const props = defineProps<{ selected: string[]; loanId?: string }>()
const router = useRouter()

function compare() {
  const query: Record<string, string> = { biens: props.selected.join(',') }
  if (props.loanId) query.prets = props.selected.map(() => props.loanId).join(',')
  void router.push({ path: '/biens/comparer', query })
}
</script>

<template>
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
</template>
