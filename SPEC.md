# Spec: immo-calc

## Objective

**immo-calc** is a local desktop web app for working out the **financial return of buying a flat to rent it out** in France. It is for one household (the author and their family). It is not a public product.

You start it by double-clicking a single executable on Linux or Windows. That starts a local web server and opens the browser on a French interface. Data is stored in a YAML file next to the executable. The whole data set can be exported to one file and imported back.

Calculations approximate French rules (taxes, loan maths) **realistically, not exactly**.

### Main features (v1)

1. **Biens (properties):** enter a flat (Prix, DPE, Loyer, Charges, Taxe foncière, Charges de copropriété) and get *Revenus mensuels bruts*, *Revenus mensuels après impôts*, *Revenu annuel net* and *Rentabilité*.
2. **Prêts (loans):** simulate a loan (Montant, Taux, Durée) and get the *mensualité*, the amortization schedule, the total interest and the total cost of the loan. Simulations are saved.
3. **Link between the two:** a property can reference a saved loan simulation, and you can move from one to the other in one click. Folding the loan into the profitability calculation (cash-flow, return on own contribution) is planned. It is designed for from the start, though its exact rules are defined in `SPEC-property.md`.

## Capability Map (approved)

| Module id | Responsibility | Depends on | Spec |
|---|---|---|---|
| `app-shell` | Go binary embedding the Vue frontend, local HTTP server, browser launch, Linux/Windows packaging, Makefile + Docker tooling, French UI skeleton and navigation | — | `SPEC-app-shell.md` |
| `storage` | YAML persistence (default `immo-calc.yaml` next to the binary, `--data` flag, open/switch to another file from the UI), atomic writes, schema versioning, full export/import | `app-shell` | `SPEC-storage.md` |
| `loan` | Loan maths (pure Go), CRUD for saved simulations, API, Vue pages | `app-shell`, `storage` | `SPEC-loan.md` |
| `property` | Property CRUD, profitability maths (pure Go), global settings (TMI, tax regime), link to a loan, API, Vue pages | `storage`, `loan` | `SPEC-property.md` |

**Build order:** `app-shell` → `storage` → `loan` → `property`

Each module gets its own spec, reviewed and approved before it is planned. This file holds the **shared conventions** that apply to every module.

## Tech Stack

| Layer | Choice | Notes |
|---|---|---|
| Backend | **Go** (latest stable, pinned in `go.mod` and Docker images at bootstrap) | Standard library `net/http` (method + path routing patterns); no web framework |
| YAML | `go.yaml.in/yaml/v3` | Maintained fork of `gopkg.in/yaml.v3` |
| Native file dialogs | `github.com/ncruces/zenity` | Pure Go: Windows native, `zenity`/`kdialog` on Linux. Used by `storage` |
| Frontend | **Vue 3** (Composition API, `<script setup>`), **TypeScript**, **Vite**, **Vue Router** | Built to static files and embedded in the binary with `go:embed` |
| Frontend tests | **Vitest** + **@vue/test-utils** | |
| Lint/format | `gofmt`, `go vet`; ESLint + Prettier + `vue-tsc` | |
| Tooling | **Docker** + **Makefile** | Every dev operation goes through `make`, which runs inside containers. The only host requirements are Docker and make |
| Targets | `linux/amd64`, `windows/amd64` (no console window: `-H windowsgui`) | Cross-compiled from the Linux container. amd64 only |

Runtime constraints:
- No network access at runtime except `127.0.0.1`. No CDN: all assets are embedded.
- No external runtime dependencies. The user only needs the executable and a browser.

**Where the logic lives:** all financial calculations are done **in Go** (`internal/loan`, `internal/property`) and exposed over a JSON API. The frontend only collects inputs and formats outputs. That gives one tested source of truth for the rules.

## Commands

All of these run through Docker. Only the final binaries run natively.

```
make dev        # Vite dev server (HMR) + Go server, Vite proxies /api → Go  → http://localhost:5173
make run        # Build frontend + backend and run the app in a container   → http://localhost:8421
make debug      # Run the Go server under Delve (headless, port 2345) with the built frontend
make test       # go test ./... -race -cover  +  vitest run (frontend)
make test-go    # Backend tests only
make test-web   # Frontend tests only
make lint       # gofmt -l, go vet, eslint, vue-tsc --noEmit
make build      # Cross-compile → dist/immo-calc-linux-amd64, dist/immo-calc-windows-amd64.exe
make clean      # Remove dist/, web/dist/, caches
```

## Project Structure

