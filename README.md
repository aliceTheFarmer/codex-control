## Installation

```bash
git clone https://github.com/aliceTheFarmer/codex-control.git
cd codex-control
make install
```

---

## `codex-auth`

Use this to switch between previously authenticated Codex accounts.

1. Run `codex login` normally. Codex will write your current credentials to:
   `~/.codex/auth.json`

2. Copy that file into the directory defined by `CODEX_AUTHS_PATH`, naming it
   however you want it to appear in the menu (for example:
   `work-account.auth.json`).

3. Set `CODEX_AUTHS_PATH` in your shell startup file so the CLI knows where to
   find your saved auth files. Example:

```bash
export CODEX_AUTHS_PATH="$HOME/projects/codex-control/auths"
```

4. Run `codex-auth` and select the profile you want.
   The menu highlights the last used profile and sorts entries by recent usage.

---

## `codex-yolo`

Starts Codex in *yolo* mode in the current directory.

```bash
codex-yolo
```

---

## `codex-yolo-resume`

Starts Codex and resumes a previous session.

```bash
codex-yolo-resume
```

---

## `codex-update`

Updates Codex to the latest available version.

```bash
codex-update
```

---

## `codex-update-select`

Lists the latest ~200 released Codex versions and installs the one you select.

```bash
codex-update-select
```
