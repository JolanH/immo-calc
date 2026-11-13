# Spec: property

> Module of [immo-calc](SPEC.md). Depends on: `storage`, `loan`. Shared conventions are in `SPEC.md`.
> The rules approximate French rules realistically, not exactly. The tax rates, ceilings and DPE dates were checked against official sources on 2026-11-13; the findings and links are in `tasks/legal-rates.md`.

## Objective

Let the user enter a flat they are considering buying to rent out, and immediately see whether it is a good investment: **Revenus mensuels bruts, Revenus mensuels après impôts, Revenu annuel net, Rentabilité**. Optionally the flat is linked to a saved loan simulation, to see the cash-flow and the return on the user's own contribution. Navigation between a flat and its loan is one click in each direction.

## Scope

**In:**
- Property CRUD and duplication
- The `properties` and `settings` sections and their validators
- Profitability and tax maths (`internal/property`), including rent vacancy, letting-agency fees and landlord insurance (PNO)
- Global settings (*Paramètres → Hypothèses et fiscalité*)
- The loan link (select, create a loan from the property, navigate both ways, delete guard)
- DPE rental-ban warnings
- API and Vue pages

**Out (v1):**
- Rental deficit (*déficit foncier*) carried to global income
- LMNP *au réel* (depreciation)
- Capital gains on resale
- Multi-year projections and IRR
- Works (*travaux*)
- A per-property notary rate (neuf/ancien)

## Inputs (per property)

| Champ (UI) | Field | Unit | Rule | Default |
|---|---|---|---|---|
| Nom | `name` | text | required, 1–80 | — |
| Prix | `price_cents` | € (net vendeur) | 10 000 – 5 000 000 € | — |
| DPE | `dpe` | `A`…`G` | required | — |
| Loyer | `monthly_rent_cents` | €/mois, **hors charges** | 1 – 20 000 € | — |
| Charges | `monthly_recoverable_charges_cents` | €/mois | 0 – 2 000 € | 0 |
| Taxe foncière | `annual_property_tax_cents` | €/an | 0 – 50 000 € | — |
| Charges de copropriété | `annual_condo_fees_cents` | €/an | 0 – 50 000 € | 0 |
| Assurance PNO | `annual_landlord_insurance_cents` | €/an | 0 – 5 000 € | 0 |
| Gestion par une agence | `agency_managed` | oui/non | — | non |
| Régime fiscal | `tax_regime` | enum | `micro_foncier`, `reel_foncier`, `micro_bic` | `micro_foncier` |
| Prêt | `loan_id` | id or empty | must exist in `loans` | empty |

*Charges* are paid by the tenant and passed on to the providers (eau chaude, chauffage, ordures…). *Charges de copropriété* are the owner's annual cost paid to the syndic. The two are distinct.

## Global settings (`settings` section)

| Paramètre | Field | Default | Rule |
|---|---|---|---|
| Tranche marginale d'imposition (TMI) | `marginal_tax_rate_pct` | 30 | one of 0, 11, 30, 41, 45 |
| Prélèvements sociaux — location nue | `social_charges_unfurnished_pct` | 17.2 | 0 – 30. Used by `micro_foncier` and `reel_foncier` (service-public F2329) |
| Prélèvements sociaux — location meublée | `social_charges_furnished_pct` | 18.6 | 0 – 30. Used by `micro_bic` (CSG raised to 10,6 % by the LFSS 2026; service-public F2329) |
| Frais de notaire | `notary_fee_pct` | **8.5** | 0 – 15, % of the price |
| Vacance locative | `vacancy_months_per_year` | 1 | 0 – 12, step 0.5 (months without a tenant per year) |
| Frais de gestion locative | `agency_fee_pct` | 7 | 0 – 15, % of rent collected (TTC). Applies to properties with *Gestion par une agence* = oui |

## Calculation rules

