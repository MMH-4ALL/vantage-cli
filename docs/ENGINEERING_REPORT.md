# VANTAGE 1.0.0 engineering report

Prepared on 11 October 2026 for `MMH-4ALL/vantage-cli`.

The name search found unrelated software using Vantage, including the Node.js project `dthree/vantage`. This Go application is distributed under the specific `MMH-4ALL/vantage-cli` repository; its name is not claimed to be exclusive. No global package-registry name was registered.

## Delivered application

VANTAGE is a working Go application with a full-screen Bubble Tea dashboard and conventional Cobra commands. Its shipped modules are Overview, Projects, read-only Git, Workflows, Tasks, System, Settings, Doctor, and a searchable command palette. Every built-in menu action is connected to a service. It does not require a backend, paid API, account, Go installation on the target machine, or runtime internet access.

Projects support canonical directory registration, duplicate detection, recent-use ordering, activation, metadata, editor opening, searching, and unregistering while preserving files. Git data comes from real subprocesses and includes branch, root, status, rename-safe paths, commits, and credential-stripped remotes. Workflows support saved literal arguments, directories, project association, timeouts, environment references/overrides, persisted approval, cancellation, bounded output, exit status, and elapsed time. Tasks support titles/descriptions, editing, completion/reopening, project associations, filters, and confirmed deletion.

Settings are validated TOML with atomic saves, graceful handling of invalid files, theme/editor/Git/startup/refresh preferences, and explicit reset. Data is stored in transactional bbolt with tracked schema version 1. This is the brief's allowed alternative to pure-Go SQLite: a small record-oriented application does not need SQL joins, and bbolt provides durable transactions and portable builds. File locks last for individual operations, so the dashboard and CLI can share a data store without blocking one another for an entire session.

## Architecture and interface

`cmd/vantage` contains the entry point and build metadata. `internal/app` wires dependencies and registries; `internal/core` owns business rules; `internal/storage` implements the repository boundary; `internal/services` isolates Git, execution, and system measurements. `internal/cli` and `internal/tui` are clients of those services. Platform-specific cancellation and configuration replacement are separate Go files. The final tree and extension example are documented in [ARCHITECTURE.md](ARCHITECTURE.md).

The interface has a header, sidebar, content frames, footer, list search, selection/focus indicators, forms, first-run workflow approvals, notifications, scrollable help, and a searchable palette. Output has vertical and horizontal scrolling. Midnight, Light, and ASCII monochrome themes are implemented. Potentially blocking operations run outside the UI update loop. Old Git/snapshot loads are canceled and stale responses are discarded.

The screenshots in `docs/screenshots` are rendered from the **actual application running in a Linux pseudo-terminal**, with real registered local data and a real Go test workflow. No image generation, random metrics, fabricated Git data, or mock interface was used. Larger screens expose more details; smaller screens retain usable controls and truncate long values with an ellipsis. Below 55×16, the application explains its minimum dashboard size.

## Installation

Prepared ZIPs contain platform binaries, licenses, and documentation. Extract the archive matching your OS and CPU, verify its SHA-256 entry, then run `vantage` (`vantage.exe` on Windows). These binaries use `CGO_ENABLED=0` and do not need Go installed. Git is needed for Git features, and each workflow/editor executable is supplied by the user.

Linux example after downloading the archive:

```sh
unzip vantage_1.0.0_linux_amd64.zip -d vantage-release
mkdir -p "$HOME/.local/bin"
install -m 755 vantage-release/vantage "$HOME/.local/bin/vantage"
"$HOME/.local/bin/vantage" project add .
"$HOME/.local/bin/vantage"
```

Windows PowerShell example:

```powershell
Expand-Archive .\vantage_1.0.0_windows_amd64.zip -DestinationPath .\vantage-release
.\vantage-release\vantage.exe project add .
.\vantage-release\vantage.exe
```

Source builds use Go 1.26 or newer; the verified toolchain is **Go 1.26.9**. Full installation, CLI, settings, storage locations, and keyboard instructions are in [README.md](../README.md).

## Executed verification

The local suite has **24 behavioral test functions and 3 explicitly invoked subprocess helpers**. Its coverage includes:

- Canonical paths, spaces, duplicate and missing directories, ambiguous project names, and symlinks when available.
- File-preserving unregister, task association changes, title/description editing, completion, and removal.
- Persisted data across reopen, transaction rollback, concurrent writes, and two simultaneous storage handles.
- TOML round trips, validation, preserved corrupt settings, and portable paths.
- Workflow validation, saved consent, literal argument handling, successful execution, failures, timeout/cancellation, output limits, secret redaction, and concurrent log polling.
- Real temporary Git repositories, empty history, staged renames, unusual filenames, detached HEAD, parse errors, missing Git, stripped remote credentials, and a sentinel test verifying that configured Git programs are not executed during inspection.
- CLI help, JSON, persistence across command instances, noninteractive consent, configuration recovery, missing Git, successful workflow results, and failure exit codes.
- TUI navigation, focus, command palette, input shortcut isolation, successful/failed forms, task changes, deletion cancellation, workflow consent/output, stale responses, and visible errors.
- Every screen and four overlays at 80×24, 100×30, 120×35, 160×45, 55×16, 40×8, and 1×1. These checks catch overflow and incorrect height.
- A registered extension action and Cobra command executing without changes to palette rendering.

