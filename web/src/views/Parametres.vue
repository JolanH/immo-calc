<script setup lang="ts">
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
import DataSection from '@/components/DataSection.vue'
</script>

<template>
  <section>
    <AppBreadcrumb :items="[{ label: 'Accueil', to: '/' }, { label: 'Paramètres' }]" />
    <h1>Paramètres</h1>
    <DataSection />
  </section>
</template>
