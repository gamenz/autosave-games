package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"time"
)

const manifestName = "manifest.json"

// BaseRef maps an archive base index back to its absolute directory.
type BaseRef struct {
	Index   int    `json:"index"`
	AbsPath string `json:"abs_path"`
}

// Manifest describes an archive; stored as manifest.json inside the zip.
type Manifest struct {
	Game       string    `json:"game"`
	Kind       Kind      `json:"kind"`
	CreatedAt  time.Time `json:"created_at"`
	AppVersion string    `json:"app_version"`
	Hash       string    `json:"hash"`
	Bases      []BaseRef `json:"bases"`
}

// ReadManifest reads manifest.json from the archive at archivePath.
func ReadManifest(archivePath string) (Manifest, error) {
	var m Manifest
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return m, fmt.Errorf("open archive: %w", err)
	}
	defer r.Close()
	f, err := r.Open(manifestName)
	if err != nil {
		return m, fmt.Errorf("archive %s: %w", archivePath, err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return m, fmt.Errorf("archive %s: decode manifest: %w", archivePath, err)
	}
	return m, nil
}

// baseRef returns the absolute base path for index n.
func (m Manifest) baseRef(n int) (string, bool) {
	for _, b := range m.Bases {
		if b.Index == n {
			return b.AbsPath, true
		}
	}
	return "", false
}
