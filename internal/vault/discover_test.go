package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscover(t *testing.T) {
	t.Run("nested working directory", func(t *testing.T) {
		home := t.TempDir()
		project := filepath.Join(home, "work", "project")
		marker := filepath.Join(project, ".specshelf")
		if err := os.MkdirAll(marker, 0o755); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(project, "internal", "vault")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		got, err := Discover(dir, home)
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if got != marker {
			t.Errorf("Discover = %q, want %q", got, marker)
		}
	})

	t.Run("no project vault", func(t *testing.T) {
		home := t.TempDir()
		dir := filepath.Join(home, "a", "b")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		_, err := Discover(dir, home)
		var absent *NoProjectVaultError
		if !errors.As(err, &absent) {
			t.Fatalf("Discover error = %v, want *NoProjectVaultError", err)
		}
		if absent.Start != dir {
			t.Errorf("Start = %q, want %q", absent.Start, dir)
		}
		if !strings.Contains(err.Error(), dir) {
			t.Errorf("error %q must name the searched directory %q", err.Error(), dir)
		}
	})

	t.Run("the search does not escape the home directory", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".specshelf"), 0o755); err != nil {
			t.Fatal(err)
		}
		home := filepath.Join(root, "home")
		dir := filepath.Join(home, "a")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		_, err := Discover(dir, home)
		var absent *NoProjectVaultError
		if !errors.As(err, &absent) {
			t.Fatalf("Discover error = %v, want *NoProjectVaultError", err)
		}
	})

	t.Run("a marker in the home directory itself is found", func(t *testing.T) {
		root := t.TempDir()
		home := filepath.Join(root, "home")
		marker := filepath.Join(home, ".specshelf")
		if err := os.MkdirAll(marker, 0o755); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(home, "a", "b")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		got, err := Discover(dir, home)
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if got != marker {
			t.Errorf("Discover = %q, want %q", got, marker)
		}
	})
}
