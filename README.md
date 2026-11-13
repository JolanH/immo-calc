# immo-calc

A local desktop app to work out whether **buying a flat to rent it out in France** is a good investment.

You enter a flat (price, rent, charges, taxe foncière, DPE…) and immediately get the monthly gross income, the monthly income after tax, the yearly net income and the *rentabilité*. You can simulate loans, link a flat to a loan to see the cash-flow and the return on your own contribution, and compare tax regimes (micro-foncier, réel foncier, micro-BIC, LMNP au réel) side by side.

It is built for one household, not as a public product. The interface is in French. The calculations approximate French tax and loan rules **realistically, not exactly**: it is a decision aid, not a tax return.

## Using the app

immo-calc is a single executable for Linux or Windows (amd64), with nothing to install.

1. Download `immo-calc-linux-amd64` or `immo-calc-windows-amd64.exe` (see [Releases](#releases)).
2. Double-click it. It starts a local web server on `127.0.0.1` and opens your browser on it.
3. Stop it with the **Quitter** button in the header. Closing the browser tab leaves the server running.

It never contacts the network: everything, including the interface, is embedded in the executable, and the server only listens on `127.0.0.1`.

### Where your data lives

All data (flats, loans, settings) is in **one human-readable YAML file**. At startup the app picks it in this order:

1. the file given with `--data <path>`;
2. the last file you opened from the app, if it still exists (remembered in `immo-calc.config.yaml` next to the executable);
3. `immo-calc.yaml` next to the executable, created on first save.

From *Paramètres → Données* you can open another file, create a new one, **export** everything to a single file and **import** it back (for example on another computer). The app makes a `<name>.bak-YYYYMMDD-HHMMSS.yaml` copy before every import or file switch. If you edit the file in a text editor while the app is running, the app refuses to overwrite your changes and offers to reload.

### Command-line options

| Option | Default | Effect |
|---|---|---|
| `--port <n>` | `8421` | Preferred port (`0` = any free port). If it is taken, the app uses a free one. |
| `--data <path>` | see above | Data file to use. Does not change the remembered file. |
| `--no-browser` | off | Don't open the browser. |
| `--debug` | off | Verbose logs. |
| `--version` | | Print the version and exit. |

Logs go to `immo-calc.log` next to the executable (or the temp directory if that folder isn't writable). Starting the app a second time just opens the browser on the instance already running.

## Development

### Requirements

Only **Docker** and **make**. Every tool (Go, Node, linters, Delve) runs inside the project's Docker images (`docker/go.Dockerfile`, `docker/node.Dockerfile`), so the toolchain is identical on every machine and in CI. The images are built automatically the first time a target needs them.

```sh
make            # list the main targets
make run        # build and run the app → http://localhost:8421
```

### Makefile targets

| Target | What it does |
|---|---|
| `make help` | List the main targets (the default target). |
| `make run` | Build the frontend and run the Go app → <http://localhost:8421>. |
| `make dev` | Vite dev server with hot reload **plus** the Go server → <http://localhost:5173>. Vite forwards `/api` to Go. Use this when working on the interface. |
| `make debug` | Run the Go server under Delve, headless on `127.0.0.1:2345`. Attach from your IDE (e.g. GoLand / VS Code "remote" debug configuration). |
| `make test` | All tests: `test-go`, `test-web` and `smoke`. |
| `make test-go` | Go tests with `-race` and coverage. |
| `make test-web` | Frontend tests (Vitest). |
| `make smoke` | Builds the real binary, starts it, and checks it serves the API and the interface, then shuts it down. |
| `make lint` | `lint-go` and `lint-web`. |
| `make lint-go` | `gofmt`, `go vet` for Linux and Windows (including the smoke test). |
| `make lint-web` | ESLint, Prettier (check only) and `vue-tsc`. |
| `make build` | Cross-compile the release binaries into `dist/`: `immo-calc-linux-amd64` and `immo-calc-windows-amd64.exe` (no console window). The version comes from `git describe`. |
| `make images` | (Re)build the Docker images. |
| `make clean` | Remove `dist/`, `web/dist/` and the Go caches. Keeps `node_modules` and your dev data. |

Good to know:

- **Dev data is separate.** `run`, `dev` and `debug` use `.cache/dev/immo-calc.yaml` and never open a browser by themselves. Your real data file is never touched during development.
- **Caches live in `.cache/`** (Go modules, Go build cache, npm cache), inside the project and gitignored. Files are created with your user id, not root.
- **The frontend is built when needed.** The Go binary embeds `web/dist`, so Go targets rebuild it when the frontend sources change, and `npm ci` re-runs when `web/package-lock.json` changes.
- **Interactive targets use the host network** so the app can keep listening on `127.0.0.1` only.
- Before pushing, run `make lint test`: it is what CI runs.

### Project layout

```
cmd/immo-calc/       entry point: flags, startup, port selection, fatal-error reporting
internal/server/     HTTP routing, JSON helpers, static/SPA handler, origin checks
internal/storage/    YAML data file, atomic saves, migrations, export/import, file dialogs
internal/loan/       loan maths, saved simulations, API
internal/property/   profitability and tax maths, settings, API
internal/browser/    open the default browser
internal/notify/     native error message (Windows message box, notify-send on Linux)
internal/applog/     log file
web/                 Vue 3 + TypeScript + Vite frontend, embedded with go:embed
docker/              development images
```

All financial calculations are done in Go and exposed as a JSON API. The frontend only collects inputs and formats results, so there is one tested source of truth for the rules.

### Documentation

- [`SPEC.md`](SPEC.md): goals, shared conventions, tech stack. Each module has its own spec: [`SPEC-app-shell.md`](SPEC-app-shell.md), [`SPEC-storage.md`](SPEC-storage.md), [`SPEC-loan.md`](SPEC-loan.md), [`SPEC-property.md`](SPEC-property.md).
- [`docs/tax-regimes.md`](docs/tax-regimes.md): how each tax regime is calculated and when it pays off.
- [`tasks/legal-rates.md`](tasks/legal-rates.md): the official sources behind the rates and ceilings.

## Continuous integration

GitHub Actions (`.github/workflows/ci.yml`) runs on every push to `develop` / `main` and on every pull request:

- **Lint and tests** on Linux: `make lint` and `make test`, in the same Docker images as local development.
- **Tests on Windows**: the Go tests and the smoke test on a real Windows runner, since the app ships as a Windows `.exe`.

## Releases

Pushing a tag starting with `v` builds the executables:

```sh
git tag v1.2.0
git push origin v1.2.0
```

After lint and tests pass on Linux and Windows, CI runs `make build` and checks that the binaries report exactly the tag as their version. It then publishes a **GitHub Release** named after the tag (repo page → *Releases*), with:

- `immo-calc-v1.2.0-linux-amd64.tar.gz`: the Linux executable (in a tarball to keep its executable bit);
- `immo-calc-windows-amd64.exe`: the Windows executable;
- `SHA256SUMS`: checksums of both, to check with `sha256sum -c SHA256SUMS`.

The release notes list the commits since the previous tag. A tag with a suffix, such as `v1.2.0-rc1`, is published as a pre-release. Re-running the workflow for a tag replaces the release's files. The same files also stay available as artifacts of the workflow run (Actions tab → the run → *Artifacts*).

To build them locally instead: `make build`, then look in `dist/`.
