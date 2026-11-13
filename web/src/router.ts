import { createRouter, type RouterHistory, type RouteRecordRaw } from 'vue-router'
import BiensListe from './views/BiensListe.vue'
import PretsListe from './views/PretsListe.vue'
import Parametres from './views/Parametres.vue'
import NotFound from './views/NotFound.vue'

export const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/biens' },
  { path: '/biens', name: 'biens', component: BiensListe },
  { path: '/prets', name: 'prets', component: PretsListe },
  { path: '/parametres', name: 'parametres', component: Parametres },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFound },
]

export function createAppRouter(history: RouterHistory) {
  return createRouter({ history, routes })
}
