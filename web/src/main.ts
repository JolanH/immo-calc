import { createApp } from 'vue'
import { createWebHistory } from 'vue-router'
import App from './App.vue'
import { createAppRouter } from './router'
import './style.css'
import { getSettings } from './api/settings'
import { applyAccent, DEFAULT_ACCENT } from './theme'

applyAccent(DEFAULT_ACCENT)
createApp(App).use(createAppRouter(createWebHistory())).mount('#app')
// The saved main colour, once known (the default stays if it can't load).
getSettings()
  .then((s) => applyAccent(s.accentColor))
  .catch(() => {})
