import { createApp } from 'vue'
import { createWebHistory } from 'vue-router'
import App from './App.vue'
import { createAppRouter } from './router'
import './style.css'
import './material.css'
import { getSettings } from './api/settings'
import { applyAccent, applyThemeMode, cachedThemeMode, DEFAULT_ACCENT } from './theme'

applyAccent(DEFAULT_ACCENT)
applyThemeMode(cachedThemeMode())
createApp(App).use(createAppRouter(createWebHistory())).mount('#app')
// The saved main colour, once known (the default stays if it can't load).
getSettings()
  .then((s) => {
    applyAccent(s.accentColor)
    applyThemeMode(s.themeMode)
  })
  .catch(() => {})
