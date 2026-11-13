import type { Timeline } from '@/api/properties'
import { formatMonthDate } from '@/format'

/** « Cash-flow mensuel positif »: a date, « Immédiat » (from the first month) or « Jamais ». */
export function positiveDate(t: Timeline): string {
  if (!t.positiveCashFlowMonth) return 'Jamais'
  return t.positiveCashFlowMonth === t.startMonth
    ? 'Immédiat'
    : formatMonthDate(t.positiveCashFlowMonth)
}

/** « Apport récupéré »: a date, « Dès l'achat » or « Au-delà de 50 ans ». */
export function paybackDate(t: Timeline): string {
  if (t.paybackAtStart) return "Dès l'achat"
  return t.paybackMonth ? formatMonthDate(t.paybackMonth) : 'Au-delà de 50 ans'
}
