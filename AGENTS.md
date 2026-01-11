# Project Guidance for codex-control

## Workflow Expectations
- Target Go 1.25.1+ and keep `go.mod` tidy by running `make tidy` whenever dependencies change.
- Primary binaries live under `cli/` and reuse packages from `internal/`; never embed business logic inside the runners.
- `make build` populates `Release/` with the five binaries (`codex-yolo`, `codex-yolo-resume`, `codex-update`, `codex-update-select`, `codex-auth`).
- `make install` must run `make clean`, `make build`, then copy every binary to `/usr/bin` using `sudo install -m 0755`.

## CLI Conventions
- All runners expose `-v, --verbosity=<0|1|2>` plus the options listed in their usage blocks. Long options require `--flag=value`; short aliases consume the next argument.
- Proxy binaries (`codex-yolo*`) pass additional Codex arguments after `--`. Example: `codex-yolo -- -t gpt-4o-mini`.
- `codex-update`/`codex-update-select` always work from `/tmp/codex-control`, wiping it before each run and cleaning it afterward. They install Codex to `/usr/bin/codex` directly, so run them with sufficient privileges when needed.
- `codex-auth` reads `CODEX_AUTHS_PATH` by default; use `--auths-path=<path>` during testing.

## Interactive Menus
- Bubble Tea + Lipgloss menus mirror the two-panel layout from `services-control`. List view: numbered entries with pointer glyphs, `↑/↓/j/k` navigation, numeric jump input, `R` reload, `Esc` clears. Action view: same glyphs plus `Enter` to execute, `Esc` to return.
- Right-hand panel displays action output; `verbosity=2` adds the environment dump before the JSON payload. `tea.Sequence(action, tea.Quit)` is the standard pattern for one-shot menus.

## Testing & Validation
- Run `go test ./...` after touching shared logic.
- Invoke `make build` before shipping changes to ensure `Release/` is refreshed.
- For menu-driven apps, include at least one manual smoke check (e.g., `codex-auth --auths-path=./auths`) before distribution.
