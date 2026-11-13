# Spec: storage

> Module of [immo-calc](SPEC.md). Depends on: `app-shell`. Shared conventions are in `SPEC.md`.

## Objective

Keep all the user's data (properties, loans, settings) in **one human-readable YAML file**. It must survive restarts, never be silently corrupted or overwritten, be easy to move to another machine (export/import), and let the user open another data file from the UI. The app remembers the last file used.

`storage` knows nothing about loans or properties. It provides a generic, versioned document store that the `loan` and `property` modules write their own sections into. That keeps the dependency direction `loan`/`property` → `storage`.

## Scope

**In:** data file resolution, load/validate/save, atomic writes, schema versioning and migrations, conflict detection with external edits, backups, export, import (replace), opening another file, creating a new file, remembering the last file, *Paramètres → Données* UI.

**Out:** the content of the `loans`, `properties` and `settings` sections. Their schemas are defined in `SPEC-loan.md` and `SPEC-property.md`.

## Data file format

```yaml
# Fichier de données immo-calc — ne pas modifier schema_version
schema_version: 1
settings: {}      # owned by property (TMI, tax regime…)
loans: []         # owned by loan
properties: []    # owned by property
```

- UTF-8, keys in `snake_case`, amounts in integer cents (see `SPEC.md`).
- A missing section is treated as empty. Unknown top-level keys are a validation error, which catches typos in hand edits.
- Comments added by hand are **not preserved** when the app saves (accepted limitation).

## App config (remembering the last file)

