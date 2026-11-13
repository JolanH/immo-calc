<script setup lang="ts">
import { computed } from 'vue'

// A month ("YYYY-MM") picked as a French month name plus a year. Built by
// hand because <input type="month"> isn't supported everywhere (Firefox) and
// wouldn't show French month names in every browser.
const props = defineProps<{ id: string; label: string; modelValue: string; error?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const monthNames = [
  'janvier',
  'février',
  'mars',
  'avril',
  'mai',
  'juin',
  'juillet',
  'août',
  'septembre',
  'octobre',
  'novembre',
  'décembre',
]

const parts = computed(() => {
  const m = /^(\d{4})-(\d{2})$/.exec(props.modelValue)
  return m ? { year: m[1], month: Number(m[2]) } : { year: '', month: 1 }
})

function emitValue(year: string, month: number) {
  emit('update:modelValue', /^\d{4}$/.test(year) ? `${year}-${String(month).padStart(2, '0')}` : '')
}

function onMonth(event: Event) {
  emitValue(parts.value.year, Number((event.target as HTMLSelectElement).value))
}

function onYear(event: Event) {
  emitValue((event.target as HTMLInputElement).value.trim(), parts.value.month)
}
</script>

<template>
  <div class="field">
    <label :for="id">{{ label }}</label>
    <div class="field__control">
      <select
        :id="id"
        class="input input--month"
        :value="String(parts.month)"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? `${id}-error` : undefined"
        @change="onMonth"
      >
        <option v-for="(name, i) in monthNames" :key="name" :value="String(i + 1)">
          {{ name }}
        </option>
      </select>
      <input
        :id="`${id}-annee`"
        type="text"
        inputmode="numeric"
        maxlength="4"
        class="input input--year"
        aria-label="Année"
        :value="parts.year"
        :aria-invalid="error ? 'true' : undefined"
        @input="onYear"
      />
    </div>
    <p v-if="error" :id="`${id}-error`" role="alert" class="field-error">{{ error }}</p>
  </div>
</template>
