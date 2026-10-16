# autosave-games

A small Windows tray app that watches your games and backs up their save files
automatically:

- a **pre** snapshot when a game starts,
- a **post** backup once the game has fully exited and its save files stopped changing,
- unchanged saves are skipped (content hash), old archives are pruned.

A CLI lists, backs up and restores archives.

## Build

Requires Go ≥ 1.25.

```powershell
powershell -NoProfile -File scripts\build.ps1
```

Produces `bin\autosave.exe` (CLI) and `bin\autosave-tray.exe` (tray, no console).
Keep both files in the same folder.

## Run

1. Start `bin\autosave-tray.exe`. On first run it writes an example config to
   `%APPDATA%\autosave-games\games.yaml` and shows a notification.
2. Tray menu → **Edit config**, add your games, save. The file is reloaded
   automatically (or use **Reload config**).
3. Optional: start with Windows — `bin\autosave.exe autostart on`
   (`off` to remove). Uses `HKCU\...\Run`, no admin rights.

## Configuration

```yaml
backup_root: "D:/GameBackups"   # archives: <backup_root>/<game-slug>/<time>_<kind>.zip
poll_seconds: 3
notify: errors                  # all | errors | none (write failures always show)
defaults:
  keep_last: 20                 # per game; prerestore archives are never pruned
  settle_seconds: 5             # game must stay closed this long before the post backup
games:
  - name: Elden Ring
    processes: [eldenring.exe]  # any of these running = game running
    saves: ["%APPDATA%/EldenRing"]
  - name: Hollow Knight
    processes: [hollow_knight.exe]
    saves: ["%USERPROFILE%/AppData/LocalLow/Team Cherry/Hollow Knight/user*.dat"]
    exclude: ["*.bak"]
    keep_last: 50
```

- Paths accept `%VAR%`, a leading `~` and globs (`*`, `?`, `[...]`). Folders are
  backed up recursively.
- Inside double quotes YAML treats `\` as an escape: use `/` or single quotes.
- `backup_root` must not be inside a save folder (and vice versa).
- Check a config and see the expanded paths: `autosave validate`.

## CLI

```text
autosave [--config path] list [game]
autosave [--config path] backup <game>
autosave [--config path] restore <game> [--at <prefix>] [--yes]
autosave [--config path] validate
autosave autostart on|off
```

`<game>` is the name or slug (`elden-ring`), case-insensitive.

`restore` refuses while the game is running, shows the files it will write,
asks for confirmation, takes a `prerestore` backup of the current saves, then
restores the latest archive (or the one whose name starts with `--at`).
Files that are not in the archive are left untouched.

## Files

| What | Where |
|---|---|
| Config | `%APPDATA%\autosave-games\games.yaml` |
| Log | `%LOCALAPPDATA%\autosave-games\logs\autosave.log` (5 MB × 3) |
| Backups | `<backup_root>\<game-slug>\` |

## Development

```powershell
go vet ./...
go test ./...
powershell -NoProfile -File scripts\check-file-length.ps1
```

See [docs/architecture.md](docs/architecture.md) and the spec in
[docs/specs/](docs/specs/).
