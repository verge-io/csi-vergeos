package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirectory_Creates(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new-dir")

	err := EnsureDirectory(dir)
	if err != nil {
		t.Fatalf("EnsureDirectory failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

func TestEnsureDirectory_AlreadyExists(t *testing.T) {
	dir := t.TempDir() // already exists

	err := EnsureDirectory(dir)
	if err != nil {
		t.Fatalf("EnsureDirectory on existing dir should succeed: %v", err)
	}
}

func TestCleanupMountPoint_NonExistent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")

	err := CleanupMountPoint(dir)
	if err != nil {
		t.Fatalf("CleanupMountPoint on non-existent dir should succeed: %v", err)
	}
}
