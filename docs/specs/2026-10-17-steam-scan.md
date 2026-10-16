# Spec: Steam library scan (tray)

- Status: Draft (awaiting review)
- Date: 2026-10-17
- Builds on: [2026-10-17-autosave-games-mvp](2026-10-17-autosave-games-mvp.md)

## 1. Goal

One tray click adds every installed Steam game to `games.yaml`, with its
install folder, process names and save folders filled in from Steam's own
files. Games whose save folder cannot be found are added as commented
entries for the user to complete.

### Non-goals

- No CLI command (tray only).
- No Ludusavi manifest, no Epic/GOG (phase 2).
- Never edits or removes games already in the config.
- No network access.

## 2. User flow

1. Tray menu → **Scan Steam games** (new item, above *Reload config*).
2. The item is disabled while the scan runs on a background goroutine.
3. The config file is updated, then the existing hot reload picks it up.
4. A toast is **always** shown, whatever `notify` says (user-initiated):
   `Steam scan: added 12 games, 7 need save paths, 3 already configured`,
   or the error (`Steam not found`, `config is invalid, fix it first`).

## 3. Discovery

| Step | Source | Notes |
|---|---|---|
| Steam root | `HKCU\Software\Valve\Steam` → `SteamPath`; fallback `HKLM\SOFTWARE\WOW6432Node\Valve\Steam` → `InstallPath` | missing → error |
| Libraries | `<root>\steamapps\libraryfolders.vdf` → every `"path"` | always include `<root>` itself |
| Apps | `<lib>\steamapps\appmanifest_*.acf` → `appid`, `name`, `installdir` | install dir = `<lib>\steamapps\common\<installdir>`; skip if missing |
| Saves | `<root>\userdata\*\<appid>\remotecache.vdf`, every account | see §4 |
| Processes | `.exe` files under the install dir | see §5 |

VDF/ACF are Valve KeyValues text files: quoted keys and values, nested
`{ }` blocks, `\\` and `\"` escapes, `//` comments. A small parser in
`internal/steam` handles them; unknown keys are ignored.

### Exclusions and grouping

- Skip appid `228980` (Steamworks Common Redistributables) and apps whose
  install dir contains no executable after §5 filtering (soundtracks, art
  books, bonus editions, tools without exe).
- Apps sharing one install dir (Half-Life 2 + Lost Coast + Episodes) become
  **one** game: name and `steam_appid` from the lowest appid; saves are the
  union of all those apps.

## 4. Save folders from Steam Cloud cache

Each `remotecache.vdf` lists synced files as `"<relative path>" { "root" "<n>" … }`.

| root | Base folder | Verified on dev machine |
|---|---|---|
| 0 | `<steam root>\userdata\<account>\<appid>\remote` | Rise of the Tomb Raider, PRAGMATA |
| 1 | game install dir | Half-Life 2 |
| 2 | `%USERPROFILE%\Documents` | assumed |
| 3 | `%LOCALAPPDATA%` | Expedition 33, Stellar Blade |
| 4 | `%APPDATA%` | Elden Ring, Sekiro |

Other root values: logged and skipped.

Deriving `saves` per app:

- **root 0** → one glob entry covering every account:
  `<steam root>\userdata\*\<appid>\remote`.
- **other roots**, per root: take the directory of every listed file across
  all accounts, compute their common directory, then cut it at the first
  segment that looks like a Steam ID (`7656119` + 10 digits) or equals a
  `userdata` account folder name. Example: `EldenRing/76561199478888042/ER0000.sl2`
  from two accounts → `%APPDATA%/EldenRing`;
  `KoeiTecmo/FatalFrameII/Savedata/<account>/SLOT00/…` → `…/Savedata`.
- **Safety:** if the result would be the root base itself (files directly in
  `%APPDATA%`, `%LOCALAPPDATA%`, Documents or the install dir), list each
  file individually instead of the folder.
- Paths are written with `%APPDATA%` / `%LOCALAPPDATA%` / `%USERPROFILE%`
  variables when they start with those folders, otherwise absolute.

No cache entry for an app → "needs save path" (§6).

## 5. Process detection

### Config fields (new, optional)

