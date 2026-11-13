<script setup lang="ts">
import NumberInput from './NumberInput.vue'
import { formatInputRate, parseRate } from '@/parse'

// A percentage, typed "3,5"; the model is the number (3.5 means 3.5 %).
defineProps<{
  id: string
  label: string
  modelValue: number | null
  error?: string
  placeholder?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: number | null] }>()
</script>

<template>
  <NumberInput
    :id="id"
    :label="label"
    :model-value="modelValue"
    :parse="parseRate"
    :format="formatInputRate"
    suffix="%"
    :error="error"
    :placeholder="placeholder"
    @update:model-value="emit('update:modelValue', $event)"
  />
</template>
