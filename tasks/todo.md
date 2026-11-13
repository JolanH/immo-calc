# immo-calc v1 — Task List

Plan and rationale: `tasks/plan.md`. Specs: `SPEC.md`, `SPEC-<module>.md`.

**Definition of Done (every task):** test written first (red → green → refactor) · `make test` green · `make lint` green · English code, French UI · no new dependency without approval · spec updated first if a decision changed · no commit unless asked.

---

## Phase 1 — app-shell (`SPEC-app-shell.md`)

### [x] Task 1: Docker + Makefile + Go health endpoint
**Description:** Bootstrap the toolchain and the smallest working server. Pin the latest stable Go in `go.mod` and the Docker images. `internal/server` gets a router (stdlib `net/http` patterns), JSON response/error helpers (shared error format with optional `fields`), `GET /api/health` and the `/api/*` 404 JSON. `cmd/immo-calc/main.go` starts it on `127.0.0.1:8421`.
**Acceptance criteria:**
- [ ] `GET /api/health` → `200 {"status":"ok","app":"immo-calc","version":"dev"}`
- [ ] `GET /api/unknown` → `404 {"error":{"code":"not_found","message":"Ressource introuvable"}}`
- [ ] `make test-go` and `make lint` (Go part: `gofmt -l`, `go vet`) run in Docker with only Docker + make on the host
**Verification:** `make test-go` (httptest for health, 404, error helper with and without `fields`) · `make lint`
**Dependencies:** None
**Files:** `go.mod`, `cmd/immo-calc/main.go`, `internal/server/{server,json,server_test}.go`, `docker/Dockerfile.*`, `Makefile`, `.gitignore`
**Scope:** M

### [x] Task 2: Vue scaffold with French layout, routes and placeholders
**Description:** Create `web/` (Vue 3 + TS + Vite + Vue Router, history mode). Header with **immo-calc** and a (not yet wired) **Quitter** button. Nav **Biens / Prêts / Paramètres**. Placeholder pages with title + « Fonctionnalité à venir ». `/` → `/biens`. Unknown route → « Page introuvable » + link to Biens. Plain CSS. Add `make test-web` and the web half of `make lint` (eslint, prettier check, `vue-tsc --noEmit`).
**Acceptance criteria:**
- [ ] `/` redirects to `/biens`. `/prets` and `/parametres` show their placeholder
- [ ] Unknown route shows « Page introuvable » with a working link to `/biens`
- [ ] `make test-web` and `make lint` pass
**Verification:** `make test-web` (router tests) · `make lint`
**Dependencies:** Task 1 · dev deps approved (plan.md, resolved Q1): also list them in `SPEC.md` Tech Stack
**Files:** `web/package.json`, `web/vite.config.ts`, `web/tsconfig*.json`, `web/src/{main.ts,App.vue,router.ts}`, `web/src/views/*.vue`, `web/src/router.spec.ts`, `docker/Dockerfile.node`, `Makefile`
**Scope:** M

### [x] Task 3: `format.ts`
**Description:** `formatEuros(cents)` → `"1 234,56 €"`, `formatPercent(value)` → `"3,50 %"`, using `Intl` `fr-FR`. Fully tested. (Tests compare after normalizing narrow no-break spaces.)
**Acceptance criteria:**
- [ ] Covers zero, negatives, large amounts, rounding, percent with 2 decimals
- [x] ~~100 % coverage of `format.ts`~~ — dropped (2026-11-13): frontend coverage numbers not needed; every branch is tested
**Verification:** `make test-web`
**Dependencies:** Task 2
**Files:** `web/src/format.ts`, `web/src/format.spec.ts`
**Scope:** XS

### [x] Task 4: Embed the frontend + SPA fallback
**Description:** `web/embed.go` (`//go:embed dist`). The server takes an `fs.FS`: serves `/assets/*` and other static files, and returns `index.html` for any other `GET` that isn't `/api/*`. `make run` builds both halves and runs the app in a container on port 8421.
**Acceptance criteria:**
- [ ] `GET /` and `GET /prets/123` serve `index.html`; `GET /assets/x.js` serves the asset with the right content type
- [ ] `/api/*` unknown still returns the 404 JSON (not `index.html`)
- [ ] `make run` → `http://localhost:8421/biens` shows the French layout, and reloading a deep link works
**Verification:** `make test-go` (httptest with a `fstest.MapFS`) · `make run` + browser check
**Dependencies:** Tasks 1, 2
**Files:** `web/embed.go`, `internal/server/static.go`, `internal/server/static_test.go`, `cmd/immo-calc/main.go`, `Makefile`
**Scope:** S

