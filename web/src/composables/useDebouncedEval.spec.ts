import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { ApiError } from '@/api/client'
import { useDebouncedEval } from './useDebouncedEval'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('useDebouncedEval', () => {
  it('evaluates once, 300 ms after the last change', async () => {
    const input = ref({ amount: 1 })
    const evaluate = vi.fn().mockResolvedValue('result')
    const { result } = useDebouncedEval(input, evaluate)

    input.value.amount = 2
    await nextTick()
    vi.advanceTimersByTime(200)
    input.value.amount = 3
    await nextTick()
    vi.advanceTimersByTime(299)
    expect(evaluate).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1)
    await flushPromises()
    expect(evaluate).toHaveBeenCalledTimes(1)
    expect(evaluate).toHaveBeenCalledWith({ amount: 3 })
    expect(result.value).toBe('result')
  })

  it('can evaluate immediately', async () => {
    const evaluate = vi.fn().mockResolvedValue(42)
    const { result, evaluateNow } = useDebouncedEval(ref({}), evaluate)

    await evaluateNow()

    expect(result.value).toBe(42)
  })

  it('exposes field errors and clears the result on a validation error', async () => {
    const input = ref({ amount: 1 })
    const evaluate = vi
      .fn()
      .mockResolvedValueOnce('ok')
      .mockRejectedValueOnce(
        new ApiError(400, 'validation_failed', 'Champs invalides', { amount: 'Trop petit' }),
      )
    const { result, fieldErrors, error, evaluateNow } = useDebouncedEval(input, evaluate)
    await evaluateNow()

    await evaluateNow()

    expect(result.value).toBeNull()
    expect(fieldErrors.value).toEqual({ amount: 'Trop petit' })
    expect(error.value).toBe('')
  })

  it('exposes other errors as a message', async () => {
    const evaluate = vi.fn().mockRejectedValue(new ApiError(0, 'network_error', 'Injoignable'))
    const { error, fieldErrors, evaluateNow } = useDebouncedEval(ref({}), evaluate)

    await evaluateNow()

    expect(error.value).toBe('Injoignable')
    expect(fieldErrors.value).toEqual({})
  })

  it('ignores answers that arrive after a newer one', async () => {
    let resolveSlow: (v: string) => void = () => {}
    const evaluate = vi
      .fn()
      .mockReturnValueOnce(new Promise((r) => (resolveSlow = r)))
      .mockResolvedValueOnce('fresh')
    const { result, evaluateNow } = useDebouncedEval(ref({}), evaluate)

    const slow = evaluateNow()
    await evaluateNow()
    resolveSlow('stale')
    await slow

    expect(result.value).toBe('fresh')
  })
})
