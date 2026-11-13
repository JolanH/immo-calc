<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { ApiError, request } from '@/api/client'
import ExternalChangeBanner from '@/components/ExternalChangeBanner.vue'

const route = useRoute()

const navLinks = [
  { to: '/biens', label: 'Biens' },
  { to: '/prets', label: 'Prêts' },
  { to: '/parametres', label: 'Paramètres' },
]

/** A section stays highlighted on its sub-pages (/prets/123 → Prêts). */
function inSection(path: string): boolean {
  return route.path === path || route.path.startsWith(`${path}/`)
}

const stopped = ref(false)
const quitError = ref('')

async function quit() {
  quitError.value = ''
  try {
    await request('POST', '/api/shutdown')
    stopped.value = true
  } catch (e) {
    quitError.value = e instanceof ApiError ? e.message : String(e)
  }
}
</script>

<template>
  <main v-if="stopped" class="app-main app-stopped">
    <p>immo-calc est arrêté, vous pouvez fermer cet onglet.</p>
  </main>
  <template v-else>
    <header class="app-header">
      <span class="app-header__name">immo-calc</span>
      <nav class="app-nav">
        <RouterLink
          v-for="link in navLinks"
          :key="link.to"
          :to="link.to"
          :class="{ 'nav-active': inSection(link.to) }"
          :aria-current="inSection(link.to) ? 'page' : undefined"
        >
          {{ link.label }}
        </RouterLink>
      </nav>
      <button type="button" class="app-header__quit" data-test="quit" @click="quit">Quitter</button>
    </header>
    <p v-if="quitError" role="alert" class="app-error">{{ quitError }}</p>
    <ExternalChangeBanner />
    <main class="app-main">
      <RouterView />
    </main>
  </template>
</template>
