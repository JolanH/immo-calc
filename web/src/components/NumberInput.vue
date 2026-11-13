<script setup lang="ts">
import { ref, watch } from 'vue'

// Text input for a number typed the French way. It keeps what the user types
// while it parses to the current value, and reformats on blur. MoneyInput and
// RateInput are thin wrappers around it.
const props = defineProps<{
  id: string
  label: string
  modelValue: number | null
  parse: (text: string) => number | null
  format: (value: number) => string
  suffix?: string
  error?: string
  placeholder?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: number | null] }>()

const display = (v: number | null) => (v === null ? '' : props.format(v))
const text = ref(display(props.modelValue))

watch(
  () => props.modelValue,
  (value) => {
    if (props.parse(text.value) !== value) text.value = display(value)
  },
)

function onInput(event: Event) {
  text.value = (event.target as HTMLInputElement).value
  emit('update:modelValue', props.parse(text.value))
}

function onBlur() {
  if (props.modelValue !== null) text.value = display(props.modelValue)
}
</script>

<template>
  <div class="field">
    <label :for="id">{{ label }}</label>
    <div class="field__control">
      <input
        :id="id"
        :value="text"
        type="text"
        inputmode="decimal"
        autocomplete="off"
        class="input"
        :placeholder="placeholder"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? `${id}-error` : undefined"
        @input="onInput"
        @blur="onBlur"
      />
      <span v-if="suffix" class="field__suffix" aria-hidden="true">{{ suffix }}</span>
    </div>
    <p v-if="error" :id="`${id}-error`" role="alert" class="field-error">{{ error }}</p>
  </div>
</template>
