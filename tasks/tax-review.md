# Adversarial review: tax calculation per regime (2026-11-13)

Scope: `internal/property/calc.go` (`Evaluate`, `taxFor`, `compareRegimes`, `notaryFees`), `financing.go` (year-1 interest), `settings.go` (defaults), `docs/tax-regimes.md`, `tasks/legal-rates.md`.

## Status (2026-11-13): R1, R2, R3, R5 and R6 fixed; R4 not done (by decision)

| Finding | Fix |
|---|---|
| R1 | Micro regimes tax rent + reimbursed charges (`microReceipts`); the micro ceilings (warning and eligibility) use the same receipts |
| R2 | Loan application and guarantee fees are deducted in year 1 under réel foncier and LMNP réel |
| R3 | The tax is computed per year from the loan schedule (`LoanFigures.YearlyInterest/YearlyInsurance`); the timeline uses each year's net income; LMNP depreciation stops after 7 (furniture) and 30 (building) years. Headline figures and the comparison stay year 1 |
| R5 | « Best » is given separately to the best unfurnished and the best furnished regime |
| R6 | `Eligible` is false for a micro regime above its ceiling; it can't be « Best » and the modal shows « plafond dépassé » |

Reference figures in `SPEC-property.md` and `docs/tax-regimes.md` were recomputed by hand (micro-foncier of the reference flat: 5 775 € taxable, 2 725,80 € tax). Eligibility is per bien: the ceilings summed across biens remain a warning.
Still open: O1–O5 (deficit carry-over, 305 € minimum, constant TMI, notary fees base, timing of 18,6 %), and R4.

The sections below are the original findings, as written before the fixes.

## Method and limits

- Read the code and ran a probe (temporary test, removed; working tree is clean) on a typical case: price 150 000 €, rent 800 €/month, recoverable charges 50 €/month, TF 900 €, non-recoverable condo 600 €, PNO 150 €, agency 7 %, vacancy 1 month, TMI 30 %, loan 150 000 € at 4 % over 25 years (year-1 interest 5 935,09 €, 1 500 € fees).
- Compared against the official sources named in `docs/tax-regimes.md` and against web results (searched 2026-11-13).
- **Not done:** I did not run any online simulator, because they need interactive input. I did not open Légifrance or BOFiP directly. Points marked *secondary source* come from search summaries and commercial sites, so confirm them before acting. The micro-BIC 305 € minimum and the micro receipts base should be checked against CGI art. 50-0 and BOFiP.
- The Go suite was not run. `go` is only available through `make`/Docker, and I ran only the probe.

## Probe results (current behaviour)

| Regime | Loan | Taxable | Tax | Net after tax |
|---|---|---|---|---|
| micro_foncier | any | 6 160 € | 2 907 € | 3 576 € |
| reel_foncier | no | 6 484 € | 3 060 € | 3 423 € |
| reel_foncier | yes | 548 € | 259 € | 6 224 € |
| micro_bic | any | 4 400 € | 2 138 € | 4 345 € |
| lmnp_reel | no | 1 809 € | 879 € | 5 604 € |
| lmnp_reel | yes | 0 € | 0 € | 6 484 € |

Collected rent E = 8 800 €, owner charges CH = 2 316 €, R = 6 484 €. The formulas match the SPEC. The differences below concern what the model leaves out.

## Findings

### Required (these push the figures away from real cases)

**R1. Micro regimes tax the rent only, not the charges the tenant reimburses.**
`calc.go:252` and `calc.go:266` use `CollectedRentCents × share`.
- Under micro-foncier and micro-BIC, the gross receipts are what the tenant pays you, charges reimbursed included. The flat allowance replaces all deductions. *Secondary reading of the rule, to confirm in BOFiP BOI-RFPI-DECLA-10-10 and BOI-BIC-DECLA-10-10.*
- The réel regimes are unaffected: the receipt and the charge cancel out.
- In the probe, C = 50 €/month for 11 occupied months adds 550 € of receipts. That means about 385 € of taxable income and about 182 € of tax a year (≈ 6 % of the micro tax). The micro regimes therefore look better than they are.
- The same base should feed the micro ceiling check (15 000 € / 83 600 €).
- Fix: add reimbursed charges `C × (12 − v)` to the micro base. Add one test per micro regime.

**R2. Loan fees are not deducted under the réel regimes.**
`FeesCents` (application and guarantee fees) is never used in `taxFor`.
- The web sources agree that interest, insurance, application fees and guarantee fees are deductible under réel foncier and LMNP réel. Where the guarantee fee is classed (financing or acquisition) is disputed. *Secondary source.*
- In the probe, fees of 1 500 € would give about 700 € less tax in year 1 under réel foncier.
- The omission makes réel look worse than it is in year 1, and the loan cost is counted only in the credit cost.
- Fix: deduct `FeesCents` in year 1 only. Because the timeline keeps year-1 tax forever (R3), the deduction should not be repeated in later years.

