# Spec: app-shell

> Module of [immo-calc](SPEC.md). Depends on: nothing. Shared conventions (stack, style, testing, boundaries) are in `SPEC.md`.

## Objective

Provide the runnable skeleton that every other module plugs into: a single Go executable with the Vue frontend embedded, which starts a local server and opens the browser on a French interface when double-clicked. Also provide the Docker + Makefile tooling for developing, testing and packaging it.

At the end of this module there is no business feature yet. There is a working, tested, packaged empty app with navigation to placeholder pages.

## Scope

**In:**
- `cmd/immo-calc` entry point and command-line flags
- `internal/server`: router, `/api/health`, JSON error helper, static + SPA fallback handler
- `internal/browser`: open the default browser on Linux and Windows
- `web/`: Vue 3 + Vite + TS + Vue Router project, layout, navigation, placeholder pages, `format.ts`
- Embedding `web/dist` in the binary
- Dockerfiles, Makefile (all targets listed in `SPEC.md`), cross-compilation
- Version string injected at build time

**Out:** data storage (`storage`), any loan or property logic.

## Behavior

### Startup

1. Parse flags:
   | Flag | Default | Meaning |
   |---|---|---|
   | `--port` | `8421` | Preferred port |
   | `--data` | `<dir of executable>/immo-calc.yaml` | Data file path (used by `storage`; parsed and passed along here) |
   | `--no-browser` | `false` | Don't open the browser (tests, dev) |
   | `--debug` | `false` | Verbose logs |
   | `--version` | — | Print the version and exit |
2. Bind `127.0.0.1:<port>`. If the port is taken:
   - If `GET http://127.0.0.1:<port>/api/health` answers as immo-calc **with the same version**, another instance is already running: open the browser on it and exit with code 0.
   - If it answers as immo-calc **with a different version** (typically a forgotten tab from an older build), stop it silently with `POST /api/shutdown` (sending its own origin), wait up to 5 s for the port to be released, then bind it and start normally. No message is shown; the replacement is logged. If the old instance doesn't stop in time, fall back to a random free port (next point).
   - Otherwise, bind a random free port on `127.0.0.1`.
3. Log the URL and the startup details (version, data file path) to the log file (see *Logging*), and to stdout when there is one (Linux terminal, `make` targets).
4. Open the default browser on the URL, unless `--no-browser` is set.

### Shutdown

- The **"Quitter"** button in the UI is the normal way to stop the app. It calls `POST /api/shutdown`. The server answers 200, the UI shows "immo-calc est arrêté, vous pouvez fermer cet onglet", and the process stops gracefully.
- Ctrl+C / SIGTERM (Linux terminal, `make` targets) also stops the server gracefully.
- Because there is no console window, a forgotten tab leaves the app running in the background. That is accepted. Double-clicking the same executable again just reopens the browser on the running instance; launching a different version replaces it (see *Startup*).

### Logging and errors (no console)

- Logs go to `immo-calc.log` in the executable's directory, truncated at each startup. `--debug` makes them verbose. If that directory isn't writable, logs go to the user's temp directory instead.
- **Fatal startup errors** (e.g. no port can be bound) must be visible to a double-click user. On Windows, show a native message box in French via `user32.MessageBoxW`, using the standard library `syscall` package with no extra dependency. On Linux, log the error and try `notify-send` if it's available. The message names the log file.

### HTTP

| Route | Behavior |
|---|---|
| `GET /api/health` | `200 {"status":"ok","app":"immo-calc","version":"<version>"}` |
| `POST /api/shutdown` | `200 {}` then graceful shutdown |
| `/api/*` (unknown) | `404 {"error":{"code":"not_found","message":"Ressource introuvable"}}` |
| `GET /assets/*`, other static files | Served from the embedded `web/dist` |
| any other `GET` | `index.html` (SPA fallback for Vue Router history mode) |

API error format, used by every module: `{"error":{"code":"<machine_code>","message":"<French message for the UI>","fields":{"<field>":"<French message>"}}}`. `fields` is optional and holds validation errors.

`POST /api/shutdown` is protected against cross-site requests: requests whose `Origin` header doesn't match the server's own origin are rejected with 403. All state-changing endpoints in later modules reuse this check (`server.RequireSameOrigin`). When `Origin` is absent, a `Sec-Fetch-Site` of `cross-site` or `same-site` is also rejected; requests with neither header (non-browser clients such as `curl`) are allowed.

**Loopback-only API (DNS-rebinding protection):** every `/api/*` request, read or write, must carry a `Host` header naming this machine: `127.0.0.1` or `localhost`, any port. Anything else gets `403 {"error":{"code":"forbidden_origin","message":"Requête refusée : origine non autorisée"}}`. This stops a web page on a domain that resolves to `127.0.0.1` from reading or changing local data. Static files are served regardless of `Host`.

