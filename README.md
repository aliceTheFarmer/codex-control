## Installation

```bash
git clone https://github.com/aliceTheFarmer/codex-control.git
cd codex-control
make install
```

---

## Configuration

Each binary reads a YAML config file from `/mnt/config/.<binary>/config.yaml` when
`/mnt/config` exists. Otherwise it falls back to `~/.<binary>/config.yaml`. On
first run, the file is created with default values. CLI flags override the YAML
values for that run.

---

## `codex-auth`

Use this to switch between previously authenticated Codex accounts.

1. Run `codex login` normally. Codex will write your current credentials to:
   `~/.codex/auth.json`

2. Copy that file into a directory where you want to store profiles, naming it
   however you want it to appear in the menu (for example:
   `work-account.auth.json`).

3. Point `codex-auth` at that directory using `--auths-path` or by updating the
   YAML config file (for example `~/.codex-auth/config.yaml`):

```bash
auths-path: "/home/you/projects/codex-control/auths"
verbosity: 1
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
