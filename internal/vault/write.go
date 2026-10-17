package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// renameOver renames a written temporary file over its destination. It is a
// variable so a test can simulate the interruption between the write and the
// rename, which an ordinary filesystem will not produce on demand.
var renameOver = os.Rename

// WriteDocument writes data to path without ever leaving it half-written: the
// bytes go to a temporary file in path's directory, are flushed to durable
// storage, and the temporary file is renamed over path. A failure before the
// rename leaves path as it was, or absent, and removes the temporary file.
func WriteDocument(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		return fmt.Errorf("create temporary file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		return failWrite(tmp, tmpName, fmt.Errorf("write %s: %w", tmpName, err))
	}
	if err := tmp.Sync(); err != nil {
		return failWrite(tmp, tmpName, fmt.Errorf("flush %s: %w", tmpName, err))
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close %s: %w", tmpName, err)
	}
	if err := renameOver(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename %s to %s: %w", tmpName, path, err)
	}
	return nil
}

// CreateDocument creates a new document at path, or at the next free "-2", "-3"
// … name when that one is taken, and returns the path it actually created. The
// create is exclusive, so an existing file is never truncated and two
// concurrent creations of the same title cannot overwrite each other: when a
// concurrent creator takes a name first, the next free name is tried.
func CreateDocument(path string, data []byte) (string, error) {
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	candidate := nextFreePath(path)
	for n := 2; ; n++ {
		if err := writeExclusive(candidate, data); err != nil {
			if !errors.Is(err, fs.ErrExist) {
				return "", err
			}
			candidate = stem + "-" + strconv.Itoa(n) + ext
			continue
		}
		return candidate, nil
	}
}

// writeExclusive creates path with data in one exclusive step: an existing path,
// including one that appears between the caller's check and this call, is an
// error wrapping fs.ErrExist rather than a file to overwrite.
func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		return failWrite(f, path, fmt.Errorf("write %s: %w", path, err))
	}
	if err := f.Sync(); err != nil {
		return failWrite(f, path, fmt.Errorf("flush %s: %w", path, err))
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}

// failWrite closes and removes a file that a write failed on, and returns err:
// an interrupted write leaves nothing behind rather than a partial file.
func failWrite(f *os.File, name string, err error) error {
	_ = f.Close()
	_ = os.Remove(name)
	return err
}
