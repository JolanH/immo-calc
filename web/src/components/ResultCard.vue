<script setup lang="ts">
import { formatEuros } from '@/format'

defineProps<{ label: string; cents: number; highlight?: boolean }>()
</script>

<template>
  <div class="result-card" :class="{ 'result-card--highlight': highlight }">
    <span class="result-card__label">{{ label }}</span>
    <strong class="result-card__value">{{ formatEuros(cents) }}</strong>
  </div>
</template>
