# Changelog

## 1.0.0 — 2026-10-11

Initial implementation of Vantage Developer Command Center.

- Full-screen Bubble Tea dashboard with keyboard navigation, search, command palette, forms, confirmations, contextual help, notifications, and responsive frames.
- Shared project, task, workflow, system, configuration, and diagnostic services for the CLI and TUI.
- Transactional pure-Go persistence with schema tracking, short file locks, concurrent-process handling, and atomic TOML preference updates.
- Project registration, activation, recent-use ordering, editor opening, canonical path handling, and file-preserving unregister.
- Read-only Git branch, working-tree status, changes, renames, commits, and sanitized remotes using machine-readable data.
- Workflow approval, literal argv execution, timeout/cancellation, environment references, redacted bounded output, live TUI logs, and exit-code handling.
- Editable local tasks with completion, reopening, project association, and filters.
- Midnight, light, and monochrome themes, startup and editor settings, refresh control, and a doctor command.
- Automated core, integration, subprocess, registry, and UI tests; native Linux terminal captures; six-target cross-compilation and checksum packaging.
- Documentation, dependency license notices, native-runner CI, and authorized-tag draft-release automation.

Platform verification, actual measurements, and limits are recorded in `docs/ENGINEERING_REPORT.md`. Compile targets are not represented as universally tested OS versions.
