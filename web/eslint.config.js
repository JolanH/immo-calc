import pluginVue from 'eslint-plugin-vue'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'

export default defineConfigWithVueTs(
  { ignores: ['dist/**', 'node_modules/**', 'coverage/**'] },
  pluginVue.configs['flat/recommended'],
  vueTsConfigs.recommended,
  // Route pages are named after the French page they show (e.g. Parametres).
  { files: ['src/views/**/*.vue'], rules: { 'vue/multi-word-component-names': 'off' } },
  skipFormatting,
)
