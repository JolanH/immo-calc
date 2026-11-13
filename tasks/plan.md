# Implementation Plan: immo-calc v1

## Overview

Build immo-calc v1 as described in `SPEC.md` and its four module specs: a single Go executable (Linux/Windows amd64) that embeds a French Vue 3 UI, serves it on `127.0.0.1`, stores everything in a versioned YAML file, simulates mortgages and evaluates the profitability of buy-to-let flats, with the two linked.

Work follows the approved build order **app-shell → storage → loan → property**, one phase per module, each ending with a human-reviewed checkpoint. Inside each phase, tasks are vertical slices: every task leaves `make test` and `make lint` green and delivers something you can observe (an endpoint, a page, a binary).

Detailed tasks, acceptance criteria and verification steps: **`tasks/todo.md`**.

## Dependency Graph

```
Docker images + Makefile ─┐
                          ├─ Go server skeleton (router, JSON errors, /api/health)
                          │     ├─ flags/config, version, cross-compile
                          │     ├─ port selection / single instance / browser open
                          │     ├─ logging + fatal-error notifier ───────────────┐
                          │     └─ same-origin middleware + shutdown             │
                          └─ Vue scaffold (router, layout, format.ts, api client)│
                                └─ go:embed + SPA fallback                       │
                                                                                 │
storage.Store (load/validate/migrate) ── save (atomic, hash conflict)            │
      └─ startup resolution + app config ── refuse-to-start ─────────────────────┘
            ├─ /api/data/info + Données UI
            ├─ export / import (+ backups)
            ├─ reload (409 banner)
            └─ open / new file (Picker → zenity)

loan maths (pure) ── dating + yearly grouping
      └─ validator + /simulate ── Prêt page (live)
            └─ loans section + CRUD ── Prêts list + save
                  ├─ duplicate / delete + DeleteGuard hook
                  ├─ amortization table
                  └─ LoanChart.vue

legal-rate check ── settings section + Paramètres UI
      └─ profitability maths (pure) ── loan-aware maths (réel, financement)
            └─ validator + /evaluate ── Bien page (live)
                  └─ properties section + CRUD ── Biens list + save + ceilings
                        └─ loan link (guard, filter, Financement card)
                              └─ "Créer un prêt pour ce bien" + "Biens liés"
```

## Architecture Decisions

- **Docker + Makefile come first (task 1).** Every later verification goes through `make`, so the riskiest piece of tooling is proven before any feature work.
- **Testable seams everywhere:** the listener, health probe, exec runner, notifier, clock, file writer/renamer and file `Picker` are all injected interfaces or functions. Unit tests never open a real browser, dialog or message box.
- **One shared HTTP toolkit in `internal/server`:** the JSON response/error helpers (`{"error":{code,message,fields}}`) and the `requireSameOrigin` middleware are built in app-shell and reused by every module.
- **One shared frontend toolkit:** `web/src/api/client.ts` (typed fetch, `ApiError` carrying `code`/`fields`), `format.ts`, `MoneyInput`/`RateInput` (French parsing), `ResultCard`, a `useUnsavedGuard` composable and a `useDebouncedEval` composable. Built the first time they're needed (in loan) and reused by property.
- **storage stays generic:** modules call `RegisterSection(key, validate)` at startup, and `Get[T]`/`Update[T]`. storage never imports loan or property. loan never imports property: the link goes through `loan.Service.RegisterDeleteGuard`.
- **Line numbers in validation errors (decision to confirm, see Open Questions):** section validators return a `storage.FieldError{Path: "loans[2].principal_cents", Message: "…"}`. storage maps the path to a line using the parsed `yaml.Node` tree. That keeps validators simple and still lets the startup error name the line.
- **First save with no file on disk:** the "external edit" hash check treats "absent at load, still absent" as unchanged. A file created externally in the meantime counts as a conflict.
- **Calculations only in Go.** The frontend never computes money: live previews go through `/simulate` and `/evaluate`.

## Phases (index — details in `todo.md`)

### Phase 1: app-shell
- Task 1: Docker + Makefile + Go health endpoint
- Task 2: Vue scaffold with French layout, routes, placeholders
- Task 3: `format.ts` (fr-FR euros and percentages)
- Task 4: Embed the frontend + SPA fallback (`make run`)
- Task 5: Flags, `--version`, version injection, cross-compile (`make build`)
- Task 6: Port selection, single-instance detection, browser launch
- Task 7: Same-origin middleware, shutdown + Quitter button
- Task 8: Log file + fatal-error notifier (Windows message box, notify-send)
- Task 9: `make dev`, `make debug`, `make clean`, smoke test
- **Checkpoint A:** app-shell acceptance criteria 1–11, Windows check by hand

