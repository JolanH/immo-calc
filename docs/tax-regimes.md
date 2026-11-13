# Tax regimes in immo-calc

How immo-calc taxes rental income, why the regimes give such different results, and where the rules come from. The calculation rules themselves (the source of truth for the code) are in [`SPEC-property.md`](../SPEC-property.md), *Calculation rules*. Checked against official sources on 2026-11-13.

immo-calc approximates French rules **realistically, not exactly**: it is a decision aid, not a tax return.

## The four regimes at a glance

| Regime (UI) | For | Taxable income | Social charges | Best when |
|---|---|---|---|---|
| **Micro-foncier** (`micro_foncier`) | Unfurnished, gross rent ≤ 15 000 €/year (all such biens together) | Rent collected **× 70 %** (flat 30 % allowance; no real charge deducted) | 17,2 % | Your real charges + loan interest are **below 30 %** of the rent |
| **Réel foncier** (`reel_foncier`) | Unfurnished, any amount (compulsory above 15 000 €) | Rent − real charges − year-1 loan interest and borrower insurance, floored at 0 | 17,2 % | Charges + interest **above 30 %** of the rent (typical with a loan) |
| **Micro-BIC** (`micro_bic`) | Long-term furnished, receipts ≤ 83 600 € (revenus 2026) | Rent collected **× 50 %** (flat 50 % allowance) | 18,6 % | Small furnished rentals with few charges and no loan |
| **LMNP au réel** (`lmnp_reel`) | Furnished (loueur en meublé non professionnel), any amount | Rent − real charges − loan interest/insurance − **depreciation**, floored at 0 | 18,6 % | Almost always for furnished rentals: depreciation often brings the taxable income to 0 |

The tax is `taxable income × (TMI + social charges)`. TMI and both social-charge rates are settings (Paramètres → Hypothèses et fiscalité).

## Why a « micro » regime can cost more

The micro regimes are a **simplification**, not an optimisation. They replace your real charges with a flat allowance (30 % or 50 % of the rent). If your real deductible charges plus loan interest are larger than that allowance, the réel regime taxes less.

Example, « SE appart 1 » (rent 360 €/month, 1 month of vacancy, taxe foncière 650 €, non-recoverable copropriété charges 700 €, TMI 30 %), rent collected 3 960 €/year, real charges 1 400 € (35 % of the rent):

| | Taxable income | Tax |
|---|---|---|
| Micro-foncier: 3 960 × 70 % | 2 772 € | ≈ 1 308 € |
| Réel foncier, no loan: 3 960 − 1 400 | 2 560 € | ≈ 1 208 € |
| Réel foncier with « Conso Immo » (≈ 678 € interest in year 1) | ≈ 1 882 € | ≈ 888 € |
| Micro-BIC (furnished): 3 960 × 50 % | 1 980 € | ≈ 962 € |
| LMNP au réel (furnished, no loan, no furniture): 2 560 − 1 060 depreciation | 1 500 € | ≈ 729 € |

Each Bien page shows this comparison for the bien in the collapsible « Régime fiscal » section.

## What is deductible under the réel regimes

Article 31 of the Code général des impôts (I, 1°, version in force on 13/11/2026) lists the deductible charges of urban properties. immo-calc uses:

- **Taxe foncière** — *c*: « les impositions, autres que celles incombant normalement à l'occupant ».
- **Insurance premiums** (landlord / PNO insurance, borrower insurance) — *a bis*: « les primes d'assurance ».
- **Charges de copropriété** — *a quater*: the provisions paid to the syndic, « diminuées du montant des provisions déduites l'année précédente qui correspond à des charges non déductibles ». BOFiP BOI-RFPI-BASE-20-30-30 §170–190 explains the yearly correction: items covered by the provision that are recoverable from the tenant (« réparations locatives ») are « en principe » not deductible and are reintegrated the following year.
  **So only the non-recoverable part is deductible in the end.** That is why the field is « Charges de copropriété non récupérables »: enter the part of the syndic's calls you keep paying, and put the recoverable part in « Charges récupérables » (passed on to the tenant, neutral for tax).
