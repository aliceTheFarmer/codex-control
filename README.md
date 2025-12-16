# codex-control

1. Clone this repository.
2. Run `make install` (needs sudo to copy binaries into `/usr/bin`).

## Binaries

- **codex-yolo** – Runs `codex --dangerously-bypass-approvals-and-sandbox` with any arguments you pass after `--`.
- **codex-yolo-resume** – Same as above but automatically runs `codex resume`.
- **codex-update** – Downloads the latest Codex release, stages it in `/tmp/codex-control`, and installs it to `/usr/bin/codex`.
- **codex-update-select** – Lists the last 200 releases; pick one to install. The UI uses ↑/↓/digits to navigate and Enter to install.
- **codex-auth** – Lets you pick an auth profile from the folder pointed to by `CODEX_AUTHS_PATH` (or `--auths-path=<folder>`). When you confirm, it copies the selected file to `~/.codex/auth.json`.

## Auth Profiles

1. Run `codex login` normally so Codex writes your current credentials to `~/.codex/auth.json`.
2. Copy that file into the folder defined by `CODEX_AUTHS_PATH`, naming it how you want it to appear in the menu (e.g., `work-account.auth.json`).
3. Set `CODEX_AUTHS_PATH` in your shell startup file so the CLI knows where to find those saved auth dumps, for example:

```bash
export CODEX_AUTHS_PATH="$HOME/projects/codex-control/auths"
```

`codex-auth` scans that directory, shows every file, and writes your selection back to `~/.codex/auth.json`. The tool errors if the variable is empty or the folder has no files, so maintain one `.auth.json` per previously authenticated account inside that directory.
