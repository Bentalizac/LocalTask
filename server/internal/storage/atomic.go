// Package storage provides low-level, filesystem-safe persistence helpers.
package storage

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path without ever leaving a partially
// written file in its place: it writes to a temp file in the same
// directory, fsyncs it, then renames it over path. The rename is atomic
// on the filesystems this application targets (local disk).
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
