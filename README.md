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
Keep both files in the same folder. The app is portable: its config and log
files live in that folder too, so it must be writable (not `C:\Program Files`).

## Run

1. Start `bin\autosave-tray.exe`. On first run it writes an example config
   `games.yaml` next to the exe and shows a notification.
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
- Optional `install_dir`: the game counts as running while **any** exe inside
  that folder runs (`processes` is then only a fallback). `steam_appid` is
  set by the Steam scan to recognise the game later.

## Steam scan

Tray menu → **Scan Steam games** adds every installed Steam game that is not
in the config yet:

- name, `steam_appid`, `install_dir` and exe names come from Steam's
  `libraryfolders.vdf` / `appmanifest_*.acf`;
- save folders come from the Steam Cloud cache
  (`userdata\<account>\<appid>\remotecache.vdf`, all accounts);
- games without cloud data are appended **commented out** at the end of the
  file: fill in `saves`, then remove the leading `# ` of their lines;
- existing games are never changed (matched by appid, install dir or name);
  scanning again adds nothing new;
- the previous file is kept as `games.yaml.bak`. Comments are preserved, blank
  lines and comment alignment may be normalised.

A toast reports `added N games, M need save paths, K already configured`.

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
| Config | `games.yaml` next to the exe |
| Log | `autosave.log` next to the exe (5 MB, plus up to 3 rotated `autosave-<time>.log`) |
| Backups | `<backup_root>\<game-slug>\` |

## Development

```powershell
go vet ./...
go test ./...
powershell -NoProfile -File scripts\check-file-length.ps1
```

See [docs/architecture.md](docs/architecture.md) and the spec in
[docs/specs/](docs/specs/).
