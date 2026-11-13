<script setup lang="ts">
// The DPE letter on its class colour; the letter itself carries the meaning.
defineProps<{ dpe: string }>()
</script>

<template>
  <span
    class="dpe"
    :class="`dpe--${dpe.toLowerCase()}`"
    :aria-label="`DPE ${dpe}`"
    data-test="dpe-badge"
    >{{ dpe }}</span
  >
</template>
