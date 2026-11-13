<script setup lang="ts">
import { computed } from 'vue'
import { formatEuros } from '@/format'
import { taxSections, type TaxStepsInput } from '@/taxSteps'

// « Comment est calculé l'impôt »: the annual tax rebuilt line by line from
// this bien's own figures.
const props = defineProps<{ input: TaxStepsInput }>()
const sections = computed(() => taxSections(props.input))
</script>

<template>
  <div class="tax-explanation" data-test="tax-explanation">
    <p class="placeholder">
      Chaque montant ci-dessous vient des valeurs saisies pour ce bien et des paramètres : vous
      pouvez refaire le calcul à la main.
    </p>
    <section v-for="s in sections" :key="s.title" class="tax-explanation__section">
      <h3>{{ s.title }}</h3>
      <table class="tax-explanation__table">
        <tbody>
          <tr
            v-for="l in s.lines"
            :key="l.label"
            :class="{ 'tax-explanation__total': l.op === '=' }"
          >
            <td class="tax-explanation__op" aria-hidden="true">{{ l.op }}</td>
            <th scope="row">
              {{ l.label }}
              <span v-if="l.detail" class="tax-explanation__formula">{{ l.detail }}</span>
            </th>
            <td class="num">{{ formatEuros(l.cents) }}</td>
          </tr>
        </tbody>
      </table>
      <p v-if="s.note" class="tax-explanation__note">{{ s.note }}</p>
    </section>
  </div>
</template>
