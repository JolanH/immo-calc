import { onBeforeUnmount, onMounted, type Ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'

export const UNSAVED_MESSAGE = 'Des modifications ne sont pas enregistrées. Quitter quand même ?'

/**
 * Asks for confirmation before leaving a page with unsaved changes: on
 * in-app navigation (French confirm) and when closing or reloading the tab
 * (the browser's own warning).
 */
export function useUnsavedGuard(dirty: Ref<boolean>) {
  onBeforeRouteLeave(() => !dirty.value || window.confirm(UNSAVED_MESSAGE))

  function onBeforeUnload(event: BeforeUnloadEvent) {
    if (dirty.value) event.preventDefault()
  }
  onMounted(() => window.addEventListener('beforeunload', onBeforeUnload))
  onBeforeUnmount(() => window.removeEventListener('beforeunload', onBeforeUnload))
}