Local commands executed successfully:

```sh
go fmt ./...
go test ./...
go vet ./...
go test -race ./...
```

The live terminal harness additionally verified launch, screen navigation, palette search/selection, first-run approval, actual workflow execution and output, resizing from 120×35 to 80×24, and clean exit. The CLI was exercised using real binaries as well as integration tests.

Native-runner CI is configured for Linux, Windows, and macOS, with cross-platform packaging after tests. Check the repository's [Actions page](https://github.com/MMH-4ALL/vantage-cli/actions) for remote run results; the presence of CI configuration alone is not represented as a passed remote run.

## Builds and compatibility

| Target | Local compilation | Local runtime verification |
| --- | --- | --- |
| Linux AMD64 | Passed | CLI, full-screen TUI, real Git, workflows, resize, exit, tests, race checks |
| Linux ARM64 | Passed | Not run on ARM hardware |
| Windows AMD64 | Passed | Not run locally on Windows 10/11 |
| Windows ARM64 | Passed | Not run locally on ARM Windows |
| macOS AMD64 | Passed | Not run locally on an Intel Mac |
| macOS ARM64 | Passed | Not run locally on Apple Silicon |

The runtime environment was a Linux AMD64 container. Individual Ubuntu, Debian, Fedora, Arch, ChromeOS Linux, Windows Terminal/PowerShell, and macOS version combinations have not all received manual terminal testing. Successfully compiling a platform binary does not verify its terminal behavior. macOS builds are unsigned and not notarized; there is no GUI installer or automatic PATH modification.

## Measured performance

These are observations from this execution environment, not promises for a Chromebook or another user's laptop.

| Measurement | Observed result | Method / scope |
| --- | --- | --- |
| Read and sort a 500-project snapshot | 1.47 ms/op; 153,563 bytes allocated/op | Go benchmark using explicitly synthetic benchmark records, including database access |
| Render 120×35 Overview | 0.120 ms/op; 44,918 bytes allocated/op | Go benchmark under a headless test environment; excludes terminal transport and Git loading |
| CLI `project list --json` startup/result | 6.05 ms median; 13.80 ms 95th-percentile sample | 25 warm subprocess runs with an existing local store |
| TUI Go heap | 0.82 MiB | Actual System-screen snapshot around 3 seconds after launch, before the test workflow |
| TUI runtime reservation | 13.87 MiB | Same actual snapshot; distinct from resident memory |
| TUI resident memory | 12.68 MiB | Same snapshot using Linux `/proc/self/statm` |

Memory changes as the app renders, runs jobs, and performs garbage collection; the snapshot is not a long-term maximum. External idle CPU sampling was unavailable because the harness could not inspect the child's `/proc/<pid>` view. No idle CPU number is invented. System polling is limited to visible Overview/System screens and the configured interval. Workflow output is capped and cached between writes.

## Security review

The review checked argv execution, project removal, path validation, secret environment handling, output control sequences, cancellation, database contention, corrupt settings, atomic replacement, and action approvals. Project directories are never recursively deleted. Workflows do not execute on launch. Git does not modify history or force operations. The application contains no network client or telemetry path.

The official Go vulnerability checker was run for **both Linux and Windows build profiles** after dependency updates. Both scans returned **“No vulnerabilities found.”** An older `golang.org/x/sys` version had been flagged by the scanner for a Windows issue; it was upgraded to the fixed `v0.44.0`. Dependency license notices for 27 linked modules and the Go runtime are preserved in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).

Saved workflows run with normal user permissions. Approval is not an executable signature or sandbox. Logs redact known environment secret values, but cannot identify every credential an arbitrary program prints; credentials should not be stored in arguments, records, or settings. Data is not encrypted at rest. Cancellation cannot undo prior side effects, and deliberately detached processes can escape a process group. See [SECURITY.md](../SECURITY.md).

**Verification basis:** this report is supported by executed automated tests, actual subprocess/Git operations, real terminal captures, successful cross-compilation, measured benchmarks, and current vulnerability database checks. It is an engineering record from the build environment, not an independent security audit or a claim of universal platform compatibility.

## Delivery and next work

The source includes the MIT license, README, contributor/security/changelog documents, architecture/development instructions, dependency notices, test suite, screenshots, six-target packager, checksum manifest generation, CI, and authorized-tag draft-release automation. The source repository is `MMH-4ALL/vantage-cli`; the delivery bundle contains all six platform archives and checksums. Public release publication remains a deliberate maintainer action.

Recommended next work is native terminal testing on Windows 10/11 and macOS, followed by workflow history, project tags, snippets, better task organization, and a read-only Git diff viewer. Cloud sync, dynamic plugins, AI providers, Docker/SSH tools, and consequential Git operations are future scope; they are not advertised as implemented.
