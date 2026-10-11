# Development

Use Go 1.26 or newer. The verified toolchain and CI pin are 1.26.9. Git is needed for Git integration tests. There is no Node, Python, CGO, database server, or backend requirement to build or run VANTAGE.

## Local checks

```sh
go mod download
go fmt ./...
go test ./...
go vet ./...
go test -race ./...
go build -o bin/vantage ./cmd/vantage
go run ./cmd/vantage --home ./local-state
```

On Windows, name the output `bin/vantage.exe` and use `.\bin\vantage.exe`. Race testing requires a supported platform and C compiler; ordinary builds do not. `--home` isolates scratch data and never changes the default user store. `local-state` is not committed; generated `.db` files are ignored.

The test suite creates real temporary Git repositories, directories containing spaces, tasks, workflows, and subprocesses. Helper test functions are launched only under explicit test environment markers. Tests do not operate on your registered source projects or system configuration. Symlink creation checks run only when the platform allows it.

TUI tests exercise every screen and overlay at the requested dimensions plus smaller windows. Navigation, search and form text, task changes, workflow approvals, live process results, stale responses, and failure messages are checked independently. Linux terminal captures provide additional evidence of real launch, palette navigation, workflow approval/run, resizing, and clean exit. See the engineering report for executed results and remaining platform checks.

## Release archives

From the repository root:

```sh
go run scripts/release.go --version 1.0.0
```

The same command works in PowerShell. The Go script builds Windows, Linux, and macOS for AMD64 and ARM64, sets `CGO_ENABLED=0`, injects version/commit/date metadata, and creates ZIP archives containing the binary, README, security notes, application license, and dependency notices. It writes `dist/checksums.txt`. It never uploads or publishes anything.

`SOURCE_DATE_EPOCH` may be set to a known Unix timestamp to stabilize build dates and ZIP timestamps for a fixed source revision and toolchain. Builds otherwise use the actual build time. This does not claim byte-for-byte reproducibility across different toolchains or uncommitted source states.

Check the archives before publishing:

```sh
cd dist
sha256sum -c checksums.txt
```

On Windows:

```powershell
Get-FileHash .\dist\vantage_1.0.0_windows_amd64.zip -Algorithm SHA256
```

Compare the resulting hash to its exact line in the manifest. Cross-compilation does not substitute for testing the executable on its target OS.

## CI and releases

`ci.yml` has read-only repository permissions. It checks formatting, downloads dependencies, tests on native Linux/Windows/macOS runners, runs `go vet`, runs race checks on Linux, and then packages and uploads build artifacts. The Actions UI is the authority for whether a remote run has passed; merely including the YAML does not establish that result.

`release.yml` has write permission only for the tag-triggered release job. An authorized `v*` tag runs checks and packaging, then prepares a **draft** release with archives and checksums. A maintainer reviews the draft and publishes it deliberately. It does not automatically make a public release. Version tags must be valid semantic versions, such as `v1.0.0`.

## Dependency updates

Keep application dependencies pinned in `go.mod` and `go.sum`. After an update, run the checks above and rescan with the official Go vulnerability checker:

```sh
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

The checker is a development tool, not a runtime dependency. A clean reachable-code result does not constitute an independent security audit. Rebuild dependency notices if distributed modules or versions change.
