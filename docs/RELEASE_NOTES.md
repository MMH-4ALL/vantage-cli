# VANTAGE 1.0.0

VANTAGE is a local developer command center with a full-screen terminal dashboard and a scriptable CLI. This release implements project management, a read-only Git dashboard, approved workflows, local tasks, system information, persistent TOML settings, and environment diagnostics.

Archives are prepared for Windows, Linux, and macOS on AMD64 and ARM64. They include standalone executables, licenses, and dependency notices; check `checksums.txt` before installing. Build and runtime verification are reported separately in `docs/ENGINEERING_REPORT.md`.

Start with `vantage project add .`, then run `vantage`. Use Ctrl+K to find actions, `?` for help, and `vantage --help` for CLI commands. Newly saved workflows need first-run approval. Git is read only, and unregistering a project preserves its directory.

This is a draft release prepared for review. Native terminal checks across Windows 10/11, individual Linux distributions, and macOS versions remain necessary before broad compatibility claims. There is no installer, cloud sync, workflow history, runtime plugin loader, or AI module in 1.0.
