# Legal rates check (Task 26)

Checked on 2026-11-13 against official French government pages, for every value marked ⚠️ in `SPEC-property.md`. Figures are quoted from the pages; the date in brackets is the page's own « Vérifié le » / « Mis à jour le ».

| Spec value | Spec default | Official value | Status | Source |
|---|---|---|---|---|
| Prélèvements sociaux, revenus fonciers (location nue) | 17,2 % | **17,2 %** (CSG 9,2 % + CRDS 0,5 % + prélèvement de solidarité 7,5 %) | ✅ confirmed | service-public F2329 (30 juin 2026); F1991 (15 avril 2026) |
| Prélèvements sociaux, location meublée non professionnelle (micro-BIC) | 17,2 % (one global rate) | **18,6 %** (CSG 10,6 % + 0,5 % + 7,5 %), LFSS 2026 (loi n° 2025-1403 du 30 décembre 2025, art. 12) | ⚠️ **differs** | service-public F2329 (30 juin 2026) |
| Micro-foncier ceiling | 15 000 € | **15 000 €** | ✅ confirmed | service-public F1991 (15 avril 2026) |
| Micro-foncier allowance | 30 % | **30 %** | ✅ confirmed | service-public F1991 (15 avril 2026) |
| Micro-BIC ceiling, location meublée de longue durée | 77 700 € | **77 700 €** for revenus 2025, **83 600 €** for revenus 2026 | ⚠️ **differs** (for 2026) | service-public F32744 (15 avril 2026) |
| Micro-BIC allowance, location meublée de longue durée | 50 % | **50 %** (2025 and 2026) | ✅ confirmed | service-public F32744 (15 avril 2026) |
| DPE G no longer rentable | 1er janvier 2025 | **1er janvier 2025** | ✅ confirmed | ecologie.gouv.fr, page DPE (19 décembre 2025); service-public F2042 (21 mars 2025) |
| DPE F no longer rentable | 1er janvier 2028 | **1er janvier 2028** | ✅ confirmed | same |
| DPE E no longer rentable | 1er janvier 2034 | **1er janvier 2034** | ✅ confirmed | same |
| Rent freeze for F (« loyer gelé ») | F | **F and G**: no rent increase since August 2022 (new tenant, renewal, annual indexation) until works reach at least class E | ✅ confirmed (also applies to G) | ecologie.gouv.fr, FAQ « Location et gel des loyers des passoires énergétiques » (19 décembre 2025) |

Notes:
- Not to be confused with **meublés de tourisme non classés** (loi Le Meur): 15 000 € ceiling and 30 % allowance. The spec's `micro_bic` regime is long-term furnished rental, so this doesn't apply.
- The DPE bans apply to **new leases and renewals** from each date; a lease in progress is not affected until it renews. The spec's warnings are informational, so no change is needed.
- No official page mentions a 2026 postponement of the DPE calendar. A new DPE calculation method (January 2026) can move some homes to a better class; that changes a home's class, not the dates.

## Sources

- https://www.service-public.gouv.fr/particuliers/vosdroits/F2329
- https://www.service-public.gouv.fr/particuliers/vosdroits/F1991
- https://www.service-public.gouv.fr/particuliers/vosdroits/F32744
- https://www.service-public.gouv.fr/particuliers/vosdroits/F2042
- https://www.ecologie.gouv.fr/politiques-publiques/diagnostic-performance-energetique-dpe
- https://www.ecologie.gouv.fr/politiques-publiques/location-gel-loyers-passoires-energetiques

## Decisions (2026-11-13, by the user)

1. Social charges: two editable settings, « location nue » 17,2 % (micro-foncier, réel) and « location meublée » 18,6 % (micro-BIC). The micro-BIC reference values in `SPEC-property.md` were recomputed accordingly.
2. Micro-BIC ceiling: 83 600 € (revenus 2026).
