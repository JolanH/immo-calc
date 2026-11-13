import { createRouter, type RouterHistory, type RouteRecordRaw } from 'vue-router'
import Accueil from './views/Accueil.vue'
import BiensListe from './views/BiensListe.vue'
import BienDetail from './views/BienDetail.vue'
import PretsListe from './views/PretsListe.vue'
import PretDetail from './views/PretDetail.vue'
import Parametres from './views/Parametres.vue'
import NotFound from './views/NotFound.vue'

export const routes: RouteRecordRaw[] = [
  { path: '/', name: 'accueil', component: Accueil },
  { path: '/biens', name: 'biens', component: BiensListe },
  // Like loans: one route for new (/biens/nouveau) and saved properties.
  { path: '/biens/:id', name: 'bien', component: BienDetail },
  { path: '/prets', name: 'prets', component: PretsListe },
  // One route for new (/prets/nouveau) and saved loans, so the page stays
  // mounted when a new loan is saved and its URL changes.
  { path: '/prets/:id', name: 'pret', component: PretDetail },
  { path: '/parametres', name: 'parametres', component: Parametres },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFound },
]

export function createAppRouter(history: RouterHistory) {
  return createRouter({ history, routes })
}
