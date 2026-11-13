<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'

// A modal dialog: closes with its button, Escape or a click on the backdrop,
// takes the focus when it opens and gives it back when it closes.
const props = defineProps<{ open: boolean; title: string }>()
const emit = defineEmits<{ close: [] }>()

let counter = 0
const titleId = `modal-title-${++counter}-${Math.random().toString(36).slice(2, 8)}`
const closeButton = ref<HTMLButtonElement | null>(null)
let opener: Element | null = null

watch(
  () => props.open,
  async (open) => {
    if (open) {
      opener = document.activeElement
      await nextTick()
      closeButton.value?.focus()
    } else if (opener instanceof HTMLElement) {
      opener.focus()
      opener = null
    }
  },
  { flush: 'post' },
)
</script>

<template>
  <div v-if="open" class="modal-backdrop" data-test="modal-backdrop" @click.self="emit('close')">
    <div
      class="modal"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="titleId"
      @keydown.esc.stop="emit('close')"
    >
      <header class="modal__header">
        <h2 :id="titleId">{{ title }}</h2>
        <button
          ref="closeButton"
          type="button"
          class="button modal__close"
          data-test="modal-close"
          aria-label="Fermer"
          @click="emit('close')"
        >
          ✕
        </button>
      </header>
      <div class="modal__body">
        <slot />
      </div>
    </div>
  </div>
</template>
