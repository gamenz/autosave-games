# Spec: autosave-games MVP

- Status: Draft (awaiting review)
- Date: 2026-10-17
- Platform: Windows 10/11, single user

## 1. Goal

Automatically back up a game's save files when the game starts (pre-session
snapshot) and after its process has fully exited, with no manual action.
Provide a CLI to list and restore backups.

### Non-goals (MVP)

- No GUI window (tray menu + YAML config only).
- No auto-detection of installed games, no Ludusavi manifest import.
- No cloud upload logic (user may point `backup_root` at a synced folder).
- No Windows Service; no admin rights required.
- Linux/macOS support.

## 2. Key decisions

| Topic | Decision |
|---|---|
| Language | Go, two binaries from one codebase (see §9) |
| Run mode | User-session tray app, autostart via `HKCU\...\Run` |
| Process detection | Poll process list every `poll_seconds` (default 3) |
| "Fully exited" | No matching process for `settle_seconds` (default 5) + save files stable |
| Archive | One `.zip` per backup, written to `.tmp` then renamed |
| Dedup | Skip backup when content hash equals the latest backup's hash |
| Retention | Keep newest `keep_last` (default 20) per game |
| Config | YAML, hot-reloaded; invalid edit keeps the previous config |

## 3. Configuration

Default location (portable): `games.yaml` in the folder of the running
executable, shared by `autosave.exe` and `autosave-tray.exe`; the CLI can
override it with `--config <path>`. On first run, if missing, the app writes a
commented example file there and notifies the user. Configs in older
locations are not migrated.

```yaml
backup_root: "D:/GameBackups"
poll_seconds: 3
notify: errors            # all | errors | none
defaults:
  keep_last: 20
  settle_seconds: 5
games:
  - name: Elden Ring
    processes: [eldenring.exe]
    saves: ["%APPDATA%/EldenRing"]
  - name: Hollow Knight
    processes: [hollow_knight.exe]
    saves: ["%USERPROFILE%/AppData/LocalLow/Team Cherry/Hollow Knight/user*.dat"]
    exclude: ["*.bak"]
    keep_last: 50
```

Rules:

- `name`: unique (case-insensitive); slugified for folder names
  (`Elden Ring` → `elden-ring`).
- `processes`: exe names, matched case-insensitively against process image
  name. A game is *running* while any listed process exists.
- `saves`: one or more paths. `%VAR%` and leading `~` are expanded. Glob
  characters (`*`, `?`, `[`) are allowed in any segment. A matched directory
  is included recursively; a matched file is included as is.
- `exclude`: optional globs matched against the path relative to the save base.
- Per-game `keep_last` / `settle_seconds` override `defaults`.
- Validation errors list the game name and field; app keeps running on the
  last valid config.

Hot reload: the config file's mtime is checked on every poll tick. Games that
are removed from config stop being tracked; in-flight sessions finish.

## 4. Session state machine (per game)

```
          process seen                  no process
  Idle ─────────────────► Running ─────────────────► Settling
   ▲      (pre snapshot)     ▲                          │
   │                         └──── process seen ────────┤ (no new snapshot)
   │                                                     │ settle elapsed
   │                                                     ▼
   └──────────── backup done / failed ◄────────── WaitingStable
```

- **Idle → Running**: take a `pre` snapshot. If the game was already running
  when the app started, enter Running **without** a snapshot (files may be
  mid-write) and log it.
- **Running → Settling**: last matching process disappeared; start timer.
- **Settling → Running**: a matching process reappears before the timer ends
  (launcher handoff, game self-restart). No new `pre` snapshot.
- **WaitingStable**: every 2 s, check that all save files (a) can be opened
  for reading without a sharing violation and (b) have unchanged size+mtime
  versus the previous check. When stable, take a `post` backup. Give up after
  60 s and back up anyway, logging a warning.

The state machine is pure logic driven by `(tick time, running bool)` input
so it can be unit-tested with a fake clock.

## 5. Backup

### Layout

```
<backup_root>/
  elden-ring/
    2026-10-17_21-30-05_pre.zip
    2026-10-17_23-02-41_post.zip
    .lock
```

Kinds: `pre`, `post`, `manual` (CLI/tray), `prerestore` (before a restore).

### Archive contents