**R3. Year-1 tax is repeated for the whole timeline. This flatters réel foncier and LMNP.**
`docs/tax-regimes.md` documents this, but its effect on a loan is large.
- Year-1 interest (5 935 €) falls every year and reaches 0 at the end. The réel taxable income of 548 € in the probe rises towards about 6 500 € (réel foncier) once the loan is repaid.
- The micro regimes are not affected. As a result, `compareRegimes`, the payback dates and the break-even dates (« cash-flow positif », « apport récupéré ») are all biased toward réel and LMNP.
- Fix options: compute the tax per year from the loan schedule (a real model), or at least label the timeline « impôt de l'année 1 » next to the dates and the comparison.

**R4. LMNP au réel ignores its running costs, so it looks too good.**
- A réel LMNP needs a chartered accountant or a management centre (typically a few hundred euros a year, deductible) and the CFE, which is due above the minimum threshold. Neither is modelled, even as optional settings. *Secondary source.*
- In the probe, LMNP with a loan gives tax 0 and net 6 484 €, which is the most flattering figure in the app. With about 400 € of accountant fees the net would be about 6 084 €, though the tax stays 0.
- The user's « strange values » are likely to come from here.
- Fix: add an optional annual « frais de comptabilité / CFE » field per bien, deductible under LMNP réel.

**R5. The regime comparison applies one rent to furnished and unfurnished regimes.**
`compareRegimes` evaluates all four regimes on the same `MonthlyRentCents`.
- The same flat rents out at a higher rent when furnished (commonly 10–25 % more).
- Micro-foncier versus micro-BIC on the same rent therefore compares non-comparable situations. The « Best » tag can steer a user toward a regime that needs different rent, furniture, and lease terms.
- Fix: group the comparison (nue: micro-foncier / réel; meublé: micro-BIC / LMNP réel), or let the user enter a furnished rent for the comparison.

**R6. Comparison offers ineligible regimes.**
- `compareRegimes` marks « Best » on micro-foncier even above 15 000 €. The ceiling is only a separate warning.
- Fix: flag ineligible rows and exclude them from « Best ».

### Optional (real differences, small or documented)

**O1. Deficits are floored at 0, with no carry-over.** `calc.go:269`.
- Foncier: the deficit from non-interest charges is imputable on the overall income up to 10 700 €. The interest deficit is carried over 10 years against property income.
- LMNP: the BIC deficit is carried over 10 years. Depreciation that could not be used is carried over without a time limit.
- Effect: the model shows the tax benefit only as a tax of 0. Documented, so no fix required.

**O2. Micro-BIC allowance: minimum of 305 €** (CGI art. 50-0, single source in my search, to confirm). Irrelevant above about 610 € of receipts, but it matters for a tiny rent.

**O3. TMI treated as constant.**
- The model ignores the 6,8 points of deductible CSG on foncier income (art. 154 quinquies, deducted from the next year's income).
- It ignores the bracket effect, because adding the rent can change the household's bracket.
- Both effects are small and should stay documented as simplifications.

**O4. LMNP building base includes 100 % of the notary fees.** `calc.go:259`.
- The base is price × 85 % + all the fees, so the fees on the land share are depreciated too.
- Common practice, but an inspector could challenge it.
- Alternatively, notary fees can be deducted in year 1 (option available when the activity is declared at acquisition; *secondary source*). Worth a line in `docs/tax-regimes.md`.

**O5. Timing of the 18,6 %.** Sources disagree on whether it applies to 2025 receipts (filed in 2026) or from 2026. The default is correct for the horizon of a purchase today.

### Confirmed correct

- Social charges: 17,2 % for location nue and 18,6 % for LMNP and micro-BIC after LFSS 2026 (search results concur; official F2329 cited in `legal-rates.md`).
- Micro-foncier 30 % allowance and 15 000 € ceiling. Micro-BIC 50 % and 83 600 €. Meublé de tourisme non classé (30 %, 15 000 €) is correctly out of scope.
- Réel foncier deducts interest and borrower insurance. Condo charges are limited to the non-recoverable part. Agency fees are deducted.
- The depreciation order is right: depreciation does not create a deficit.
- The tax splits correctly as IR + PS = taxable × (TMI + social charges), with rounding done at each step.

## Tests

- The tests reproduce the SPEC reference tables, which were produced by the same rules. They cannot catch an error in the rules themselves, such as R1 or R2.
- Missing:
  - A micro case with `MonthlyRecoverableChargesCents > 0` and a hand-checked tax.
  - A réel case with loan fees.
  - A case where the timeline's tax changes over the years.
- If R1 is confirmed, add the cases above before changing the code.

## Suggested order

R1 → R2 → R5/R6 (small, local) → R4 (new field) → R3 (largest change, needs a per-year model).

## Sources

- https://www.service-public.gouv.fr/particuliers/vosdroits/F2329, F1991, F32744 (already cited in `docs/tax-regimes.md`)
- https://www.hagnere-patrimoine.fr/guides-patrimoine/investissement-immobilier/lfss-2026-prelevements-sociaux-186-lmnp-impact-rendement (secondary)
- https://www.jedeclaremonmeuble.com/lmnp-faq-sur-les-charges-et-amortissements-deductibles-au-regime-reel/ (secondary)
- https://www.investissement-locatif.com/abattement-lmnp.html (secondary)
