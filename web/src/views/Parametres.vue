<script setup lang="ts">
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
import DataSection from '@/components/DataSection.vue'
import SettingsForm from '@/components/SettingsForm.vue'
</script>

<template>
  <section>
    <AppBreadcrumb :items="[{ label: 'Accueil', to: '/' }, { label: 'Paramètres' }]" />
    <h1>Paramètres</h1>
    <SettingsForm />
    <DataSection />
  </section>
</template>
