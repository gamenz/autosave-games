# Architecture

Spec: [specs/2026-10-17-autosave-games-mvp.md](specs/2026-10-17-autosave-games-mvp.md)

## Packages

| Package | Responsibility |
|---|---|
| `cmd/autosave` | CLI entry: logging + `cli.Run` |
| `cmd/autosave-tray` | Tray entry: single instance, `app.App` loop, tray menu |
| `configs` | Embedded example `games.yaml` |
| `internal/config` | Parse, defaults, `%VAR%`/`~` expansion, validation, save bases, file watcher, comment-preserving `AddGames` |
| `internal/process` | `Lister.Snapshot()` (exe names + paths, gopsutil) + `Fake`; running = name match or exe inside `install_dir` |
| `internal/steam` | Steam root (registry), VDF parser, libraries/manifests, cloud-cache save folders, exe filter, `Scan` |
| `internal/session` | Pure per-game state machine; save-file stability check |
| `internal/backup` | Collect → hash → zip (tmp + rename) → prune; lock; list; restore |
| `internal/notify` | Level-filtered notifications (toast) + logging of every event |
| `internal/logging` | slog text handler on a rotating file |
| `internal/appdir` | Folder of the running exe: home of `games.yaml` and `autosave.log` (portable) |
| `internal/app` | Watcher loop: ticks, runners, worker goroutines, hot reload |
| `internal/cli` | Subcommands `list`, `backup`, `restore`, `validate`, `autostart` |
| `internal/autostart` | `HKCU\...\Run` value |
| `internal/tray` | systray menu, named mutex, embedded icon |

Dependencies point downward: `cmd → app/cli/tray → backup/session/notify →
config/process`. `session` has no I/O except `stable.go`.

## Loop

Each `App.Tick` (every `poll_seconds`, or every 2 s while a game waits for
its files to settle):

1. drain finished backup workers (notify, `BackupDone`, drop removed runners),
2. reload config if `games.yaml` changed (invalid → keep old, notify),
3. list processes once — on error skip stepping (never treated as "exited"),
4. step each runner's state machine and start backup workers for its actions.

Workers run `backup.Service.Run` on a copy of the service, so a reload never
races with an in-flight backup. Only `Tick` mutates runners (under `App.mu`).

## Session state machine

| From | Condition | To | Action |
|---|---|---|---|
| Idle | running | Running | pre snapshot |
| Running | not running | Settling | — |
| Settling | running again | Running | — (no new pre) |
| Settling | `settle_seconds` elapsed | WaitingStable | — |
| WaitingStable | running again | Running | — |
| WaitingStable | files stable, or 60 s | BackingUp | post backup |
| BackingUp | worker finished | Idle | — |

A game already running when the app starts begins in Running without a pre
snapshot. "Stable" = two consecutive checks with equal size + mtime and every
file openable for reading.

## Archive format

`<backup_root>/<slug>/2006-01-02_15-04-05_<kind>[_n].zip`, kinds `pre`,
`post`, `manual`, `prerestore`. Inside: `files/<base index>/<relative path>`
and `manifest.json` (game, kind, created_at, app_version, content hash, base
index → absolute path). Restore maps entries back through the manifest and
rejects paths escaping their base.
