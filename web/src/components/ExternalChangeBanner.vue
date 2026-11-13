<script setup lang="ts">
import { ref } from 'vue'
import { ApiError, request } from '@/api/client'
import { externalChange } from '@/api/externalChange'

const reloadError = ref('')

async function reload() {
  reloadError.value = ''
  try {
    await request('POST', '/api/data/reload')
    // Every page refetches its data from the reloaded file.
    window.location.reload()
  } catch (e) {
    reloadError.value = e instanceof ApiError ? e.message : String(e)
  }
}
</script>

<template>
  <div v-if="externalChange.detected" role="alert" class="alert alert--warning app-banner">
    <div>
      <p class="app-banner__title">Le fichier a été modifié en dehors d'immo-calc.</p>
      <p>
        Vos dernières modifications n'ont pas été enregistrées. Rechargez le fichier pour continuer.
      </p>
      <p v-if="reloadError" class="app-banner__error">{{ reloadError }}</p>
    </div>
    <button type="button" class="button" @click="reload">Recharger</button>
  </div>
</template>
