# Architecture

VANTAGE has two clients: Cobra commands and a Bubble Tea dashboard. Both call the same core services; neither contains a separate version of project, task, or workflow business rules.

```text
cmd/vantage/main.go             Process entry and build metadata
internal/app/                  Dependency wiring, diagnostics, registries
internal/cli/                  Cobra commands, JSON, consent, exit codes
internal/config/               Validated TOML, standard paths, atomic writes
internal/core/                 Project, task, workflow rules and validation
internal/domain/               Shared persisted records
internal/storage/              Repository interface, transactional bbolt store
internal/services/executor/    Safe argv execution, output bounds, cancellation
internal/services/git/         Machine-readable, read-only Git integration
internal/services/system/      Runtime and platform measurements
internal/tui/                  Model, update, forms, screen rendering, tests
internal/tui/components/       Dimension-safe frames and clipping
internal/tui/themes/           Midnight, light, and ASCII monochrome palettes
tests/integration/             CLI lifecycle and subprocess behavior
scripts/release.go             Six-target packager and checksums
.github/workflows/             CI and authorized-tag draft releases
docs/                         Instructions, actual captures, verification report
```

## Data and transactions

`storage.Repository` is the boundary used by `core.Service`. A snapshot contains schema version 1, projects, tasks, workflows, and the active project ID. Records use random 48-bit identifiers, native filesystem paths, and timestamped metadata. Project lists are sorted by actual last use.

This release uses bbolt rather than SQLite. The state is small and does not need joins, SQL queries, or a separate database process. bbolt supplies ACID transactions, file locking, fsync-backed commits, and pure-Go cross-compilation with a smaller dependency tree. This is an intentional alternative permitted by the brief. Each operation opens the database, takes a short file lock, performs one transaction, and closes it. An idle dashboard therefore allows separate CLI processes to read and write the same data. A one-second lock timeout bounds contention. Schema versions are checked before records are read; unsupported versions are rejected without overwriting data. Version 1 has an initialization migration, not a speculative series of future migrations.

The snapshot representation keeps the code small and easy to inspect. Operations decode the stored state, so extremely large task/workflow databases would benefit from indexed buckets in a later version. Tests cover simultaneous handles and transaction rollback. Backups should be copied with no write in progress.

Projects are identified by canonical paths. Resolving symlinks catches duplicate registrations. Removing a project changes records only; there is no recursive filesystem deletion in core code. Its tasks and workflows remain saved, with the association cleared. Workflow working directories remain explicit.

## Subprocess boundary

`executor.Run` receives an executable and argument array. It validates the directory and timeout, merges allowed environment overrides, starts a context-bound command, and captures stdout/stderr through a synchronized 2 MiB log. Environment secrets remain in the live process; secret-looking literal overrides are rejected. Known secret values are redacted and terminal controls are stripped before presentation. No implicit shell is involved.

On Unix, subprocesses enter a separate process group so cancellation terminates descendants. Windows uses the built-in `taskkill.exe` from SystemRoot with a two-second timeout and a direct-process fallback. There is no system-wide installation or configuration change. The CLI prints the captured result after completion. The TUI polls a safe log snapshot while the job runs and provides a scrolling viewport.

Git uses `status --porcelain=v1 --branch -z`, fixed `log` fields separated by NUL, and `--no-optional-locks`. Arguments are constructed as arrays, preserving names containing whitespace, quotes, or newlines. Remote URLs have user information and queries removed before display. Git calls are bounded to eight seconds each and never run on the UI update thread.

Inspection also disables filesystem monitors, pagers, and automatic signature verification. It enumerates configuration key names and overrides every configured external content filter before status runs. Submodule working trees are excluded. Bypassing content filters can change reported status in repositories that require those transformations; this tradeoff keeps project inspection from automatically executing repository programs.

## UI state

The UI model owns navigation, focus, input widgets, selection, notifications, confirmations, themes, a help offset, and workflow presentation state. Background commands load snapshots and Git information. Generation IDs keep stale responses from replacing a newer screen. Closing or refreshing a screen cancels its old load context; quitting cancels owned jobs. The live process runner is joined before resources are closed.

Rendering uses measured cell widths through Lip Gloss and ANSI-aware clipping. The frame component has explicit width and height budgets. Lists render a visible window around the selected row. The interface has a sidebar, header, active panels, footer, modal forms, first-run approvals, command palette, and a scrolling output viewport. System refresh ticks collect information only while Overview or System is visible. Spinner ticks stop when work stops.

## Extension points

`app.Registry` accepts destination metadata, palette actions, and Cobra command factories. Built-in services are injected through `App`. Additional services can be placed beside existing services and attached to the application wiring; a dynamic runtime plugin loader is deliberately not included.

A standalone palette action can be registered without changing the palette renderer:

```go
registry.AddAction(app.Action{
    ID: "notes:open",
    Title: "Open developer notes",
    Description: "Use an independently supplied notes service",
    Handler: notes.Open,
})
```

Likewise, `registry.AddCommand(factory)` attaches a Cobra command factory. The extension test in `internal/app/registry_test.go` registers and executes an action and a command. Adding a new **screen** also requires a renderer and screen state/update behavior; registering a title alone does not implement a screen.

## Configuration

Settings are distinct from application data. TOML parse or validation failures produce defaults plus a warning while preserving the original file. Validated edits use a private temporary file, fsync, and an atomic replacement; Windows uses MoveFileEx. Read-only diagnostics copy settings under a lock when necessary. No project folder, global Git setting, shell profile, or system PATH is modified.
