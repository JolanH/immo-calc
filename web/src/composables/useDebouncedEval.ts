import { ref, watch, type Ref } from 'vue'
import { ApiError } from '@/api/client'

/**
 * Re-evaluates `input` with `evaluate` (a call to the Go API) 300 ms after
 * the last change, for live results. Answers that arrive after a newer
 * request are ignored. Validation errors (400 with fields) become
 * fieldErrors; anything else becomes error.
 */
export function useDebouncedEval<I, R>(
  input: Ref<I>,
  evaluate: (input: I) => Promise<R>,
  delayMs = 300,
) {
  const result = ref<R | null>(null) as Ref<R | null>
  const fieldErrors = ref<Record<string, string>>({})
  const error = ref('')
  let timer: ReturnType<typeof setTimeout> | undefined
  let latest = 0

  async function evaluateNow(): Promise<void> {
    clearTimeout(timer)
    const call = ++latest
    try {
      const value = await evaluate(input.value)
      if (call !== latest) return
      result.value = value
      fieldErrors.value = {}
      error.value = ''
    } catch (e) {
      if (call !== latest) return
      result.value = null
      if (e instanceof ApiError && Object.keys(e.fields).length > 0) {
        fieldErrors.value = e.fields
        error.value = ''
      } else {
        fieldErrors.value = {}
        error.value = e instanceof ApiError ? e.message : String(e)
      }
    }
  }

  watch(
    input,
    () => {
      clearTimeout(timer)
      timer = setTimeout(() => void evaluateNow(), delayMs)
    },
    { deep: true },
  )

  return { result, fieldErrors, error, evaluateNow }
}
