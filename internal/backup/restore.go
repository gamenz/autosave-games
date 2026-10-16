package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"autosave-games/internal/config"
)

// RestoreTarget maps one archived file to where it will be written.
type RestoreTarget struct {
	ZipName string
	AbsPath string
}

// FindArchive picks the newest archive in dir, or the single archive whose
// name starts with atPrefix.
func FindArchive(dir, atPrefix string) (ArchiveInfo, error) {
	archives, err := List(dir)
	if err != nil {
		return ArchiveInfo{}, err
	}
	var matches []ArchiveInfo
	for _, a := range archives {
		if strings.HasPrefix(a.Name, atPrefix) {
			matches = append(matches, a)
		}
	}
	switch {
	case len(matches) == 0:
		return ArchiveInfo{}, fmt.Errorf("no archive matches %q", atPrefix)
	case atPrefix == "" || len(matches) == 1:
		return matches[0], nil
	}
	return ArchiveInfo{}, fmt.Errorf("ambiguous %q: %d archives match", atPrefix, len(matches))
}

// PlanRestore validates every archived file and returns its target path.
func PlanRestore(archivePath string) ([]RestoreTarget, error) {
	m, err := ReadManifest(archivePath)
	if err != nil {
		return nil, err
	}
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()
	var targets []RestoreTarget
	for _, f := range r.File {
		if f.Name == manifestName || strings.HasSuffix(f.Name, "/") {
			continue
		}
		target, err := targetPath(m, f.Name)
		if err != nil {
			return nil, err
		}
		targets = append(targets, RestoreTarget{ZipName: f.Name, AbsPath: target})
	}
	return targets, nil
}

// Restore writes every archived file back to its save base. Files that are
// not in the archive are left untouched.
func Restore(archivePath string) ([]string, error) {
	targets, err := PlanRestore(archivePath)
	if err != nil {
		return nil, err
	}
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()
	var written []string
	for _, t := range targets {
		if err := extract(r, t); err != nil {
			return written, err
		}
		written = append(written, t.AbsPath)
	}
	return written, nil
}

func targetPath(m Manifest, zipName string) (string, error) {
	rest, ok := strings.CutPrefix(zipName, "files/")
	idx, rel, ok2 := strings.Cut(rest, "/")
	n, err := strconv.Atoi(idx)
	if !ok || !ok2 || err != nil || rel == "" {
		return "", fmt.Errorf("archive entry %q: unexpected layout", zipName)
	}
	base, found := m.baseRef(n)
	if !found {
		return "", fmt.Errorf("archive entry %q: unknown save base %d", zipName, n)
	}
	native := filepath.FromSlash(rel)
	if filepath.IsAbs(native) || filepath.VolumeName(native) != "" {
		return "", fmt.Errorf("archive entry %q: absolute path", zipName)
	}
	target := filepath.Join(base, native)
	if !config.IsWithin(target, base) || filepath.Clean(target) == filepath.Clean(base) {
		return "", fmt.Errorf("archive entry %q: escapes save base", zipName)
	}
	return target, nil
}

func extract(r *zip.ReadCloser, t RestoreTarget) error {
	src, err := r.Open(t.ZipName)
	if err != nil {
		return fmt.Errorf("read %s: %w", t.ZipName, err)
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(t.AbsPath), 0o755); err != nil {
		return fmt.Errorf("restore %s: %w", t.AbsPath, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(t.AbsPath), ".restore-*")
	if err != nil {
		return fmt.Errorf("restore %s: %w", t.AbsPath, err)
	}
	_, copyErr := io.Copy(tmp, src)
	closeErr := tmp.Close()
	if copyErr == nil && closeErr == nil {
		if err := os.Rename(tmp.Name(), t.AbsPath); err == nil {
			return nil
		} else {
			copyErr = err
		}
	}
	os.Remove(tmp.Name())
	if copyErr == nil {
		copyErr = closeErr
	}
	return fmt.Errorf("restore %s: %w", t.AbsPath, copyErr)
}
