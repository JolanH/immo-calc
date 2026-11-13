import type { Timeline } from '@/api/properties'
import { formatMonthDate } from '@/format'

/** « Cash-flow mensuel positif » date, or « jamais ». */
export function positiveDate(t: Timeline): string {
  return t.positiveCashFlowMonth ? formatMonthDate(t.positiveCashFlowMonth) : 'jamais'
}

/** « Apport récupéré » date, « dès l'achat » or « au-delà de 50 ans ». */
export function paybackDate(t: Timeline): string {
  if (t.paybackAtStart) return "dès l'achat"
  return t.paybackMonth ? formatMonthDate(t.paybackMonth) : 'au-delà de 50 ans'
}