### [x] Task 5: Flags, `--version`, version injection, cross-compile
**Description:** Config parsing for `--port`, `--data` (default `<exe dir>/immo-calc.yaml`), `--no-browser`, `--debug`, `--version`, as a pure function over args + exe dir. `make build` cross-compiles `dist/immo-calc-linux-amd64` and `dist/immo-calc-windows-amd64.exe` with `CGO_ENABLED=0 -trimpath -ldflags "-s -w -X main.version=…"` (+ `-H windowsgui` for Windows). The health endpoint reports the injected version.
**Acceptance criteria:**
- [ ] Table-driven flag tests (defaults, each flag, invalid port)
- [ ] `make build` produces both binaries from a clean checkout; `dist/immo-calc-linux-amd64 --version` prints the git-describe version
- [ ] `/api/health` returns that version
**Verification:** `make test-go` · `make build` · run `--version`
**Dependencies:** Task 4
**Files:** `cmd/immo-calc/{main,config,config_test}.go`, `Makefile`
**Scope:** S

### [x] Task 6: Port selection, single instance, browser launch
**Description:** Bind `127.0.0.1:<port>`. If it's taken, probe `/api/health`: if it's immo-calc, open the browser on it and exit 0; otherwise bind `127.0.0.1:0`. `internal/browser` picks `xdg-open` / `rundll32 url.dll,FileProtocolHandler` per OS through an injected runner. Skip the browser with `--no-browser`.
**Acceptance criteria:**
- [ ] Unit tests with injected listen/probe functions cover: free port, taken by immo-calc (exit 0, no server), taken by something else (random port, URL logged)
- [ ] Browser command chosen correctly for linux and windows (injected runner, no real launch)
- [ ] The listener never binds anything other than `127.0.0.1` (asserted in test)
**Verification:** `make test-go` · by hand: run two instances of the Linux binary; occupy 8421 with `nc -l` and start the app
**Dependencies:** Task 5
**Files:** `cmd/immo-calc/listen.go`, `cmd/immo-calc/listen_test.go`, `internal/browser/{browser,browser_test}.go`
**Scope:** M

### [x] Task 7: Same-origin middleware, shutdown + Quitter
**Description:** `server.RequireSameOrigin` middleware (reused by every state-changing route later). `POST /api/shutdown` → `200 {}` then graceful shutdown through an injected callback. SIGINT/SIGTERM also shut down gracefully. Frontend: `web/src/api/client.ts` (typed fetch, `ApiError{status, code, message, fields}`), Quitter calls the endpoint and replaces the page with « immo-calc est arrêté, vous pouvez fermer cet onglet ».
**Acceptance criteria:**
- [ ] Shutdown with a foreign `Origin` → 403 and the callback isn't called; same origin → 200 and the callback is called
- [ ] Quitter shows the stopped message after a mocked successful call
- [ ] The process exits cleanly on Quitter and on Ctrl+C
**Verification:** `make test` · by hand with `make run`: click Quitter
**Dependencies:** Task 4
**Files:** `internal/server/{origin,shutdown}.go` + tests, `cmd/immo-calc/main.go`, `web/src/api/client.ts` + spec, `web/src/App.vue`, `web/src/App.spec.ts`
**Scope:** M

### [x] Task 8: Log file + fatal-error notifier
**Description:** Logs to `immo-calc.log` in the exe dir (truncated at start, verbose with `--debug`), falling back to the temp dir when not writable; also stdout. Startup line logs version, URL and data path. Fatal startup errors go through a `Notifier` interface: Windows `user32.MessageBoxW` via `syscall` (build tag), Linux `notify-send` if present. French message names the log file.
**Acceptance criteria:**
- [ ] Log path falls back to the temp dir when the exe dir isn't writable (test with a read-only temp dir)
- [ ] A fatal error calls the notifier with a French message containing the log path (fake notifier)
- [ ] The Windows code compiles (`GOOS=windows go vet ./...` in `make lint`)
**Verification:** `make test-go` · `make lint` · by hand on Windows at Checkpoint A
**Dependencies:** Task 5
**Files:** `internal/applog/{log,log_test}.go`, `internal/notify/{notify,notify_windows,notify_linux,notify_test}.go`, `cmd/immo-calc/main.go`
**Scope:** M

### [x] Task 9: `make dev`, `make debug`, `make clean`, smoke test
**Description:** `make dev`: Vite dev server on 5173 with HMR, proxying `/api` to the Go server (`--no-browser`). `make debug`: Delve headless on 2345 with the built frontend. `make clean`. Smoke test as part of `make test`: build the binary, start it with `--no-browser --port 0`, read the URL, check `/api/health` and `/`, then shut it down via the API.
**Acceptance criteria:**
- [ ] `make dev` serves the app at `http://localhost:5173`, an edit to a `.vue` file hot-reloads, and `/api/health` goes through the proxy
- [ ] `make debug` accepts a Delve connection on 2345
- [ ] `make test` includes the smoke test and it passes; `make clean` removes `dist/`, `web/dist/` and caches
**Verification:** run each target
**Dependencies:** Tasks 6, 7, 8
**Files:** `Makefile`, `web/vite.config.ts`, `docker/*`, `scripts/smoke.sh` (or `cmd/immo-calc/smoke_test.go`)
**Scope:** S

