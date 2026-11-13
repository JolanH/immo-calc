# Spec: loan

> Module of [immo-calc](SPEC.md). Depends on: `app-shell`, `storage`. Shared conventions are in `SPEC.md`.

## Objective

Let the user simulate a mortgage (*prêt immobilier amortissable à mensualités constantes*), see what it really costs, and **save** simulations so they can be compared, duplicated, and later attached to a property (`property` module).

Inputs: **Montant, Taux, Durée**, plus optional borrower insurance and fees. Outputs: **mensualité**, **intérêts** (total and per period), **coût du crédit**, and the **tableau d'amortissement**.

## Scope

**In:**
- Pure loan maths (`internal/loan`)
- The `loans` section of the data file and its validator
- CRUD and duplication of saved simulations
- Live (unsaved) simulation
- API
- Vue pages *Prêts* (list) and *Prêt* (form + results + schedule)
- A **delete-guard hook** so `property` can block deleting a loan that is in use

**Out:**
- TAEG (legal APR)
- Variable rates, deferral (*différé*), early repayment (*remboursement anticipé*), smoothing (*lissage*) of several loans
- Borrowing capacity (*taux d'endettement*)
- The property link itself: it's stored on the property side and defined in `SPEC-property.md`

## Calculation rules

The usual French convention for a fixed-rate amortizing loan:

- **Monthly rate** = annual nominal rate / 12 (*taux proportionnel*), applied to the **remaining principal** each month. This "interest on remaining principal" is what the request called composed interest. The UI shows it as **Intérêts**.
- **Mensualité hors assurance** `M = P·r / (1 − (1+r)^−n)`, rounded to the cent. At 0 % it's `P / n`.
- **Schedule**, for each month `k = 1..n`:
  - `intérêts_k = round(restant_{k−1} · r)`
  - `capital_k = M − intérêts_k`
  - `restant_k = restant_{k−1} − capital_k`
  - The **last installment** is adjusted so that `restant_n = 0` exactly (it can differ from `M` by a few cents).
- **Borrower insurance** (optional): annual rate as a % of the **initial principal** (the most common contract type). `assurance mensuelle = round(P · taux_assurance / 100 / 12)`, constant every month.
- **Fees** (optional amounts): *frais de dossier*, *frais de garantie* (caution or hypothèque).
- **Totals:**
  - `Intérêts totaux = Σ intérêts_k`
  - `Coût de l'assurance = Σ assurance_k`
  - **`Coût total du crédit` = intérêts totaux + coût de l'assurance + frais de dossier + frais de garantie**
  - `Montant total remboursé` = P + intérêts totaux + coût de l'assurance
  - `Mensualité totale` = mensualité hors assurance + assurance mensuelle
- **Dates:** installment `k` falls in month `first_payment_month + (k−1)`. Month precision only, no day.
- **Yearly view:** rows grouped by **calendar year** (sums of interest, principal and insurance, remaining principal at year end). The first and last years can be partial and show their number of installments, e.g. « 2026 (2 échéances) ».

### Reference values (become unit tests; no insurance, no fees)

| Montant | Taux | Durée | Mensualité | Intérêts totaux | Dernière échéance |
|---|---|---|---|---|---|
| 200 000 € | 3,50 % | 20 ans | 1 159,92 € | 78 380,66 € | 1 159,78 € |
| 150 000 € | 4,00 % | 25 ans | 791,76 € | 87 525,43 € | 789,19 € |
| 250 000 € | 3,20 % | 15 ans | 1 750,60 € | 65 108,61 € | 1 751,21 € |
| 100 000 € | 0 % | 10 ans | 833,33 € | 0,00 € | 833,73 € |

I computed these with the rules above. They will be cross-checked against one public French simulator before being frozen in tests. A ±1 € difference on the totals is expected, because simulators round differently.

Insurance example: 200 000 € at 0,30 % gives 50,00 €/month, 12 000,00 € over 20 years.

## Inputs and validation

| Champ (UI) | Field | Unit / storage | Rule | Default |
|---|---|---|---|---|
| Nom | `name` | text | required, 1–80 chars | « Prêt N » |
| Montant | `principal_cents` | € → cents | 1 000 € – 5 000 000 € | — |
| Taux nominal | `annual_rate_pct` | % | 0 – 15, max 3 decimals | — |
| Durée | `duration_months` | years in UI (integer) → months | 1 – 30 years | 20 ans |
| Taux assurance | `insurance_rate_pct` | % of initial principal / year | 0 – 2, max 3 decimals | 0 |
| Frais de dossier | `application_fee_cents` | € → cents | 0 – 50 000 € | 0 |
| Frais de garantie | `guarantee_fee_cents` | € → cents | 0 – 100 000 € | 0 |
| Première échéance | `first_payment_month` | `YYYY-MM` | valid month, 2000-01 – 2100-12 | the month after today |

Validation errors come back as `fields` in the shared error format, with French messages (e.g. « Le montant doit être compris entre 1 000 € et 5 000 000 € »).

## Data (`loans` section)

```yaml
loans:
  - id: 9f2c4e1a7b3d5f60          # 16 hex chars, crypto/rand
    name: Appart Lyon — Crédit Mutuel
    principal_cents: 20000000
    annual_rate_pct: 3.5
    duration_months: 240
    insurance_rate_pct: 0.3
    application_fee_cents: 100000
    guarantee_fee_cents: 250000
    first_payment_month: 2026-12
    created_at: 2026-11-12T20:41:00+01:00
    updated_at: 2026-11-12T20:41:00+01:00
```

Only inputs are stored. Results are always recomputed, so a rule fix applies to every saved loan. The validator is registered with `storage` and enforces the rules above, plus unique ids.

## API

| Route | Behavior |
|---|---|
| `POST /api/loans/simulate` | body = inputs → results + schedule, **not saved** (live preview) |
| `GET /api/loans` | list: id, name, principal, rate, duration, total monthly payment, total cost |
| `POST /api/loans` | create → `201` with the full loan |
| `GET /api/loans/{id}` | inputs + results + monthly schedule |
| `PUT /api/loans/{id}` | update inputs → full loan |
| `POST /api/loans/{id}/duplicate` | copy named « <nom> (copie) » → `201` |
| `DELETE /api/loans/{id}` | `204`, or `409 loan_in_use` if a delete guard refuses (message lists the properties) |

Result JSON (camelCase, cents):

```json
{
  "monthlyPaymentCents": 115992, "monthlyInsuranceCents": 5000, "monthlyTotalCents": 120992,
  "totalInterestCents": 7838066, "totalInsuranceCents": 1200000,
  "totalFeesCents": 350000, "totalCostCents": 9388066, "totalRepaidCents": 29038066,
  "schedule": [{"index": 1, "month": "2026-12", "paymentCents": 115992, "interestCents": 58333,
                "principalCents": 57659, "insuranceCents": 5000, "remainingCents": 19942341}]
}
```

**Hook for `property`:** `loan.Service.RegisterDeleteGuard(func(loanID string) error)`. `property` uses it to block deleting a loan that a property uses. `loan` never imports `property`.

## UI

**/prets: Prêts list.** Table with Nom, Montant, Taux, Durée, Mensualité (assurance incluse), Coût total du crédit. Clicking a row opens the loan. Buttons: **Nouvelle simulation**. Empty state: « Aucune simulation de prêt. » with the same button.

**/prets/nouveau and /prets/:id: Prêt page.**
- **Left, form:** Nom, Montant, Taux, Durée (ans, whole years only), Première échéance (month picker, « décembre 2026 »). A collapsible section « Assurance et frais » holds Taux assurance, Frais de dossier, Frais de garantie. Amounts are typed in euros with French formatting (`200 000`, `3,5`).
- **Right, live results** (debounced `simulate`, ~300 ms): Mensualité hors assurance, Assurance mensuelle, **Mensualité totale**, Intérêts totaux, Coût de l'assurance, Frais, **Coût total du crédit**, Montant total remboursé.
- **Actions:** **Enregistrer** (create/update), **Dupliquer**, **Supprimer** (confirmation; shows the French `loan_in_use` message if blocked). Leaving with unsaved changes asks for confirmation.
- **Tableau d'amortissement** below, with an **Annuel / Mensuel** toggle (annual by default). Columns: Année (« 2027 ») or Mois (« déc. 2026 »), Échéance, Intérêts, Capital remboursé, Assurance, Capital restant dû.
- **Graphique** between the results and the table, as **two charts stacked vertically that share the same years** (decided 2026-11-13, replacing a single chart with a second y-axis, which would make the two scales look comparable):
  - « Remboursement par année »: a **stacked bar chart, one bar per calendar year**, with **Capital remboursé** and **Intérêts** stacked (insurance not shown).
  - « Capital restant dû »: a **line chart** of the remaining principal at each year end, with its own y-axis.
  - Tooltip on hover gives the year and the three amounts in euros, on either chart. There's a French legend, colours that read well in light and dark mode (validated palette), and an accessible text alternative (`<title>`/`aria-label` summary plus the table below). It's built as a **hand-written SVG Vue component (`LoanChart.vue`), no chart library**: the chart is simple and this avoids a dependency.
- **« Biens liés »** section: a placeholder hidden until `property` exists. Then it lists the linked properties as links (filled through the property API).

## Acceptance Criteria

1. Every reference value in the table above is reproduced exactly by `internal/loan` tests.
2. In every schedule, the remaining principal ends at exactly 0, the sum of principal repaid equals the amount, and the sum of interest equals the total interest.
3. The Prêt page updates its results while typing, without saving. **Enregistrer** stores the loan in the YAML. After a restart, the loan and identical results are there.
4. Invalid inputs show French messages next to the fields, and nothing is saved. The API answers `400` with `fields`.
5. **Dupliquer** creates an independent copy. **Supprimer** removes the loan after confirmation, or shows the blocking message when a guard refuses.
6. The annual and monthly schedules are consistent: each calendar-year row equals the sum of its months, and the schedule starts at the chosen first-payment month.
8. The bar chart shows one bar per calendar year whose heights match the yearly table, the line chart below shows the remaining principal at each year end, and both update live with the form.
7. A loan edited by hand in the YAML with invalid values makes the app refuse to start (storage rule), and the message names the loan.

## Testing

- `internal/loan` maths: table-driven tests (reference values, month dating and calendar-year grouping with partial first and last years, 0 % rate, 1-month loan, 30-year loan, last-installment adjustment, insurance, fees, rounding at half a cent, invalid inputs). Coverage ≥ 90 %.
- Property-style checks over many generated inputs: remaining principal reaches 0, sums are consistent, and the payment is monotonic in rate and amount.
- Service: CRUD, duplicate, delete guard, persistence round trip through a temp `storage.Store`.
- Validator: each rule, with French messages.
- `httptest`: every route, `400` field errors, `404` unknown id, `409` loan in use, same-origin check on writes.
- Frontend: `LoanChart` (bar count = number of calendar years, bar heights proportional to the yearly sums, tooltip content, empty or invalid input shows no chart); money/rate inputs (French parsing `200 000`, `3,5`), debounced simulate, Annuel/Mensuel toggle, unsaved-changes guard, delete confirmation and `409` message (mocked API).

## Decisions (resolved)

- Insurance: a single basis, % of the initial principal (constant premium).
- Fees: *frais de dossier* and *frais de garantie* only.
- Duration: whole years only (1–30).
- Chart: yes, in v1 (hand-written SVG, no dependency): two charts sharing the years, no second y-axis.
- First-payment month: yes. The yearly view groups by calendar year.

## Open Questions

None.