### Phase 2: storage
- Task 10: Store load + validate + migrations
- Task 11: Atomic save + external-edit detection
- Task 12: Startup resolution, app config, refuse-to-start, `/api/data/info` + Données page
- Task 13: Export + import (with backup)
- Task 14: Reload after an external edit (409 banner + Recharger)
- Task 15: Open another file (Picker / zenity + typed-path fallback)
- Task 16: New file
- **Checkpoint B:** storage acceptance criteria 1–12

### Phase 3: loan
- Task 17: Loan maths: payment, schedule, insurance, fees, totals (reference values)
- Task 18: Month dating + calendar-year grouping + property-style checks
- Task 19: Loan validator + `POST /api/loans/simulate`
- Task 20: Prêt page with live simulation (shared inputs and result cards)
- Task 21: `loans` section + CRUD service + endpoints
- Task 22: Prêts list + save/load on the Prêt page + unsaved-changes guard
- Task 23: Duplicate + delete with the delete-guard hook
- Task 24: Amortization table (Annuel / Mensuel)
- Task 25: `LoanChart.vue` (hand-written SVG)
- **Checkpoint C:** loan acceptance criteria 1–8

### Phase 4: property
- Task 26: Check legal rates against official sources (research, no code)
- Task 27: `settings` section + `/api/settings` + *Hypothèses et fiscalité* form
- Task 28: Profitability maths (reference tables × 3 regimes) + DPE warnings
- Task 29: Loan-aware maths (réel foncier deduction, Financement figures)
- Task 30: Property validator + `POST /api/properties/evaluate`
- Task 31: Bien page with live evaluation
- Task 32: `properties` section + CRUD + Biens list + micro-ceiling warnings
- Task 33: Loan link: `loan_id`, delete guard, `?loanId=`, Financement card
- Task 34: « Créer un prêt pour ce bien » round trip + « Biens liés »
- **Checkpoint D (v1 done):** property acceptance criteria 1–9 + project success criteria

## Parallelization

Mostly sequential: there is one developer and the shared toolkits are built along the way. Safe parallel lanes if wanted:
- Phase 1: tasks 5, 6 and 8 are independent once task 4 is done.
- Phase 2: tasks 13, 14, 15 and 16 are independent once task 12 is done (16 reuses the Picker from 15).
- Phase 3: tasks 24 and 25 are independent once task 22 is done.
- Phase 4: task 26 (research) can run in parallel with phase 3. Task 28 only needs task 27.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Docker/Make cross-compile or windowsgui build issues | High | Task 1 proves the toolchain. Task 5 builds both binaries early. Windows double-click is checked by hand at Checkpoint A, not at the end |
| Windows-only code (`MessageBoxW`, `rundll32`) can't be run in CI | Med | Build tags plus an injected notifier/runner. Compile it in `make build` and `make lint` (`GOOS=windows go vet`). Manual check on Windows at every checkpoint |
| Spec reference values (loan, property) are off by a cent | Med | Task 17/28 compute them under TDD first. Any mismatch is raised to the human; the spec is fixed before the test is frozen (spec boundary: "update the spec before changing a decision") |
| Legal rates (PS 17.2 %, micro ceilings, DPE dates) are out of date | Med | Task 26 checks them against official sources before any property maths. A change needs approval since it changes a calculation rule |
| Mapping validation errors to YAML line numbers | Med | `FieldError` path → `yaml.Node` lookup (see decisions). Tested in task 10 |
| zenity missing on the user's Linux box | Low | Typed-path fallback (task 15), tested with a fake "unavailable" Picker |
| Hash-check race (edit between hash and rename) | Low | Accepted: single-user app. Hash check plus atomic rename covers the realistic case |

## Resolved Questions (2026-11-13)

1. **Frontend dev dependencies — approved.** `@vitejs/plugin-vue`, `jsdom`, `eslint-plugin-vue`, `@vue/eslint-config-typescript`, `@vue/eslint-config-prettier`, `@vue/tsconfig`, `@types/node` may be added in task 2 (build/test only). Add them to the Tech Stack table in `SPEC.md` in the same task.
2. **Validator line numbers — approved.** Validators return `FieldError{Path, Message}`; storage maps the path to a YAML line. Update the `RegisterSection` sketch in `SPEC-storage.md` in task 10.
3. **Delve in the dev image — approved.**
4. **Loan reference values — approved:** one-time web lookup in task 17 against a public French simulator (public figures only), source recorded in the test.

## Open Questions

None.
