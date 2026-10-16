package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

// WriteArchive writes entries and the manifest to path atomically:
// everything goes to path+".tmp", which is synced and then renamed.
func WriteArchive(path string, entries []Entry, m Manifest) (err error) {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(tmp)
		}
	}()

	zw := zip.NewWriter(f)
	for _, e := range entries {
		if err = addFile(zw, e); err != nil {
			return err
		}
	}
	w, err := zw.Create(manifestName)
	if err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err = enc.Encode(m); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err = zw.Close(); err != nil {
		return fmt.Errorf("finish archive: %w", err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("sync archive: %w", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename archive: %w", err)
	}
	return nil
}

func zipEntryName(e Entry) string {
	return "files/" + strconv.Itoa(e.BaseIndex) + "/" + e.RelPath
}

func addFile(zw *zip.Writer, e Entry) error {
	src, err := os.Open(e.AbsPath)
	if err != nil {
		return fmt.Errorf("read save file: %w", err)
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("read save file: %w", err)
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("zip header %s: %w", e.RelPath, err)
	}
	hdr.Name, hdr.Method = zipEntryName(e), zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("zip entry %s: %w", e.RelPath, err)
	}
	if _, err := io.Copy(w, src); err != nil {
		return fmt.Errorf("zip entry %s: %w", e.RelPath, err)
	}
	return nil
}
