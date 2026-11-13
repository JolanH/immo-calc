<script setup lang="ts">
import { RouterLink } from 'vue-router'

// « Prêts › Appart Lyon »: items with a `to` link back up; the last item is
// the current page.
defineProps<{ items: { label: string; to?: string }[] }>()
</script>

<template>
  <nav aria-label="Fil d'Ariane" class="breadcrumb">
    <ol>
      <li v-for="(item, i) in items" :key="i">
        <RouterLink v-if="item.to && i < items.length - 1" :to="item.to">{{
          item.label
        }}</RouterLink>
        <span v-else aria-current="page">{{ item.label }}</span>
      </li>
    </ol>
  </nav>
</template>
