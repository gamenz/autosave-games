package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ContentHash fingerprints the save set: base index, relative path, size
// and content of every entry, in order. The format is fixed so hashes stay
// comparable across app versions.
func ContentHash(entries []Entry) (string, error) {
	h := sha256.New()
	for _, e := range entries {
		sum, err := fileHash(e.AbsPath)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%d\x00%s\x00%d\x00%x\n", e.BaseIndex, e.RelPath, e.Size, sum)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fileHash(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("hash save file: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, fmt.Errorf("hash save file: %w", err)
	}
	return h.Sum(nil), nil
}
