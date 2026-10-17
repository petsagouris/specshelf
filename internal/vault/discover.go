package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

// projectMarker is the directory name that marks a project vault.
const projectMarker = ".specshelf"

// Discover returns the project vault for dir: the first directory at or above
// dir that holds a .specshelf directory, walking up and stopping at home or the
// filesystem root, whichever comes first. No marker is a NoProjectVaultError
// naming dir — discovery never falls back to another vault. home may be empty,
// which removes the home bound and leaves only the filesystem root.
func Discover(dir, home string) (string, error) {
	start := dir
	if !filepath.IsAbs(start) {
		abs, err := filepath.Abs(start)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", dir, err)
		}
		start = abs
	}
	start = filepath.Clean(start)
	if home != "" {
		home = filepath.Clean(home)
	}

	for level := start; ; {
		candidate := filepath.Join(level, projectMarker)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		if home != "" && level == home {
			break
		}
		parent := filepath.Dir(level)
		if parent == level {
			break
		}
		level = parent
	}
	return "", &NoProjectVaultError{Start: dir}
}