```yaml
  - name: ELDEN RING
    steam_appid: 1245620
    install_dir: 'D:\Steam\steamapps\common\ELDEN RING'
    processes: [eldenring.exe, start_protected_game.exe, modengine2_launcher.exe]
    saves: ['%APPDATA%/EldenRing']
```

- `install_dir` (expanded like saves): the game is **running when any
  running process's executable path is inside it** (case-insensitive,
  segment-aware, `config.IsWithin`).
- `processes` stays as fallback: a name match also counts as running. This
  covers processes whose path cannot be read (e.g. anti-cheat protected).
- Validation: a game needs `processes` **or** `install_dir` (was: processes).
- `steam_appid`: used only to recognise games on later scans.

### Lister change

`process.Lister.Names()` becomes `Snapshot() (Snapshot, error)` with
`Names map[string]bool` and `Paths []string` (lower-cased full exe paths;
processes whose path cannot be read are omitted from `Paths` only). Budget:
one snapshot ≤ 30 ms for ~300 processes (today's names-only listing: ~6 ms).
First implementation step: confirm the path of `eldenring.exe` (Easy
Anti-Cheat) is readable; if not, the name fallback covers it.

### Process names written at scan

Base names of `.exe` files under the install dir, depth ≤ 4, lower-cased,
deduplicated, at most 10. Excluded (case-insensitive globs):
`*setup*`, `*redist*`, `*prereq*`, `vc_redist*`, `dxsetup*`, `dotnet*`,
`unins*`, `crashpad_handler*`, `unitycrashhandler*`, `*crashreport*`,
`*crashhandler*`, `*bugreport*`.

## 6. Writing games.yaml

- Refuse to write when the current file fails `config.Load` (toast).
- Back up the current file to `games.yaml.bak`, then write via `.tmp` + rename.
- Edit through `yaml.v3` `Node` so user comments survive; a flow `games: []`
  becomes a block sequence. Formatting of the rest of the file may be
  normalised (indent 2); content and comments are kept.
- A Steam app is **already configured** when an existing game has the same
  `steam_appid`, the same `install_dir`, or the same slug — then it is skipped.
- New games with saves → appended as normal entries.
- New games without saves → appended as a commented block at the end of
  `games`, containing the marker `steam_appid <id>`:

  ```yaml
  # - name: Cities: Skylines   # steam_appid 255710 — save path unknown, fill in "saves"
  #   install_dir: 'D:\Steam\steamapps\common\Cities_Skylines'
  #   processes: [cities.exe]
  #   saves: []
  ```
  A later scan skips apps whose marker already appears in the file.
- Scanning twice in a row changes nothing (idempotent).

## 7. Logging

One line per app: `msg="steam scan" appid=… name=… result=added|needs-saves|configured|skipped reason=…`,
plus a summary line. Unknown cloud roots and unreadable files are warnings.

## 8. Layout

```
internal/steam/      locate.go (registry), vdf.go, library.go (libraries +
                     manifests), cloud.go (remotecache → saves), exes.go,
                     scan.go (Scan(root) → []Found, grouping, exclusions)
internal/config/     schema/validate (new fields), merge.go (Node edit + commented blocks)
internal/process/    Snapshot with paths
internal/app/        running = name match OR path inside install_dir; ScanSteam()
internal/tray/       "Scan Steam games" item
```

`steam.Scan` takes the Steam root and an `EnvFunc`, so tests run on a fake
Steam tree in `testdata/` without registry access.

## 9. Testing

- VDF parser: nesting, escapes, comments, malformed input → error.
- Library + manifests on a fixture tree (2 libraries, missing install dir,
  redistributables app, shared install dir).
- Cloud saves: roots 0/1/3/4, unknown root, two accounts with different
  Steam IDs → one folder, files directly under a root → per-file entries.
- Exe filter: excluded patterns, depth, cap.
- Merge: comments preserved, `games: []` converted, existing game kept,
  dedup by appid / install_dir / slug, commented block + marker skip,
  idempotent second run, invalid config refused, `.bak` written.
- Process: running by path inside `install_dir`, by name fallback, neither.
- Manual: scan on the dev machine → expect 12 games with saves and the
  others as commented blocks, Elden Ring detected while
  running, toast shown with `notify: errors`.

## 10. Phase 2 (out of scope)

Ludusavi manifest for games without cloud data, Epic/GOG libraries, tray
**Restore** and **Start with Windows** (would make the CLI optional).
