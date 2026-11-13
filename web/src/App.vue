<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <header class="app-header">
    <span class="app-header__name">immo-calc</span>
    <nav class="app-nav">
      <RouterLink to="/biens">Biens</RouterLink>
      <RouterLink to="/prets">Prêts</RouterLink>
      <RouterLink to="/parametres">Paramètres</RouterLink>
    </nav>
    <button type="button" class="app-header__quit">Quitter</button>
  </header>
  <main class="app-main">
    <RouterView />
  </main>
</template>
