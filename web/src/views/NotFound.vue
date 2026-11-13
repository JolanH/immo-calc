<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <section>
    <h1>Page introuvable</h1>
    <p>Cette page n'existe pas.</p>
    <RouterLink to="/biens" data-test="back-to-biens">Retour aux biens</RouterLink>
  </section>
</template>