- **Recoverable charges paid during vacancy** — *a ter*: « les dépenses supportées pour le compte du locataire […] dont [le propriétaire] n'a pu obtenir le remboursement ».
- **Letting-agency fees** — management costs (BOFiP BOI-RFPI-BASE-20-10).
- **Loan interest and borrower insurance** — immo-calc uses the first 12 installments (year 1) and keeps that tax over time.

Not modelled: works (travaux), the déficit foncier carried to global income, the micro-foncier « 50 % » variants for specific schemes.

## LMNP au réel

Furnished rentals under the réel regime are taxed as BIC with an accounting result: receipts − real charges − loan interest/insurance − **depreciation (amortissements)**. Depreciation is what makes the regime attractive:

- **Building**: the price excluding furniture, **minus a 15 % land share** (land doesn't wear out), plus the notary fees, over **30 years**.
- **Furniture**: over **7 years** — both furniture *included in the price* and furniture *bought on top*.
- Depreciation **cannot create a deficit**: it is used only up to the result (immo-calc doesn't carry the unused part forward).

Rates and durations follow common practice (building 25–40 years, furniture 5–10 years, land 10–20 %); immo-calc fixes 15 %, 30 years and 7 years.

### Furniture already in the flat

When the flat is sold furnished, the deed lists the furniture and its value. In immo-calc:

- **« Mobilier compris dans le prix »**: part of the price. It is **excluded from the notary-fee base** (the transfer taxes, the bulk of the « frais de notaire », don't apply to furniture), for every regime, and it is depreciated over 7 years under LMNP au réel.
- **« Mobilier acheté en plus »**: added to the acquisition cost (paid with own funds), depreciated over 7 years under LMNP au réel.

Reference values are in `SPEC-property.md` (*LMNP au réel reference*).

## Simplifications to keep in mind

- **Year-1 tax kept over time** — the profitability timeline keeps the year-1 tax (interest decreases, furniture depreciation stops after 7 years in reality).
- **Resale is out of scope** — capital gains, and since 2025 the reintegration of LMNP depreciation into the gain.
- **LMP not modelled** — the professional status (loueur meublé professionnel) and its thresholds are not checked.
- **Micro ceilings** — they are checked across all biens of the regime and shown as warnings; immo-calc doesn't switch the regime automatically.

## Sources

- Légifrance — Code général des impôts, article 31 (version in force on 13/11/2026): https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000030059714
- BOFiP — BOI-RFPI-BASE-20-30-30 (provisions pour charges de copropriété): https://bofip.impots.gouv.fr/bofip/4052-PGP.html/identifiant=BOI-RFPI-BASE-20-30-30-20151228
- BOFiP — BOI-RFPI-BASE-20-10 (frais d'administration et de gestion): https://bofip.impots.gouv.fr/bofip/5804-PGP.html/identifiant=BOI-RFPI-BASE-20-10-20160706
- service-public F1991 — revenus locatifs, location non meublée (micro-foncier 15 000 €, 30 %; deductible charges under the réel): https://www.service-public.gouv.fr/particuliers/vosdroits/F1991
- service-public F32744 — location meublée (micro-BIC 50 %, 77 700 € / 83 600 €): https://www.service-public.gouv.fr/particuliers/vosdroits/F32744
- service-public F2329 — prélèvements sociaux (17,2 % / 18,6 %): https://www.service-public.gouv.fr/particuliers/vosdroits/F2329
- LMNP charges and depreciation practice (secondary sources): https://www.jedeclaremonmeuble.com/charges-lmnp-deductibles-amortissables-recuperables/ · https://www.bailfacile.fr/guides/charges-deductibles-lmnp
- Earlier rate checks: [`tasks/legal-rates.md`](../tasks/legal-rates.md)
