package vault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/petsagouris/specshelf/internal/docs"
)

func TestInitVault(t *testing.T) {
	t.Run("fresh directory", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}

		want := map[string]string{
			"templates/spec.md":     docs.SpecTemplate,
			"templates/decision.md": docs.DecisionTemplate,
			"README.md":             docs.VaultReadme,
		}
		for name, wantData := range want {
			got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			if string(got) != wantData {
				t.Errorf("%s = %q, want %q", name, got, wantData)
			}
		}
		for _, dir := range []string{"specs", "decisions", "templates"} {
			info, err := os.Stat(filepath.Join(root, dir))
			if err != nil {
				t.Fatalf("stat %s: %v", dir, err)
			}
			if !info.IsDir() {
				t.Errorf("%s is not a directory", dir)
			}
		}
	})

	t.Run("edited template survives", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
			t.Fatal(err)
		}
		const edited = "my own template\n"
		path := filepath.Join(root, "templates", "spec.md")
		if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != edited {
			t.Errorf("edited template = %q, want %q", got, edited)
		}
	})

	t.Run("already initialised is a no-op", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("first initVault: %v", err)
		}
		const edited = "custom readme\n"
		path := filepath.Join(root, "README.md")
		if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := initVault(root); err != nil {
			t.Fatalf("second initVault: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != edited {
			t.Errorf("README after second init = %q, want %q", got, edited)
		}
	})
}