Notation: `L` = monthly rent excluding charges, `C` = monthly recoverable charges, `TF` = annual property tax, `CP` = annual condo fees, `PNO` = annual landlord insurance, `v` = vacancy months/year, `g%` = agency fee rate (0 if the property isn't agency-managed), `N%` = notary rate, `T` = TMI + the regime's social charges (location nue for `micro_foncier` and `reel_foncier`, location meublée for `micro_bic`).

1. **Coût d'acquisition** `A = Prix × (1 + N%)`
2. **Loyers encaissés** (annual) `E = L × (12 − v)`
3. **Charges du propriétaire** (annual) `CH = TF + CP + PNO + E × g% + C × v`. During vacancy, the owner pays the recoverable charges no tenant reimburses.
4. **Revenu annuel avant impôts** `R = E − CH`
5. **Revenu imposable** (annual, floored at 0):
   - `micro_foncier`: `E × 70 %`. Warn if `E` summed over all `micro_foncier` properties is over **15 000 €** (service-public F1991).
   - `micro_bic` (furnished long-term): `E × 50 %`. Warn if `E` summed over all `micro_bic` properties is over **83 600 €** (revenus 2026; 77 700 € for revenus 2025; service-public F32744).
   - `reel_foncier`: `R − intérêts année 1 − assurance emprunteur année 1` (from the linked loan's first 12 installments, or 0 without a loan). A deficit is floored at 0 and not carried over.
6. **Impôt annuel** `I = imposable × T`
7. **Revenu annuel net** (after charges **and** taxes) `RN = R − I`
8. **Revenus mensuels bruts** = `L`
9. **Revenus mensuels après impôts** = `RN / 12`
10. **Rentabilité**, all on `A`:
    - brute = `12·L / A`
    - nette (de charges) = `R / A`
    - **nette après impôts** (highlighted) = `RN / A`

**With a linked loan** (« Financement » card):

11. **Mensualité** = the loan's total monthly payment (insurance included)
12. **Cash-flow mensuel avant impôts** = `R/12 − Mensualité`
13. **Cash-flow mensuel après impôts** = `RN/12 − Mensualité`
14. **Apport** = `A` + loan fees − loan amount. If ≤ 0, the return on contribution shows « n/a ».
15. **Rendement sur apport** = `12 × cash-flow après impôts / Apport`
15b. **Rentabilité nette après crédit** (added 2026-11-13, the headline KPI for choosing between biens and loans) = `(RN − coût annuel moyen du crédit) / A`, where the average annual credit cost = the loan's **coût total du crédit** (interest + insurance + fees) × 12 / its duration in months, rounded to the cent. Principal repaid is not a cost (it builds equity). **Without a loan** the credit cost is 0, so it equals the « nette après impôts » yield, which keeps every bien comparable. Shown highlighted on the Bien page (Rentabilité card), as a column of the Biens lists (`/biens` and the home page, which uses the active loan) and as a highlighted row of the comparison (following each column's loan).
  Reference: « SE appart 1 » with « Conso Immo » (coût total 1 984,13 € over 5 years → 396,83 €/an): (1 251,62 − 396,83) / 36 890 = **2,32 %** (3,39 % without the loan).

**Dates de rentabilité** (added 2026-11-13 at the user's request; extended the same day to biens without a loan). A month-by-month timeline starting at the loan's first installment or, **without a loan, the month after today** (the default first installment of a new loan):

16. **Cash-flow of month k** = `RN/12` − (installment k + insurance k) while the loan runs (without a loan: `RN/12` every month), then `RN'/12` after the last installment, where `RN'` is the net annual income without the loan (under `reel_foncier` the interest deduction stops; the micro regimes are unchanged).
17. **Cumulative position** starts at −Apport (without a loan, Apport = `A`: everything is paid with own funds) and adds each month's cash-flow.
18. **« Cash-flow mensuel positif »** = the first month whose cash-flow is > 0; « Immédiat » when that is the first month, « Jamais » if it never is.
19. **« Apport récupéré »** = the first month the cumulative position is ≥ 0; « Dès l'achat » when the apport isn't positive, « Au-delà de 50 ans » if not reached within 50 years.
20. Dates are months; the UI shows them as `01/MM/YYYY`. Without a loan the dates depend on today's date (the start month moves forward).
  Reference without a loan: « SE appart 1 » unfinanced, evaluated in November 2026 (start 12/2026, apport 36 890 €, +104,30 €/month): cash-flow mensuel positif « Immédiat », apport récupéré **01/05/2056**.

Reference (user's case): « SE appart 1 » (prix 34 000 €, loyer 360 €, charges 50 €, TF 650 €, CP 700 €, micro-foncier, revenus mensuels après impôts 104,30 €) with « Conso Immo » (15 000 € à 5 % sur 5 ans, first installment 01/2027, apport 21 890 €): cash-flow mensuel positif **01/01/2032**, apport récupéré **01/01/2058**.

Rounding: compute in `float64`, round each displayed amount to the cent and each rate to 2 decimals.

### Reference example (becomes a unit test)

Prix 150 000 €, Loyer 700 €, Charges 50 €, TF 900 €, CP 1 200 €, PNO 150 €. Settings: TMI 30 %, PS location nue 17,2 %, PS location meublée 18,6 %, notaire 8,5 %, vacance 1 mois, gestion 7 %. No loan.

Common figures: A = 162 750 €, E = 7 700 €, rentabilité brute = 5,16 %.

**Without an agency** (CH = 2 300 €, R = 5 400 €, nette = 3,32 %):

| | micro-foncier | réel | micro-BIC |
|---|---|---|---|
| Revenu imposable | 5 390,00 € | 5 400,00 € | 3 850,00 € |
| Impôt annuel | 2 544,08 € | 2 548,80 € | 1 871,10 € |
| **Revenu annuel net** | 2 855,92 € | 2 851,20 € | 3 528,90 € |
| **Revenus mensuels après impôts** | 237,99 € | 237,60 € | 294,08 € |
| **Rentabilité nette après impôts** | 1,75 % | 1,75 % | 2,17 % |

**With an agency** (frais de gestion 539 €, R = 4 861 €, nette = 2,99 %):

| | micro-foncier | réel | micro-BIC |
|---|---|---|---|
| Revenu imposable | 5 390,00 € | 4 861,00 € | 3 850,00 € |
| Impôt annuel | 2 544,08 € | 2 294,39 € | 1 871,10 € |
| **Revenu annuel net** | 2 316,92 € | 2 566,61 € | 2 989,90 € |
| **Revenus mensuels après impôts** | 193,08 € | 213,88 € | 249,16 € |
| **Rentabilité nette après impôts** | 1,42 % | 1,58 % | 1,84 % |

**With a linked loan** (same flat, no agency; computed with the loan rules, first-year figures = first 12 installments):

| Loan | Regime | Revenu imposable | Impôt annuel | Mensualité | Cash-flow avant / après impôts | Apport | Rendement sur apport |
|---|---|---|---|---|---|---|---|
| 150 000 € à 4 % sur 25 ans (loan reference 2), no insurance, no fees. Intérêts année 1 : 5 935,09 € | micro-foncier | 5 390,00 € | 2 544,08 € | 791,76 € | −341,76 € / −553,77 € | 12 750,00 € | −52,12 % |
| same | réel | 0,00 € (5 400 € − 5 935,09 € floored at 0) | 0,00 € | 791,76 € | −341,76 € / −341,76 € | 12 750,00 € | −32,17 % |
| 50 000 € à 3 % sur 20 ans, assurance 0,30 %, 1 000 € de frais. Intérêts année 1 : 1 474,66 €, assurance : 150,00 € | réel | 3 775,34 € | 1 781,96 € | 289,80 € | 160,20 € / 11,70 € | 113 750,00 € | 0,12 % |
| 200 000 € à 3,5 % sur 20 ans | any | — | — | 1 159,92 € | — | ≤ 0 | n/a |

## DPE warnings (loi Climat et résilience)

| DPE | Badge | Message |
|---|---|---|
| G | red | « Logement G : interdit à la location depuis le 1er janvier 2025 » |
| F | orange | « Logement F : interdit à la location à partir du 1er janvier 2028 ; loyer gelé » |
| E | yellow | « Logement E : interdit à la location à partir du 1er janvier 2034 » |
| A–D | neutral | — |

Warnings are informational and don't change the maths. Dates checked on ecologie.gouv.fr (page DPE) and service-public F2042; the bans apply to new leases and renewals from each date.

## Data

```yaml
settings:
  marginal_tax_rate_pct: 30
  social_charges_unfurnished_pct: 17.2
  social_charges_furnished_pct: 18.6
  notary_fee_pct: 8.5
  vacancy_months_per_year: 1
  agency_fee_pct: 7
properties:
  - id: 3a7e0c91d24b6f58
    name: T2 Lyon 7e
    price_cents: 15000000
    dpe: D
    monthly_rent_cents: 70000
    monthly_recoverable_charges_cents: 5000
    annual_property_tax_cents: 90000
    annual_condo_fees_cents: 120000
    annual_landlord_insurance_cents: 15000
    agency_managed: false
    tax_regime: micro_foncier
    loan_id: 9f2c4e1a7b3d5f60     # optional
    created_at: …
    updated_at: …
```

Only inputs are stored, and results are always recomputed. A missing `settings` section means the defaults apply. Validators enforce the rules above, and that `loan_id` exists. A dangling `loan_id` in a hand-edited file means the app refuses to start (storage rule).

## Loan link and navigation

- On the property form, a **Prêt** field selects one of the saved loans (or « Aucun »). There's a **Voir le prêt →** link and a **Créer un prêt pour ce bien** button.
- **Créer un prêt pour ce bien** → `/prets/nouveau?bien=<id>`, with *Montant* prefilled with `A` and *Nom* with « Prêt – <nom du bien> ». On **Enregistrer**, the loan is created **and** linked to the property. Then a « ← Retour au bien » link is shown.
- On the Prêt page, **Biens liés** lists the properties using the loan, as links (`GET /api/properties?loanId=`).
- Deleting a loan that a property uses is refused (`409 loan_in_use`) through `loan.RegisterDeleteGuard`. The message names the properties.
- Several properties may share one loan (allowed but unusual).

## API

| Route | Behavior |
|---|---|
| `POST /api/properties/evaluate` | body = inputs → results + warnings (live preview, not saved) |
| `GET /api/properties[?loanId=][&activeLoanId=]` | list with key results; `activeLoanId` finances every bien with that loan as saved (home page) |
| `POST /api/properties` | create → `201` |
| `GET /api/properties/{id}` | inputs + results + warnings + linked loan summary |
| `PUT /api/properties/{id}` | update |
| `POST /api/properties/{id}/duplicate` | copy « <nom> (copie) » |
| `DELETE /api/properties/{id}` | `204` |
| `GET /api/settings`, `PUT /api/settings` | global settings; all results recompute from them |

## Home page (added 2026-11-13)

The home page « Accueil » (`/`) has two panes:

- **Left (~80 % width): the Biens list**, the same columns as `/biens`, with **Nouveau bien**.
- **Right (~20 % width): a simplified list of saved loans**: name, **amount and duration** (« 200 000 € · 20 ans ») and monthly total, each with a link to its loan page. One **active loan** (or « Aucun ») is picked by clicking anywhere on its row; the active row is highlighted. The rows are a radio group underneath, so the choice also works with the keyboard and screen readers. The selection is session-only (server memory, see `SPEC-loan.md`).

**Active loan and profitability:** while an active loan is selected, it **overrides** every bien's own linked loan in the home page's Biens list: the Financement figures shown there (Mensualité, Cash-flow, Apport, Rendement sur apport, and the `reel_foncier` interest deduction) use the active loan for every bien. With « Aucun », each bien uses its own linked loan, as on `/biens`. The bien detail page and `/biens` always use the bien's own link.

**Borrowed amount (decided 2026-11-13, revised the same day):** the active loan applies to every bien **exactly as saved** (its amount, rate, duration, insurance and fees). So the « apport » is `A` + fees − the loan's amount (« n/a » when not positive), and the figures match what the right pane shows. The Prêt column of the home list then shows the active loan. A first version used the loan as a template borrowing each bien's acquisition cost; the user found it misleading (a 15 000 € loan silently became 36 890 €) and chose this rule instead.

## Comparison (added 2026-11-13)

Up to **4 biens** side by side at `/biens/comparer?biens=<id>,<id>…&prets=<loanId|>,…`.

- **Selecting:** a checkbox column in the Biens list and in the home page's Biens pane. From the home page, while a loan is active, every column starts on the active loan (as on the home list); otherwise each bien starts on its own loan. Once 4 are ticked the others are disabled. A **Comparer (n)** button opens the comparison when at least 2 are ticked. On the comparison page a column can be removed, and a bien added while there are fewer than 4.
- **One column per bien, one row per figure:** Prix, Coût d'acquisition, Loyer, Régime fiscal, DPE (badge); Revenus mensuels bruts, Revenus mensuels après impôts, Revenu annuel net, Rentabilité brute / nette / après impôts; Prêt (selector), Mensualité, Cash-flow avant / après impôts, Apport, Rendement sur apport; Cash-flow mensuel positif, Apport récupéré; Avertissements.
- **A distinct loan per bien:** each column has a « Prêt » selector (« Aucun » or any saved loan), defaulting to the bien's own loan. The chosen loan applies **as saved** (rules 11–20) for that column only. It is **not saved** on the bien; it lives in the URL (`prets`), so a reload keeps the comparison.
- Each column is computed with `POST /api/properties/evaluate?id=<bien>` and the chosen `loanId`, so the micro ceilings still count every saved bien once. A bien that no longer exists shows « Bien introuvable » in its column.

## UI

- **/biens: Biens list.** Table with Nom, Prix, Loyer, DPE badge, Rentabilité nette après impôts, Revenus mensuels après impôts, Cash-flow après impôts (if there's a loan), Prêt (link). Button **Nouveau bien**. Empty state message.
- **/biens/nouveau and /biens/:id: Bien page.**
  - Form on the left (inputs above).
  - Live results on the right (debounced `evaluate`) as cards: **Revenus mensuels bruts**, **Revenus mensuels après impôts**, **Revenu annuel net**, **Rentabilité** (brute / nette / **après impôts**).
  - A collapsible « Détail du calcul » lists A, E, each owner charge, R, revenu imposable and impôt, so the figures can be checked.
  - With a loan, the « Financement » card shows Mensualité, Cash-flow avant/après impôts, Apport and Rendement sur apport.
  - A **« Date de rentabilité »** result card shows « Cash-flow mensuel positif : DD/MM/YYYY » and « Apport récupéré : DD/MM/YYYY » (rules 16–20), and a **« Trésorerie cumulée »** chart below the Financement card plots the cumulative position over time with a zero line and both dates marked.
  - The DPE warning and regime-ceiling warnings appear above the results.
  - Actions: Enregistrer, Dupliquer, Supprimer (confirmation), unsaved-changes guard.
- **/parametres: Hypothèses et fiscalité** section (TMI, PS location nue, PS location meublée, notaire, vacance, gestion), next to the *Données* section from `storage`.

## Acceptance Criteria

1. Both reference tables are reproduced exactly for the three regimes.
2. Entering a flat shows the four requested outputs live. Saving stores it in the YAML, and after a restart the flat and identical results are there.
3. Changing any global setting (TMI, notary rate, vacancy, agency rate…) changes every affected property's results.
4. Linking a loan shows the Financement card. Under `reel_foncier`, the year-1 interest and borrower insurance reduce the taxable income.
5. **Créer un prêt pour ce bien** opens a prefilled loan form. Saving creates and links the loan and offers a way back. The loan page lists its linked properties.
6. Deleting a loan in use is refused with a message naming the property. Deleting a property leaves the loan intact.
7. DPE G/F/E show the right warning. Going over a micro ceiling across all flats on that regime shows a warning.
8. Invalid inputs give French field messages and nothing is saved.
9. Before implementation, the defaults (PS rate, micro ceilings and allowances, DPE dates) are checked against official sources, and the sources are cited in a comment next to the constants.

## Testing

- `internal/property` maths: table-driven tests (both reference tables × 3 regimes, with a loan, vacancy 0 and 12, agency on/off, floor at 0, apport ≤ 0, ceilings summed across properties, every DPE). Coverage ≥ 90 %.
- Validators and settings: each rule, unknown `loan_id`, defaults when `settings` is missing, French messages.
- Service: CRUD, duplicate, settings change propagates, delete guard registered with `loan`, persistence round trip through a temp store.
- `httptest`: every route, `?loanId=` filter, `400`/`404`/`409`.
- Frontend: Bien form (French number parsing, debounced evaluate, warnings, Financement card and « Détail du calcul »), "create loan for this property" round trip and return link, settings form (mocked API).

## Decisions (resolved)

- Revenu annuel net = after owner charges **and** taxes.
- Rentabilité is calculated on price + notary fees.
- *Charges* are neutral (except during vacancy). The full copro amount is an owner cost.
- Regimes: micro-foncier, réel foncier, micro-BIC.
- Vacancy, letting-agency fees and PNO are included.
- Notary fees: one global rate, 8,5 %, configurable.
- Legal rates are checked against official sources before implementation.
- Defaults confirmed: vacancy 1 month/year (global), agency fees 7 % (global rate, enabled per property).

## Open Questions

None. The vacancy (1 month/year) and agency fee (7 %) defaults are confirmed.