Calling an existing API path with an unsupported method returns `405 {"error":{"code":"method_not_allowed","message":"Méthode non autorisée"}}` with an `Allow` header.

### UI skeleton (French)

- Header: app name **immo-calc** and the **Quitter** button.
- Navigation: **Biens** (`/biens`), **Prêts** (`/prets`), **Paramètres** (`/parametres`). The section of the current page stays highlighted on its sub-pages (`/prets/123` → Prêts). The app name **immo-calc** in the header links to the home page.
- **Home page « Accueil » (`/`)** (added 2026-11-13): Biens on the left (~80 % width), a simplified list of loans with an *active loan* selector on the right (~20 %). Its content is defined in `SPEC-property.md` (*Home page*); app-shell provides the route and layout.
- Each page shows its title and "Fonctionnalité à venir".
- Unknown route: "Page introuvable" with a link back to Biens.
- Plain, readable CSS with no UI framework. Usable on a laptop screen. No mobile-specific work.
- Every page except the home page shows a **breadcrumb** above its title, starting with « Accueil » (the home page), e.g. « Accueil › Prêts › Appart Lyon » (added 2026-11-13 at the user's request). It is a `<nav aria-label="Fil d'Ariane">` with the current page marked `aria-current="page"`.
- `web/src/format.ts`: `formatEuros(cents)` → `"1 234,56 €"` and `formatPercent(value)` → `"3,50 %"`, using `Intl` with `fr-FR`.

### Packaging

- `make build` outputs `dist/immo-calc-linux-amd64` and `dist/immo-calc-windows-amd64.exe`, built with `CGO_ENABLED=0`, `-trimpath`, and `-ldflags "-s -w -X main.version=<git describe or 'dev'>"`.
- The Windows binary is built with `-H windowsgui`, so double-clicking opens **no console window**.
- The data file can be changed at startup with `--data`. Switching or loading another file from inside the app belongs to `storage` (see `SPEC-storage.md`).

## Acceptance Criteria

1. `make build` produces both binaries from a clean checkout, with only Docker and make on the host.
2. Running the Linux binary (and the Windows one, checked manually on a Windows machine) opens the browser on the home page « Accueil » showing the French layout within 3 s.
3. Running a second instance of the same version while the first is up opens the browser on the first instance and exits 0. No second server is started. Running a different version stops the first one through `/api/shutdown` and takes over its port.
4. When port 8421 is taken by another program, the app starts on a free port and prints that URL.
5. `GET /api/health` returns the documented JSON with the injected version.
6. Deep links like `/prets/123` reload correctly (SPA fallback). Unknown `/api/x` returns a 404 JSON error.
7. "Quitter" stops the process. A `POST /api/shutdown` with a foreign `Origin` is rejected with 403. Any `/api/*` request whose `Host` isn't `127.0.0.1` or `localhost` is rejected with 403.
11. On Windows, double-clicking the `.exe` opens no console window, and a fatal startup error shows a French message box pointing to `immo-calc.log`.
8. The server never listens on anything other than `127.0.0.1`.
9. `make test`, `make lint`, `make run`, `make dev` and `make debug` all work as described in `SPEC.md`.
10. `--version` prints the version and exits.

## Testing

- `internal/server`: `httptest` tests for health, shutdown (origin check, triggers the shutdown callback), 404 JSON, SPA fallback, static assets served from a test `fs.FS`.
- Port selection, "already running" detection and replacement of a different version: unit tests with injected listener/probe/shutdown functions.
- `internal/browser`: test command selection per OS via an injected `exec` runner. No browser is actually launched.
- Flags/config parsing: table-driven tests.
- Logging: the log file path falls back to the temp directory when the executable's directory isn't writable. Fatal-error reporting goes through an injected notifier, tested without opening a real dialog.
- Frontend: router (redirect `/` → `/biens`, 404 page), `format.ts` (cents, negatives, zero, rounding), Quitter button calls the API and shows the stopped message (mocked fetch).
- Smoke test (`make test`): build the binary, start it with `--no-browser --port 0`, check `/api/health` and `/`, then shut it down via the API.

## Decisions (resolved)

- Windows: **no console window** (`-H windowsgui`). The app is stopped with "Quitter".
- Default port **8421**.
- Default data file **`immo-calc.yaml`** next to the executable, overridable with `--data`. Loading another file from the UI is part of `storage`.
- Target architecture: **amd64 only**.

## Open Questions

None.
