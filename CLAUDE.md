# autosave-games

Windows tray app (Go) that watches game processes and backs up save files
when a game starts and after it fully exits.

## Docs layout

- Specs: `docs/specs/YYYY-MM-DD-<topic>.md` (committed).
- Plans: `docs/plans/*` (git-ignored, local only). Name:
  `docs/plans/YYYY-MM-DD-<topic>.md`; a plan over 250 lines is split into an
  index file plus parts `docs/plans/YYYY-MM-DD-<topic>-NN-<part>.md`.
- Superpowers skills MUST use these paths, never `docs/superpowers/...`:
  brainstorming → `docs/specs/`, writing-plans → `docs/plans/`.
- Plans are executed natively with `superpowers:executing-plans` (no
  subagents); plan "Commit" steps run only when the user explicitly asks.
- **Never link to or reference any path under `docs/plans/`** from specs,
  code, comments, README or this file. Plans may link to specs, not the reverse.

## File size limits

| Kind         | Soft  | Hard (fail) |
|--------------|-------|-------------|
| Go source    | 200   | 300 lines   |
| Go test      | 300   | 400 lines   |
| Function     | 40    | 60 lines    |
| Markdown doc | 150   | 250 lines   |
| CLAUDE.md    | 100   | 150 lines   |

Split by responsibility before hitting the hard limit. No `utils/` or
`helpers/` packages; name files after what they do.

## Conventions

- Code identifiers and comments in English. LF line endings.
- Tests live next to code (`foo.go` + `foo_test.go`).
