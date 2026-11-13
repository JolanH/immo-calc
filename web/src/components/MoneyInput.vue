<script setup lang="ts">
import NumberInput from './NumberInput.vue'
import { formatInputEuros, parseEuros } from '@/parse'

// An amount in euros, typed "200 000" or "1 234,56"; the model is in cents.
defineProps<{
  id: string
  label: string
  modelValue: number | null
  error?: string
  placeholder?: string
  /** Help text under the field. */
  hint?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: number | null] }>()
</script>

<template>
  <NumberInput
    :id="id"
    :label="label"
    :model-value="modelValue"
    :parse="parseEuros"
    :format="formatInputEuros"
    suffix="€"
    :error="error"
    :placeholder="placeholder"
    :hint="hint"
    @update:model-value="emit('update:modelValue', $event)"
  />
</template>
