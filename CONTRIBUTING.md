# Contributing

Keep changes focused and describe the user behavior they improve. Add services behind existing boundaries and reuse them in both the CLI and TUI. Do not add fake data, runtime network dependencies, automatic project-script execution, or repository mutations to the read-only Git service.

Use Go 1.26 or newer. Run `go fmt ./...`, `go test ./...`, and `go vet ./...`; run `go test -race ./...` where supported. Use an isolated `--home` directory while developing. For interface changes, check the required terminal sizes, text entry, focus, keyboard help, overflow, empty states, errors, and all themes. Preserve confirmation for deleting records, overwriting preferences, and first workflow execution.

Documentation should describe shipped behavior and mark plans as plans. Include the actual tests run in a pull request. Keep generated binaries and personal database files out of commits. See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for packaging and [ARCHITECTURE.md](docs/ARCHITECTURE.md) for extension points.