### [x] Task 9a (added 2026-11-13): replace a different running version
**Description:** Requested after Checkpoint B: a forgotten instance of an older build kept answering, so a new binary just opened the browser on it. Now, when the preferred port answers as immo-calc with a different version, the new binary stops it silently with `POST /api/shutdown`, waits up to 5 s for the port, and starts in its place (random port if it doesn't stop). Same version: unchanged (open the browser on it, exit 0). Spec updated in `SPEC-app-shell.md`.
**Acceptance criteria:**
- [x] Different version running → it's shut down and the new one takes over the port (unit tests + two real binaries)
- [x] Same version running → browser opened on it, exit 0, no shutdown
- [x] Old version that won't stop → random free port
**Scope:** S

### [x] Checkpoint A — app-shell done
- [x] `make test`, `make lint`, `make build` pass from a clean checkout
- [x] app-shell acceptance criteria 1–11 verified (2 and 11 by hand on a Windows machine: no console window, French message box on fatal error)
  - Verified on Linux: 1, 3–10 by Claude; 2 by the user (binary opens the web UI).
  - **Windows check deferred (2026-11-13), assumed OK:** criterion 2 (double-click opens the browser on the home page) and 11 (no console window, French message box naming `immo-calc.log` on a fatal error). To be tested by the user on a Windows machine later.
- [x] Human review before Phase 2

---

## Phase 2 — storage (`SPEC-storage.md`)

### [x] Task 10: Store load + validate + migrations
**Description:** `internal/storage.Store`: parse YAML (`go.yaml.in/yaml/v3`) into a document of raw sections; missing file → empty v1 document; missing section → empty; unknown top-level key → error; `schema_version` newer than supported → error; migration registry (tested with a fake v0 → v1). `RegisterSection(key, validate)`; validators return `FieldError{Path, Message}`, mapped to a YAML line. `Get[T]` reads a typed section. Update the `RegisterSection` sketch in `SPEC-storage.md` to the `FieldError` form (approved).
**Acceptance criteria:**
- [ ] Round trip of a document with test sections; empty/missing sections give zero values
- [ ] Unknown key, too-new version, malformed YAML and a failing validator each give an error naming the file and the line
- [ ] Fake v0 → v1 migration is applied in memory
**Verification:** `make test-go` (all in `t.TempDir()`)
**Dependencies:** Checkpoint A
**Files:** `internal/storage/{store,document,migrate,errors}.go` + tests, `SPEC-storage.md`
**Scope:** M

### [x] Task 11: Atomic save + external-edit detection
**Description:** `Update[T](s, key, fn)` under a mutex: apply `fn`, validate, then write atomically (temp file in the same dir, fsync, rename) through an injectable filesystem. Record the SHA-256 on load and after each save; before saving, re-hash and return `ErrFileChangedExternally` if it differs. First save creates the file.
**Acceptance criteria:**
- [ ] A failing write or rename leaves the original file byte-identical
- [ ] Changing the file between load and save → `ErrFileChangedExternally`, nothing written
- [ ] `fn` returning an error → nothing saved; the in-memory state is unchanged; no backup is ever created by a save
**Verification:** `make test-go -race`
**Dependencies:** Task 10
**Files:** `internal/storage/{save,fs}.go` + tests
**Scope:** S

### [x] Task 12: Startup resolution, app config, refuse-to-start, data info + Données page
**Description:** Resolve the data file (`--data` > existing `last_data_file` > default), read/write `immo-calc.config.yaml` (exe dir, fallback `os.UserConfigDir()/immo-calc/`). A remembered file that has gone missing → fall back and set a warning. Load failure at startup → app-shell fatal path with the French message. `GET /api/data/info`. Map `ErrFileChangedExternally` → `409 file_changed_externally` in a shared helper. Paramètres page gets a *Données* section: path, counts, dismissible warning banner.
**Acceptance criteria:**
- [ ] Table-driven resolution tests (4 cases); `--data` never writes `last_data_file`
- [ ] An invalid file at startup → process exits non-zero after notifying a French error with file + line
- [ ] `/parametres` shows the current path and counts, and the « introuvable » banner when relevant
**Verification:** `make test` · by hand: start with a broken YAML, with a missing remembered file
**Dependencies:** Task 11
**Files:** `internal/storage/{resolve,config,handlers}.go` + tests, `cmd/immo-calc/main.go`, `web/src/views/Parametres.vue`, `web/src/components/DataSection.vue` + spec, `web/src/api/data.ts`
**Scope:** M

### [x] Task 13: Export + import (with backup)
**Description:** `GET /api/data/export` (YAML attachment `immo-calc-export-YYYY-MM-DD.yaml`). `POST /api/data/import`: full parse + migrate + validate; on success back up the current file to `<name>.bak-YYYYMMDD-HHMMSS.yaml` (injected clock; none if the file doesn't exist), replace and save. UI: **Exporter**, **Importer…** with the French confirmation (N biens, M prêts) and « Import terminé ».
**Acceptance criteria:**
- [ ] Export then import into another store gives an equal document
- [ ] Invalid import → `400` French message, file and memory unchanged, no backup
- [ ] Valid import creates exactly one backup with the right name; foreign `Origin` → 403
**Verification:** `make test` · by hand: export, edit, import
**Dependencies:** Task 12
**Files:** `internal/storage/{backup,exchange}.go` + tests, `internal/storage/handlers.go`, `web/src/components/DataSection.vue` + spec
**Scope:** M

### [x] Task 14: Reload after an external edit
**Description:** `POST /api/data/reload`: reload from disk with normal validation; invalid → `400`, memory kept. Frontend: the api client turns any `409 file_changed_externally` into a global banner « Le fichier a été modifié en dehors d'immo-calc » with **Recharger**.
**Acceptance criteria:**
- [ ] Reload of a valid edited file → `200` and the new content is served; invalid → `400`, old state kept
- [ ] A mocked 409 on any request shows the banner; Recharger calls reload and hides it on success
**Verification:** `make test` · by hand (once a writable section exists, re-checked at Checkpoint C)
**Dependencies:** Task 12
**Files:** `internal/storage/handlers.go` + test, `web/src/api/client.ts`, `web/src/components/ExternalChangeBanner.vue` + spec, `web/src/App.vue`
**Scope:** S

### [x] Task 15: Open another file
**Description:** `Picker` interface (open/save dialogs) with a zenity implementation (approved dep `github.com/ncruces/zenity`) and an "unavailable" error. `POST /api/data/open`: no body → dialog; `{"path"}` → typed path. Cancel → `204`. Invalid → `400`, stay on current file. Valid → backup current, switch, write `last_data_file`. UI: **Ouvrir un autre fichier…**, falling back to a path text field when the dialog is unavailable.
**Acceptance criteria:**
- [ ] Fake Picker covers cancel / path / error / unavailable
- [ ] Switching backs up the previous file and remembers the new one; after a restart the app opens it
- [ ] An invalid chosen file is rejected and nothing changes
**Verification:** `make test` · by hand on Linux with zenity
**Dependencies:** Task 12
**Files:** `internal/storage/{picker,picker_zenity,switch}.go` + tests, `internal/storage/handlers.go`, `go.mod`, `web/src/components/DataSection.vue` + spec
**Scope:** M

### [x] Task 16: New file
**Description:** `POST /api/data/new`: save dialog (default `immo-calc-nouveau.yaml`) or `{"path"}`. Cancel → `204`. Existing path → `400` « Ce fichier existe déjà… ». Otherwise back up current, write an empty v1 document, switch, remember. UI: **Nouveau fichier…** with the same fallback.
**Acceptance criteria:**
- [ ] Existing path is refused and the existing file is byte-identical afterwards
- [ ] New path → empty file on disk, active, remembered, previous file backed up
**Verification:** `make test`
**Dependencies:** Task 15
**Files:** `internal/storage/switch.go` + test, `internal/storage/handlers.go`, `web/src/components/DataSection.vue` + spec
**Scope:** S

### [x] Checkpoint B — storage done
- [x] `make test`, `make lint`, `make build` pass
- [x] storage acceptance criteria 1–12 verified (3: kill during save by hand or via the injected-failure test; 5 re-checked in Phase 3)
  - Verified by Claude: 2, 4, 6, 7, 8, 10, 12 with the real binary (typed paths); 1, 3, 9, 11 by unit tests (3 via injected write/rename failures). 1 end to end and 5 come back in Phase 3, once a module writes data.
  - **To check by hand:** the native dialogs for « Ouvrir un autre fichier… » and « Nouveau fichier… » (Linux with zenity/kdialog, and Windows). Not exercised automatically, to avoid popping dialogs on the desktop.
- [x] Human review before Phase 3

---

## Phase 3 — loan (`SPEC-loan.md`)

### [x] Task 17: Loan maths: payment, schedule, insurance, fees, totals
**Description:** Pure `internal/loan`: `MonthlyPayment`, schedule with interest on remaining principal and last-installment adjustment, constant insurance on the initial principal, fees, all totals. Cent rounding half away from zero. Cross-check the reference table against one public French simulator first (approved: one-time web lookup, public figures only), and record the simulator, URL and date in the test. A difference beyond ±1 € on totals → stop and ask before changing the spec.
**Acceptance criteria:**
- [ ] The 4 reference rows (payment, total interest, last installment) and the insurance example match exactly
- [ ] Edge cases: 0 %, 1 month, 360 months, half-cent rounding, invalid inputs → error
- [ ] Coverage ≥ 90 %
**Verification:** `make test-go` with `-cover`
**Dependencies:** Checkpoint B
**Files:** `internal/loan/{calc,round}.go`, `internal/loan/calc_test.go`
**Scope:** S

### [x] Task 18: Month dating + calendar-year grouping + property-style checks
**Description:** Date installment k at `first_payment_month + (k−1)`. Yearly view grouped by calendar year with installment counts for partial years. Generated-input checks: remaining reaches 0, Σ principal = amount, Σ interest = total, payment monotonic in rate and amount.
**Acceptance criteria:**
- [ ] Partial first/last years show the right counts (e.g. start 2026-11 → « 2026 (2 échéances) »)
- [ ] Each yearly row equals the sum of its months
- [ ] Property-style checks pass over many generated inputs
**Verification:** `make test-go`
**Dependencies:** Task 17
**Files:** `internal/loan/{schedule,yearly}.go` + tests
**Scope:** S

### [x] Task 19: Loan validator + `POST /api/loans/simulate`
**Description:** `loan.Input` with the spec's rules and French messages (field-level). `POST /api/loans/simulate` returns the result JSON (camelCase, cents) with the monthly and yearly schedules, not saved.
**Acceptance criteria:**
- [ ] Each validation rule has a test with its French message
- [ ] Valid input → `200` with the documented JSON shape; invalid → `400` with `fields`
**Verification:** `make test-go`
**Dependencies:** Task 18
**Files:** `internal/loan/{input,validate,handlers}.go` + tests, `cmd/immo-calc/main.go`
**Scope:** S

### [x] Task 20: Prêt page with live simulation
**Description:** `/prets/nouveau` form (Nom, Montant, Taux, Durée in years, Première échéance month picker, collapsible « Assurance et frais »). Shared components: `MoneyInput`, `RateInput` (French parsing `200 000`, `3,5`), `ResultCard`, `useDebouncedEval` (~300 ms). Results panel shows the 8 result figures; field errors from `400` appear next to fields.
**Acceptance criteria:**
- [ ] Typing updates the results without saving (mocked API, fake timers for the debounce)
- [ ] French number parsing tested (spaces, comma decimal, invalid text)
- [ ] `400` field messages show next to the right fields
**Verification:** `make test-web` · `make dev` by hand with a reference loan
**Dependencies:** Task 19
**Files:** `web/src/views/PretDetail.vue` + spec, `web/src/components/{MoneyInput,RateInput,ResultCard}.vue` + specs, `web/src/composables/useDebouncedEval.ts`, `web/src/api/loans.ts`, `web/src/router.ts`
**Scope:** M

### [x] Task 21: `loans` section + CRUD service + endpoints
**Description:** Register the `loans` section and validator (rules + unique ids) with storage. `loan.Service`: list, create (id from `crypto/rand`, timestamps), get, update. Endpoints `GET/POST /api/loans`, `GET/PUT /api/loans/{id}` behind the same-origin check; results recomputed on read.
**Acceptance criteria:**
- [ ] Create → `201`; get returns inputs + results + schedule; unknown id → `404`; write with a foreign Origin → `403`
- [ ] Persistence round trip through a temp `storage.Store` (restart = new Store on the same file)
- [ ] A hand-edited invalid loan makes startup fail with a message naming the loan
**Verification:** `make test-go`
**Dependencies:** Task 19
**Files:** `internal/loan/{service,section}.go` + tests, `internal/loan/handlers.go`, `cmd/immo-calc/main.go`
**Scope:** M

### [x] Task 22: Prêts list + save/load + unsaved-changes guard
**Description:** `/prets` table (Nom, Montant, Taux, Durée, Mensualité totale, Coût total), row click, **Nouvelle simulation**, empty state. Prêt page: **Enregistrer** (create → navigate to `/prets/:id`, or update), loads `/prets/:id`. `useUnsavedGuard` composable (router guard + `beforeunload`).
**Acceptance criteria:**
- [ ] Saved loan appears in the list; after restart it's there with identical results
- [ ] Leaving with unsaved changes asks for confirmation (tested with mocked router)
- [ ] Empty state shows « Aucune simulation de prêt. »
**Verification:** `make test-web` · by hand with `make run`, including a restart · re-check storage AC 5 (409 banner) now that writes exist
**Dependencies:** Tasks 20, 21
**Files:** `web/src/views/{PretsListe,PretDetail}.vue` + specs, `web/src/composables/useUnsavedGuard.ts` + spec, `web/src/api/loans.ts`
**Scope:** M

### [x] Task 23: Duplicate + delete with the delete-guard hook
**Description:** `Service.RegisterDeleteGuard(func(id) error)`. `POST /api/loans/{id}/duplicate` → « <nom> (copie) », `201`. `DELETE /api/loans/{id}` → `204` or `409 loan_in_use` with the guard's French message. UI: **Dupliquer**, **Supprimer** with confirmation and the 409 message.
**Acceptance criteria:**
- [ ] Duplicate is independent (editing it doesn't change the original)
- [ ] A refusing guard → `409 loan_in_use`, loan kept; no guard → deleted
- [ ] UI shows the confirmation and the blocking message (mocked API)
**Verification:** `make test`
**Dependencies:** Task 22
**Files:** `internal/loan/{service,handlers}.go` + tests, `web/src/views/PretDetail.vue` + spec
**Scope:** S

### [x] Task 24: Amortization table (Annuel / Mensuel)
**Description:** Below the results: table with an **Annuel / Mensuel** toggle (annual by default). Columns Année/Mois, Échéance, Intérêts, Capital remboursé, Assurance, Capital restant dû; French month labels (« déc. 2026 ») and partial-year labels.
**Acceptance criteria:**
- [ ] Toggle switches views; annual is the default
- [ ] Labels and amounts are formatted through `format.ts` (and a `formatMonth` helper added there with tests)
**Verification:** `make test-web`
**Dependencies:** Task 22
**Files:** `web/src/components/AmortizationTable.vue` + spec, `web/src/format.ts` + spec, `web/src/views/PretDetail.vue`
**Scope:** S

### [x] Task 25: `LoanChart.vue`
**Description:** Hand-written SVG: one stacked bar per calendar year (Capital remboursé + Intérêts), overlaid line for Capital restant dû on a second axis, hover tooltip (year + 3 amounts), French legend, colours readable in light and dark, `<title>`/`aria-label` summary. Updates live with the form.
**Acceptance criteria:**
- [ ] Bar count = number of calendar years; bar heights proportional to the yearly sums
- [ ] Tooltip content correct; empty or invalid input shows no chart
**Verification:** `make test-web` · by hand in light and dark mode
**Dependencies:** Task 22
**Files:** `web/src/components/LoanChart.vue` + spec, `web/src/views/PretDetail.vue`
**Scope:** M

### [x] Task 25a (added 2026-11-13): breadcrumb on detail pages
**Description:** Requested at Checkpoint C. `AppBreadcrumb` (« Prêts › <nom du prêt> », « Prêts › Nouvelle simulation ») above the loan page title, « Prêts » linking back to the list. Reusable for the property pages (« Biens › <nom> »). Specs updated (`SPEC-app-shell.md`, `SPEC-loan.md`).
**Acceptance criteria:**
- [x] Labelled `<nav aria-label="Fil d'Ariane">`, current page marked `aria-current="page"`
- [x] « Prêts » navigates back to the list (component + page tests)
**Scope:** XS

### [x] Task 25b (added 2026-11-13): home page skeleton and active loan
**Description:** Requested at Checkpoint C. `/` becomes the home page « Accueil » (no longer a redirect): left pane (~80 %) for the Biens list (placeholder until Task 32), right pane (~20 %) with a simplified list of saved loans (name, monthly total, link) and a radio selector for one active loan or « Aucun ». Active loan kept in server memory: `GET/PUT /api/active-loan` (404 unknown loan, same-origin on PUT, cleared when the loan is deleted). Breadcrumb starts with « Accueil » on every other page; the header name links home. Specs updated (`SPEC-app-shell.md`, `SPEC-loan.md`, `SPEC-property.md` *Home page*).
**Acceptance criteria:**
- [x] `/` shows the two panes; selecting a loan persists across a page reload, not across a restart
- [x] Unknown loan → 404; deleting the active loan clears it
- [x] Breadcrumbs: « Accueil › Prêts », « Accueil › Prêts › <nom> », « Accueil › Paramètres », « Accueil › Biens »
**Verification:** `make test` · real binary: select, reload, restart
**Dependencies:** Tasks 23, 25a
**Files:** `internal/loan/{active,handlers}.go` + tests, `web/src/views/Accueil.vue` + spec, `web/src/api/loans.ts`, `web/src/router.ts`, `web/src/App.vue`, list/detail views
**Scope:** M

### [x] Checkpoint C — loan done
- [x] `make test`, `make lint`, `make build` pass; `internal/loan` coverage ≥ 90 %
- [x] loan acceptance criteria 1–8 verified; storage AC 5 (external edit → 409 banner → Recharger) verified end to end
  - Verified by Claude: loan 1, 2, 6 (unit + 2 000 generated loans), 3 (create via the real binary, restart, identical results; live results in component tests), 4, 5, 7 (startup refusal names the loan and line), 8 (chart rendered and inspected in light and dark). Storage 5 end to end against the real binary: hand edit kept, save → 409, reload picks it up; the banner itself is covered by component tests.
  - **To try by hand:** typing in the Prêt form in a real browser (live results, unsaved-changes prompt), the chart tooltip on hover and with the arrow keys.
- [x] Human review before Phase 4

---

## Phase 4 — property (`SPEC-property.md`)

### [x] Task 26: Check legal rates against official sources
**Description:** Check against impots.gouv.fr / service-public.fr: prélèvements sociaux rate on property income, micro-foncier ceiling + 30 % allowance, micro-BIC (long-term furnished) ceiling + 50 % allowance, DPE rental-ban dates (G/F/E). Write a short findings note in `tasks/legal-rates.md` with URLs and dates checked. **Any difference from the spec → stop and ask** before changing the spec (it's a calculation rule).
**Acceptance criteria:**
- [ ] Every ⚠️ value in `SPEC-property.md` has a cited official source and a confirmed or corrected value
- [ ] Spec updated (after approval) if anything changed
**Verification:** human review of `tasks/legal-rates.md`
**Dependencies:** None (can run any time before Task 28)
**Files:** `tasks/legal-rates.md`, possibly `SPEC-property.md`
**Scope:** XS

### [x] Task 27: `settings` section + `/api/settings` + Hypothèses et fiscalité
**Description:** `internal/property` settings type with defaults (missing section → defaults), validator (TMI ∈ {0,11,30,41,45}, ranges, vacancy step 0.5). `GET/PUT /api/settings`. Paramètres page gets the *Hypothèses et fiscalité* form next to *Données*.
**Acceptance criteria:**
- [ ] Missing `settings` → defaults; each rule rejected with a French message
- [ ] Saved settings persist across a restart
- [ ] Form shows field errors and saves (mocked API)
**Verification:** `make test`
**Dependencies:** Tasks 12, 26
**Files:** `internal/property/{settings,settings_handlers}.go` + tests, `cmd/immo-calc/main.go`, `web/src/components/SettingsForm.vue` + spec, `web/src/views/Parametres.vue`, `web/src/api/settings.ts`
**Scope:** M

### [x] Task 28: Profitability maths + DPE warnings
**Description:** Pure `Evaluate(input, settings, loanFigures?)` implementing rules 1–10 (acquisition cost, collected rent, owner charges incl. vacancy charges and agency fee, R, taxable income per regime floored at 0, tax, RN, monthly figures, three yields). Constants cite their sources from Task 26. DPE warnings for G/F/E.
**Acceptance criteria:**
- [ ] Both reference tables reproduced exactly for the 3 regimes (no loan)
- [ ] Edge cases: vacancy 0 and 12, agency on/off, taxable floor at 0, every DPE
- [ ] Coverage ≥ 90 %
**Verification:** `make test-go -cover`
**Dependencies:** Task 27
**Files:** `internal/property/{calc,dpe,constants}.go` + tests
**Scope:** M

### [x] Task 29: Loan-aware maths
**Description:** From a loan's results: year-1 interest + insurance (first 12 installments) for `reel_foncier`; Financement figures (Mensualité, cash-flow before/after tax, Apport = A + fees − principal, Rendement sur apport, « n/a » when Apport ≤ 0). Add a loan reference case built from a `SPEC-loan.md` reference loan and record it in `SPEC-property.md`.
**Acceptance criteria:**
- [ ] `reel_foncier` taxable income reduced by year-1 interest + insurance; deficit floored at 0
- [ ] Apport ≤ 0 → return on contribution is n/a
- [ ] Linked-loan reference case passes and is recorded in the spec
**Verification:** `make test-go -cover`
**Dependencies:** Tasks 18, 28
**Files:** `internal/property/{financing,calc}.go` + tests, `SPEC-property.md`
**Scope:** S

### [x] Task 30: Property validator + `POST /api/properties/evaluate`
**Description:** `property.Input` validation (spec ranges, DPE A–G, regime enum, French messages; `loan_id` must exist when set). `evaluate` returns results, détail du calcul figures, warnings and financement (if `loanId`).
**Acceptance criteria:**
- [ ] Each rule tested with its French message; unknown `loanId` → field error
- [ ] Valid → `200` with results + warnings; invalid → `400` with `fields`
**Verification:** `make test-go`
**Dependencies:** Tasks 21, 29
**Files:** `internal/property/{input,validate,handlers}.go` + tests, `cmd/immo-calc/main.go`
**Scope:** S

### [x] Task 31: Bien page with live evaluation
**Description:** `/biens/nouveau`: form (reusing `MoneyInput`/`RateInput`), Prêt select (« Aucun » + saved loans), live cards (Revenus mensuels bruts, après impôts, Revenu annuel net, Rentabilité brute/nette/**après impôts**), collapsible « Détail du calcul », DPE and ceiling warnings above results, Financement card when a loan is selected.
**Acceptance criteria:**
- [ ] Typing updates the four outputs without saving (mocked API, debounced)
- [ ] DPE warning and Financement card show when relevant
- [ ] Field errors appear next to fields
**Verification:** `make test-web` · `make dev` with the spec's reference flat
**Dependencies:** Task 30
**Files:** `web/src/views/BienDetail.vue` + spec, `web/src/components/{DpeBadge,FinancingCard,CalcDetail}.vue` + specs, `web/src/api/properties.ts`, `web/src/router.ts`
**Scope:** M

### [x] Task 32: `properties` section + CRUD + Biens list + ceiling warnings
**Description:** Register `properties` (validator incl. unique ids, `loan_id` exists — dangling id refuses start). Service + endpoints: list (with key results), create, get, update, duplicate, delete. Micro-ceiling warnings computed across all properties on the same regime. `/biens` list (columns per spec, DPE badge, empty state, **Nouveau bien**). Save/load/Dupliquer/Supprimer + unsaved guard on the Bien page.
**Acceptance criteria:**
- [ ] CRUD + duplicate round trip through a temp store; restart gives identical results
- [ ] Changing a setting changes every affected property's results
- [ ] Going over a micro ceiling summed across flats shows the warning
**Verification:** `make test` · by hand with `make run`, including a restart
**Dependencies:** Task 31
**Files:** `internal/property/{service,section,handlers}.go` + tests, `web/src/views/{BiensListe,BienDetail}.vue` + specs
**Scope:** L — split during implementation if needed: (a) backend CRUD + ceilings, (b) list + save UI

### [x] Task 33: Loan link
**Description:** Register the delete guard with `loan` (message names the properties). `GET /api/properties?loanId=`. Property GET returns a linked-loan summary. **Voir le prêt →** link; Prêt column in the list; Cash-flow column when linked.
**Acceptance criteria:**
- [ ] Deleting a loan in use → `409` naming the property; deleting a property leaves its loan intact
- [ ] `?loanId=` filter returns only linked properties
- [ ] Bien page and list link to the loan
**Verification:** `make test`
**Dependencies:** Tasks 23, 32
**Files:** `internal/property/{service,handlers}.go` + tests, `cmd/immo-calc/main.go`, `web/src/views/{BiensListe,BienDetail}.vue` + specs
**Scope:** S

### [x] Task 33b (added 2026-11-13): active loan on the home page Biens list
**Description:** Fill the home page's left pane with the Biens list (reuse Task 32's list) and apply the active loan: while one is selected it **overrides** every bien's own linked loan for the Financement figures on that list (`SPEC-property.md`, *Home page*). **First settle the open question** (amount borrowed: the loan's own amount, or the loan as a template with the bien's acquisition cost) with the user.
**Acceptance criteria:**
- [x] Open question answered and recorded in `SPEC-property.md`
- [x] With an active loan, every bien on the home list shows Financement figures from it; with « Aucun », each uses its own link
- [x] `/biens` and the bien detail page are unaffected by the active loan
**Verification:** `make test` · by hand with two biens and two loans
**Dependencies:** Tasks 25b, 33
**Scope:** M

### [x] Task 34: « Créer un prêt pour ce bien » + « Biens liés »
**Description:** Button → `/prets/nouveau?bien=<id>` with Montant = A and Nom « Prêt – <nom> ». On Enregistrer, create the loan and link it (PUT property), then show « ← Retour au bien ». On the Prêt page, show « Biens liés » from `?loanId=`.
**Acceptance criteria:**
- [ ] Prefill correct; save creates and links the loan; return link goes back to the property
- [ ] Prêt page lists linked properties as links
**Verification:** `make test-web` · by hand round trip with `make run`
**Dependencies:** Task 33
**Files:** `web/src/views/{PretDetail,BienDetail}.vue` + specs, `web/src/components/LinkedProperties.vue` + spec
**Scope:** S

### [ ] Checkpoint D — v1 done
- [x] `make test`, `make lint`, `make build` pass; `internal/loan` (94,8 %) and `internal/property` (92,8 %) coverage ≥ 90 %
- [x] property acceptance criteria 1–9 verified
  - By Claude: 1 (both reference tables × 3 regimes, plus the linked-loan cases), 2 (create via the real binary, restart, identical results; live evaluation in component tests), 3 (settings change propagates, unit), 4 (Financement card; réel deducts year-1 interest and insurance), 5 (« Créer un prêt pour ce bien » prefill checked in the real binary, link and return in component tests), 6 (deleting a used loan → 409 naming the bien, real binary; deleting a bien keeps its loan), 7 (DPE G/F/E and micro ceilings summed across biens), 8 (French field errors, nothing saved), 9 (official sources in `tasks/legal-rates.md`, cited next to the constants).
- [ ] Project success criteria (`SPEC.md`) verified, including on Windows: double-click, data survives restart, export → import on another machine — include the app-shell Windows checks deferred at Checkpoint A (criteria 2 and 11)
  - On Linux by Claude: data survives a restart; export from one instance then import into another gives identical properties, loans, settings and export file.
  - **Left for the user:** the Windows checks (double-click, no console window, French message box; criteria 2 and 11 of app-shell), the native file dialogs (Ouvrir / Nouveau fichier), and a pass through the UI in a real browser (typing in the forms, chart tooltip, home page active loan).
- [ ] Human review
