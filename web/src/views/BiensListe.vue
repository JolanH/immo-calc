<script setup lang="ts">
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
</script>

<template>
  <section>
    <AppBreadcrumb :items="[{ label: 'Accueil', to: '/' }, { label: 'Biens' }]" />
    <h1>Biens</h1>
    <p class="placeholder">Fonctionnalité à venir</p>
  </section>
</template>
