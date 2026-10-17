package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/petsagouris/specshelf/internal/docs"
)

// initVault creates the vault directories and the built-in README and
// templates. Each file is written only when it is missing, so user edits to a
// template or the README survive.
func initVault(root string) error {
	for _, dir := range []string{"specs", "decisions", "templates"} {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
	}

	files := []struct {
		path string
		data string
	}{
		{filepath.Join(root, "templates", "spec.md"), docs.SpecTemplate},
		{filepath.Join(root, "templates", "decision.md"), docs.DecisionTemplate},
		{filepath.Join(root, "README.md"), docs.VaultReadme},
	}
	for _, f := range files {
		if err := writeIfMissing(f.path, f.data); err != nil {
			return err
		}
	}
	return nil
}

// writeIfMissing creates path with data, leaving an existing file untouched.
func writeIfMissing(path, data string) error {
	// O_EXCL is what makes an existing file — including a user-edited one — win.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if _, err := f.WriteString(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
