package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// editorCommand honours the precedence the CLI spec and docs/CONFIG.md state:
// $VISUAL, then $EDITOR, then the configuration file's editor setting.
func TestEditorCommandResolution(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeConfig := func(contents string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeConfig("editor = \"nano -w\"\n")
	t.Setenv("SPECSHELF_CONFIG", configPath)

	join := func(t *testing.T) string {
		t.Helper()
		got, err := editorCommand()
		if err != nil {
			t.Fatalf("editorCommand: %v", err)
		}
		return strings.Join(got, " ")
	}

	t.Run("configured editor, split so its flags survive", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "")
		if got := join(t); got != "nano -w" {
			t.Errorf("editorCommand = %q, want %q", got, "nano -w")
		}
	})

	t.Run("EDITOR beats the configured editor", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "vi")
		if got := join(t); got != "vi" {
			t.Errorf("editorCommand = %q, want %q", got, "vi")
		}
	})

	t.Run("VISUAL beats EDITOR", func(t *testing.T) {
		t.Setenv("VISUAL", "emacs")
		t.Setenv("EDITOR", "vi")
		if got := join(t); got != "emacs" {
			t.Errorf("editorCommand = %q, want %q", got, "emacs")
		}
	})

	t.Run("nothing set is a runtime error", func(t *testing.T) {
		t.Setenv("VISUAL", "")
		t.Setenv("EDITOR", "")
		writeConfig("# no editor setting\n")
		if _, err := editorCommand(); err == nil {
			t.Error("editorCommand succeeded with no editor configured")
		}
	})
}
