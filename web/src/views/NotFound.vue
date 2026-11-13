<script setup lang="ts">
import { RouterLink } from 'vue-router'
import AppBreadcrumb from '@/components/AppBreadcrumb.vue'
</script>

<template>
  <section>
    <AppBreadcrumb :items="[{ label: 'Accueil', to: '/' }, { label: 'Page introuvable' }]" />
    <h1>Page introuvable</h1>
    <p>Cette page n'existe pas.</p>
    <RouterLink to="/biens" data-test="back-to-biens">Retour aux biens</RouterLink>
  </section>
</template>