```
cmd/immo-calc/main.go      → Entry point: flags, wiring, start server, open browser
internal/server/           → HTTP routing, JSON helpers, SPA/static handler
internal/browser/          → Open the default browser (xdg-open / rundll32), no dependency
internal/storage/          → YAML data store, export/import
internal/loan/             → Loan maths + loan service/handlers
internal/property/         → Profitability maths + property service/handlers
web/                       → Vue app (package.json, vite.config.ts, src/)
web/src/views/             → Pages (BiensListe, BienDetail, PretsListe, PretDetail, Parametres…)
web/src/components/        → Reusable components (MoneyInput, ResultCard…)
web/src/api/               → Typed fetch client for the Go API
web/src/format.ts          → fr-FR formatting (euros, percentages, dates)
web/embed.go               → //go:embed dist  (package web)
docker/                    → Dockerfiles (build/test/dev images)
Makefile
SPEC.md, SPEC-<module>.md  → Specs (this map is the index)
tasks/                     → plan.md, todo.md per planning phase
```

Go tests sit next to their code (`*_test.go`). Frontend tests sit next to their components (`*.spec.ts`).

## Code Style

**Language rule:** code, identifiers, comments, commit messages and API field names are in **English**. Everything the user sees (labels, messages, errors shown in the UI) is in **French**.

**Money and rates:**
- Amounts are stored and exchanged as **integer euro cents** (`int64` in Go, `number` in TS).
- Rates are `float64` **percentages** (`3.5` means 3.5 %).
- Intermediate maths may use `float64`. Results are rounded to the cent (half away from zero) at the boundary.
- Durations are in **months** internally. The UI may show years.

Go: small pure functions, table-driven tests, errors wrapped with context, no global state.

```go
// MonthlyPayment returns the constant monthly installment (in cents) for a
// fully amortizing loan. annualRatePct is a percentage, e.g. 3.5 for 3.5 %.
func MonthlyPayment(principalCents int64, annualRatePct float64, months int) (int64, error) {
	if principalCents <= 0 || months <= 0 || annualRatePct < 0 {
		return 0, fmt.Errorf("monthly payment: invalid input (principal=%d, rate=%v, months=%d)",
			principalCents, annualRatePct, months)
	}
	if annualRatePct == 0 {
		return roundCents(float64(principalCents) / float64(months)), nil
	}
	r := annualRatePct / 100 / 12
	p := float64(principalCents) * r / (1 - math.Pow(1+r, -float64(months)))
	return roundCents(p), nil
}
```

Vue/TS: `<script setup lang="ts">`, typed props, French text in templates, all formatting through `format.ts`:

```vue
<script setup lang="ts">
import { formatEuros } from '@/format'
defineProps<{ label: string; cents: number }>()
</script>

<template>
  <div class="result-card">
    <span class="result-card__label">{{ label }}</span>
    <strong class="result-card__value">{{ formatEuros(cents) }}</strong>
  </div>
</template>
```

Naming: Go packages are short, lowercase and singular (`loan`, `property`). Vue components use PascalCase. JSON fields use camelCase (`monthlyPayment`). YAML keys use snake_case (`monthly_rent`).

## Testing Strategy

- **Test-driven, always:** write a failing test before each piece of behavior (red → green → refactor).
- **Calculation packages** (`internal/loan`, `internal/property`): table-driven unit tests with **reference values checked against public French simulators** and recorded in the test. Target **≥ 90 % statement coverage**.
- **Storage:** round-trip tests in temporary directories (write → read → equal), plus corrupt/old-version files and import validation.
- **HTTP API:** `httptest` integration tests per endpoint (status codes, JSON shape, validation errors).
- **Frontend:** Vitest + Vue Test Utils for components with logic (forms, validation, formatting). `format.ts` is fully tested.
- **Smoke test:** the built binary starts with `--no-browser`, then `GET /api/health` returns 200 and `GET /` serves the Vue index.
- `make test` must pass before any commit.

## Boundaries

**Always**
- Write the failing test first.
- English in code, French in the UI.
- Go through `make` / Docker for every build, test and run operation.
- Bind the server to `127.0.0.1` only.
- Keep the YAML backward-compatible via a `schema_version` field, with migrations when it changes.
- Update the relevant spec before implementing a changed decision.

**Ask first**
- Adding any dependency (Go module or npm package) not listed in this spec.
- Changing the YAML schema or a calculation rule once it's defined.
- Changing the module map or the build order.
- Committing or pushing (done only on request).

**Never**
- Send any data off the machine (no telemetry, no CDN, no external API).
- Overwrite or delete the user's YAML file without a backup.
- Skip, weaken or delete a failing test to get to green.
- Put French identifiers in code or English text in the UI.

## Success Criteria (whole project, v1)

- Double-clicking the executable on Linux and on Windows opens the French UI in the default browser.
- A flat can be created, edited, deleted and listed, and its four outputs are shown.
- A loan can be simulated and saved, and its mensualité, amortization schedule, total interest and total cost are shown.
- A property can be linked to a saved loan, with navigation both ways.
- Data survives a restart (YAML next to the binary). Export then import on another machine gives back the same data.
- `make test` and `make lint` pass, and calculation packages are at ≥ 90 % coverage.

Detailed criteria live in each module spec.

## Open Questions (project-level)

- Loan rules: resolved in `SPEC-loan.md`.
- Property rules: resolved in `SPEC-property.md` (legal rates to be checked against official sources before implementation).
