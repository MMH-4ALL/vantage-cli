# Security

VANTAGE is a local application. It does not send telemetry, contact a backend, require credentials, or automatically execute source-project scripts. A workflow you explicitly approve may run any installed executable with your user permissions; its program can access files or networks according to those permissions. Registering a folder is not an execution approval.

## Protections

- Executables receive literal argument arrays through `os/exec`; shell interpolation is not used by default.
- Project removal unregisters metadata only; it never removes the source directory.
- Git operations are fixed, read-only commands with bounded execution and machine-readable output.
- Git inspection disables repository-configured filesystem monitors, content filters, pagers, and automatic signature verification. Submodule working trees are excluded to avoid executing their configured programs.
- Newly saved workflows require explicit first-run approval. A scripted `--yes` is a deliberate approval.
- Workflow directories, names, arguments, environment keys, and timeouts are validated.
- Secret-looking literal environment overrides are rejected. `--inherit-env` stores variable names instead of values.
- Known secret environment values are redacted from captured output; terminal control sequences and unsafe direction controls are removed.
- Output is bounded to the latest 2 MiB; Git capture rejects oversized output rather than parsing a partial result.
- Owned processes support context cancellation and platform-specific descendant termination.
- Local data uses transactional storage, short file locks, and schema checks. Configuration updates use validated atomic replacement.
- Unix data/config permissions request 0600 files and 0700 directories. Windows permissions follow the user's filesystem ACLs.
- Invalid settings are preserved and reported; replacing them requires confirmation. Destructive record removals always require consent.

## Practical limits

Approval is tied to the saved command definition, not a cryptographic hash of the executable. Replacing an approved executable changes what runs. Credentials embedded in command arguments, task text, descriptions, arbitrary log content, or unusually named environment variables cannot be reliably identified. Do not store secrets there. Redaction is a best-effort presentation safeguard, not a secure vault. Data is not encrypted at rest; protect and back it up using your OS facilities.

Cancellation cannot recover changes already made by a workflow. A program can intentionally detach itself from its process group; Windows process-tree behavior also needs native verification. The application has no sandbox for approved commands. CI and dependency checks are useful evidence, not an independent penetration test.

## Reporting

Do not post credentials or a working exploit that exposes someone's files in a public issue. Use GitHub's private vulnerability-reporting facility if the maintainer enables it, or establish a private contact with the repository owner first. This file does not claim that a private-reporting endpoint has already been configured. For ordinary reproducible bugs without sensitive details, open a repository issue with version, OS, terminal, isolated steps, and sanitized output.
