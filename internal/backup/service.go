package backup

import (
	"path/filepath"
	"time"

	"autosave-games/internal/config"
)

// Reasons reported in Result when a backup is skipped.
const (
	ReasonUnchanged = "unchanged"
	ReasonNoFiles   = "no save files found"
)

// Service runs backups into Root/<game slug>/.
type Service struct {
	Root       string
	AppVersion string
	Now        func() time.Time
}

// Result is the outcome of one Run.
type Result struct {
	Path     string
	Skipped  bool
	Reason   string
	Warnings []error
}

// GameDir is the backup folder of g.
func (s *Service) GameDir(g config.Game) string { return filepath.Join(s.Root, g.Slug) }

// Files lists the current save files of g (used for stability checks).
func (s *Service) Files(g config.Game) []string {
	entries, _ := Collect(config.ResolveSaves(g.Saves), g.Exclude)
	return Paths(entries)
}

// Run backs up g once (spec §5). pre/post archives are skipped when the
// content equals the latest archive; manual and prerestore always write.
func (s *Service) Run(g config.Game, kind Kind) (Result, error) {
	var res Result
	dir := s.GameDir(g)
	// Lock staleness compares against file mtimes, so it needs the wall clock.
	release, err := AcquireLock(dir, time.Now())
	if err != nil {
		return res, err
	}
	defer release()

	bases := NormalizeBases(config.ResolveSaves(g.Saves))
	entries, warnings := Collect(bases, g.Exclude)
	res.Warnings = warnings
	if len(entries) == 0 {
		res.Skipped, res.Reason = true, ReasonNoFiles
		return res, nil
	}
	hash, err := ContentHash(entries)
	if err != nil {
		return res, err
	}
	if (kind == KindPre || kind == KindPost) && s.latestHash(dir) == hash {
		res.Skipped, res.Reason = true, ReasonUnchanged
		return res, nil
	}

	now := s.Now()
	m := Manifest{Game: g.Name, Kind: kind, CreatedAt: now, AppVersion: s.AppVersion, Hash: hash}
	for _, b := range bases {
		m.Bases = append(m.Bases, BaseRef{Index: b.Index, AbsPath: b.AbsBase})
	}
	res.Path = UniqueArchivePath(dir, now, kind)
	if err := WriteArchive(res.Path, entries, m); err != nil {
		return res, err
	}
	if _, err := Prune(dir, g.KeepLast); err != nil {
		res.Warnings = append(res.Warnings, err)
	}
	return res, nil
}

func (s *Service) latestHash(dir string) string {
	archives, err := List(dir)
	if err != nil || len(archives) == 0 {
		return ""
	}
	m, err := ReadManifest(archives[0].Path)
	if err != nil {
		return ""
	}
	return m.Hash
}
