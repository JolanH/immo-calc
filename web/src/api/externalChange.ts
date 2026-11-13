import { reactive } from 'vue'

/**
 * Set when any API call answers 409 file_changed_externally: the data file
 * was edited outside immo-calc. ExternalChangeBanner shows it app-wide.
 */
export const externalChange = reactive({ detected: false })