A small file `immo-calc.config.yaml` next to the executable (falling back to the user config dir, `os.UserConfigDir()/immo-calc/`, if that folder isn't writable):

```yaml
last_data_file: /home/me/Documents/immo/famille.yaml
```

**Data file resolution at startup**, first match wins:
1. `--data <path>` flag. This does not overwrite `last_data_file`.
2. `last_data_file` from the app config, if that file exists.
3. `immo-calc.yaml` next to the executable.

If `last_data_file` points to a file that no longer exists, the app falls back to step 3 and the UI shows a dismissible banner: « Le fichier `<path>` est introuvable, immo-calc utilise `<default>` ».

If the resolved file doesn't exist (first launch), the app starts with an empty document and creates the file on the first save.

## Behavior

### Load and validation

On load, the file is parsed, migrated if older, then validated: structure first, then each section by the validator registered by its owning module.

**At startup, any failure means refuse to start.** This covers malformed YAML, unknown keys, invalid section content, and a `schema_version` newer than the app supports. The error goes through the app-shell fatal-error path (message box on Windows) with a French message naming the file, the line when known, and the cause. Example: « Le fichier /…/immo-calc.yaml est invalide (ligne 12) : montant négatif pour le prêt « Appart Lyon ». Corrigez-le ou déplacez-le, puis relancez immo-calc. »

### Save

- Data is kept in memory behind a mutex. Every successful change from the API is saved immediately.
- **Atomic write:** write to a temp file in the same directory, `fsync` it, then rename it over the data file.
- **External edit detection:** the SHA-256 of the file is recorded on load and after each save. Before saving, the current file is hashed. If it changed (someone edited it in a text editor while the app was running), the save is refused with `409 {"error":{"code":"file_changed_externally",…}}`. The UI then shows « Le fichier a été modifié en dehors d'immo-calc » with a **Recharger** button, which reloads from disk and runs the normal validation (an invalid file is reported, the in-memory state is kept).

### Backups

Backups are made **only before an import or a file switch** (Ouvrir / Nouveau fichier), never on ordinary saves. The current data file is copied to `<name>.bak-YYYYMMDD-HHMMSS.yaml` in the same folder. If the current file doesn't exist yet, no backup is made. Old backups are never deleted automatically.

### Migrations

There is a registry of `func(v N document) → v N+1 document` functions. A file older than the current version is migrated in memory. It is written back, after a backup, only on the next save. v1 has no migrations, but the mechanism is in place and tested with a fake v0 → v1.

### Export

`GET /api/data/export` downloads the in-memory document as YAML, in the same format as the data file, with the name `immo-calc-export-YYYY-MM-DD.yaml`.

### Import (replace)

1. In *Paramètres → Données*, the user clicks **Importer…** and picks a `.yaml` file in the browser.
2. The UI shows a confirmation: « Toutes les données actuelles (N biens, M prêts) seront remplacées par celles du fichier importé. Une sauvegarde du fichier actuel sera créée. Continuer ? »
3. On confirm, the browser sends `POST /api/data/import` with the file content. The server parses, migrates and validates it **completely**. If it fails: `400` with a French message, and nothing changes.
4. If it succeeds: back up the current file, replace the in-memory document, save. The UI reloads its data and shows « Import terminé ».

The data file path doesn't change: imported data is written into the current file.

### Open another file

A browser picker gives the app a file's content, never its path. So the Go process, which runs on the user's machine, opens a **native OS file dialog** instead. That keeps the "pick a file" experience and still lets the app remember the path.

1. **Ouvrir un autre fichier…** → `POST /api/data/open` → the server shows a native "Ouvrir" dialog filtered on `*.yaml`.
2. Cancel → `204`, nothing changes.
3. File chosen → load and validate it. If it's invalid: `400` with a French message, and the app stays on the current file (no crash, no change).
4. If it's valid: back up the current file, switch the active file to the chosen path, write `last_data_file` in the app config. The UI reloads and shows the new path.

If no native dialog is available (Linux without `zenity`/`kdialog`), the UI falls back to a text field where the path is typed or pasted (`POST /api/data/open {"path": "…"}`), with the same validation.

Dependency: **`github.com/ncruces/zenity`** (pure Go: native dialogs on Windows, `zenity`/`kdialog` on Linux). Approved.

### New file

1. **Nouveau fichier…** → `POST /api/data/new` → the server shows a native "Enregistrer sous" dialog (default name `immo-calc-nouveau.yaml`, filter `*.yaml`). The same text-field fallback applies.
2. Cancel → `204`, nothing changes.
3. If the chosen path **already exists**, it is refused (`400` « Ce fichier existe déjà, choisissez un autre nom ou utilisez « Ouvrir » »). Existing files are never overwritten.
4. Otherwise: back up the current file, write an empty v1 document at the new path immediately, make it the active file, and write `last_data_file`. The UI reloads (empty lists) and shows the new path.

### API

| Route | Behavior |
|---|---|
| `GET /api/data/info` | `{"path": "...", "schemaVersion": 1, "counts": {"properties": N, "loans": M}, "warning": "..."?}` |
| `GET /api/data/export` | YAML download |
| `POST /api/data/import` | body = YAML; `200` with new counts, or `400` |
| `POST /api/data/open` | native dialog (no body) or `{"path"}`; `200` with new info, `204` on cancel, `400` if invalid |
| `POST /api/data/new` | native save dialog (no body) or `{"path"}`; `200` with new info, `204` on cancel, `400` if the file exists or isn't writable |
| `POST /api/data/reload` | reload from disk after an external edit; `200` or `400` |

All `POST` routes use the same-origin check from app-shell.

### Go interface for other modules (sketch)

```go
// Section reads a typed section; a missing section yields the zero value.
func Get[T any](s *Store, key string) (T, error)
// Update applies fn to a typed section and saves atomically; fn's error aborts without saving.
func Update[T any](s *Store, key string, fn func(*T) error) error
// RegisterSection declares a section and its validator (called at startup by loan/property).
func (s *Store) RegisterSection(key string, validate func(raw []byte) error)

// A validator returns a *FieldError to point at the bad value. Path is relative to the
// section ("[2].principal_cents", "marginal_tax_rate_pct"); storage maps it to the YAML
// line for the startup error. Message is French. Any other error is reported on the
// section's own line.
type FieldError struct{ Path, Message string }
```

### UI: Paramètres → Données

This shows the current file path, the counts, and the buttons **Ouvrir un autre fichier…**, **Nouveau fichier…**, **Exporter**, **Importer…**. It also shows the banners (remembered file not found, file changed externally) and error messages, all in French.

## Acceptance Criteria

1. On first launch, with no file present, the app starts empty and creates `immo-calc.yaml` next to the executable on the first change.
2. After a restart, all data is still there. The file is valid YAML in the format above.
3. Killing the process during a save never leaves a truncated or half-written data file (atomic rename).
4. A malformed file, an invalid section or a too-new `schema_version` at startup means the app refuses to start and shows a French error naming the file and the line.
5. Editing the file by hand while the app runs, then changing something in the UI, gives a 409 and the "modifié en dehors" banner. Nothing is overwritten. **Recharger** loads the edited content.
6. Export, then import on another instance, gives back the same data (round trip).
7. Importing an invalid file changes nothing. A valid import shows the confirmation, creates exactly one `.bak-…yaml`, and replaces the data.
8. Opening another file switches to it, creates a backup of the previous one, and after a restart the app reopens that file. `--data` overrides it without overwriting the remembered path.
9. An invalid file chosen in "Ouvrir" is rejected and the app stays on the current file.
10. A remembered file that has disappeared falls back to the default file and shows the banner.
11. No backup is created by ordinary saves.
12. **Nouveau fichier…** creates an empty file at the chosen path, backs up the previous file, switches to it and remembers it. Choosing an existing path is refused and nothing is overwritten.

## Testing

- Store unit tests in `t.TempDir()`: round trip, empty/missing sections, unknown keys rejected, validators invoked, line numbers in errors.
- Atomic write: inject a failing writer/renamer and check that the original file is untouched.
- Hash conflict: modify the file between load and save, expect `ErrFileChangedExternally`.
- Migrations: fake v0 → v1 migration applied, too-new version rejected.
- Resolution order: table-driven (`--data`, remembered and present, remembered and missing, default).
- Backups: created on import and switch only, naming format with an injected clock.
- File dialog behind an interface (`Picker`), faked in tests (cancel / path / error / unavailable → fallback).
- `httptest` for each endpoint, including the same-origin rejection.
- Frontend: Paramètres → Données component (confirmation dialog, error and banner display, fallback path field) with mocked API.

## Decisions (resolved)

- Open another file: native OS dialog via `github.com/ncruces/zenity`, with a typed-path fallback. The last file is remembered.
- Import replaces all data, after a confirmation and a backup.
- Backups only before import and file switch (open / new).
- Malformed file at startup: refuse to start.
- **Nouveau fichier…** is in scope.

## Open Questions

None.
