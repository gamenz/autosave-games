package backup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireLock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "game")
	now := time.Now()
	release, err := AcquireLock(dir, now)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := AcquireLock(dir, now); !errors.Is(err, ErrLocked) {
		t.Fatalf("second acquire err = %v, want ErrLocked", err)
	}
	release()
	release2, err := AcquireLock(dir, now)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	release2()
}

func TestAcquireLockStale(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(dir, ".lock")
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	old := now.Add(-11 * time.Minute)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	release, err := AcquireLock(dir, now)
	if err != nil {
		t.Fatalf("stale lock not taken over: %v", err)
	}
	release()
}