- `files/<n>/<relative path>`: `<n>` is the index of the save base (see below).
- `manifest.json`: game name, kind, created-at (local time with offset),
  app version, content hash, and `bases`: list of `{index, abs_path}`.

The **save base** of a `saves` entry is its longest leading part without glob
characters, after expansion. Relative paths inside the archive are computed
from that base; restore writes back to the same absolute base.

### Algorithm

1. Acquire `<game>/.lock` (exclusive file create; stale after 10 min).
2. Expand `saves`, walk matches, apply `exclude`, sort paths.
3. No files found → skip with a warning (notify per `notify` setting).
4. Content hash = SHA-256 over each `(relpath, size, sha256(content))` in order.
5. Equal to the latest archive's hash → skip (`pre` and `post` only; `manual`
   and `prerestore` always write).
6. Write zip to `<name>.zip.tmp`, fsync, rename to `<name>.zip`.
7. Prune: delete oldest archives beyond `keep_last`; `prerestore` archives are
   never pruned automatically.
8. Release lock; log result; notify per setting.

Missing save path for one entry: warn and continue with the others.
Any write error: delete the `.tmp`, log, always notify (even if `notify: errors`).

## 6. CLI

Binary `autosave.exe` (console):

| Command | Behaviour |
|---|---|
| `autosave list` | Games in config with backup count and latest backup time |
| `autosave list <game>` | Archives of one game (time, kind, size) |
| `autosave backup <game>` | Manual backup (kind `manual`) |
| `autosave restore <game> [--at <timestamp>]` | Restore latest, or the archive matching the timestamp prefix |
| `autosave autostart on\|off` | Add/remove tray binary in `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` |
| `autosave validate` | Validate config and print expanded save paths |

`<game>` matches name or slug, case-insensitive. Restore:

1. Refuse if any of the game's processes is running.
2. Show what will be written and ask `y/N` (skip with `--yes`).
3. Take a `prerestore` backup of current files.
4. Extract archive files over their bases. Files not in the archive are left
   untouched (no deletion in MVP).

## 7. Tray app

Binary `autosave-tray.exe` (GUI subsystem, no console window).

- Single instance via a named mutex; a second launch exits silently.
- Menu: status line (`Watching N games · Running: X`), `Backup now ▸ <game>`,
  `Open backup folder`, `Edit config`, `Reload config`, `Open log`, `Quit`.
- Toast notifications via Windows toast API, filtered by `notify`.

## 8. Logging

`autosave.log` in the folder of the running executable, structured text
(`log/slog`), rotated at 5 MB, keep 3 files. Both binaries log there. Every session
state change is logged as `msg="session state" game=… from=… to=…` (plus
`stable=true|false` on entering BackingUp, false meaning the 60 s timeout).

## 9. Project layout

```
cmd/autosave/            CLI entry (console subsystem)
cmd/autosave-tray/       Tray entry (built with -H windowsgui)
internal/config/         load, validate, expand paths, detect changes
internal/process/        ProcessLister interface + gopsutil implementation
internal/session/        per-game state machine, stability check
internal/backup/         collect, hash, archive, retention, lock, restore
internal/app/            wires config + watcher loop + backup + notify
internal/tray/           systray menu
internal/notify/         toast wrapper honouring `notify`
internal/autostart/      HKCU Run key on/off
internal/logging/        slog setup + rotation
configs/games.example.yaml
scripts/check-file-length.ps1
```

Planned dependencies: `gopkg.in/yaml.v3`, `github.com/shirou/gopsutil/v4`,
`fyne.io/systray`, `git.sr.ht/~jackmordaunt/go-toast`,
`golang.org/x/sys/windows/registry`, `gopkg.in/natefinch/lumberjack.v2`.
Exact versions are pinned in `go.mod` during implementation.

## 10. Testing

- Unit: path expansion and glob bases, config validation, slugify, state
  machine (fake clock + fake process lister), hash/dedup, retention, restore
  round-trip in `t.TempDir()`.
- `scripts/check-file-length.ps1` enforces the limits in `CLAUDE.md`.
- Manual acceptance: configure Notepad (`notepad.exe`) with a temp folder as
  "save", open/close Notepad, verify `pre` and `post` archives and dedup.

## 11. Phase 2 (out of scope, for reference)

Ludusavi manifest import, Steam/Epic/GOG detection, tiered retention
(daily/weekly), restore from tray, local web UI, `--clean` restore mode.
