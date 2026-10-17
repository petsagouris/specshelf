package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// TestMain builds the real command binary and a fake editor, and puts both on
// PATH so the command scripts run `exec specshelf` against the built binary —
// exercising cmd/specshelf's exit wiring rather than the test binary — and set
// $VISUAL to the cross-platform fakeeditor.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "specshelf-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create binary directory:", err)
		os.Exit(1)
	}
	exe := func(name string) string {
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		return filepath.Join(dir, name)
	}
	for name, pkg := range map[string]string{
		"specshelf":  "github.com/petsagouris/specshelf/cmd/specshelf",
		"fakeeditor": "./testdata/fakeeditor",
	} {
		build := exec.Command("go", "build", "-o", exe(name), pkg)
		if out, err := build.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "build %s: %v\n%s", name, err, out)
			os.Exit(1)
		}
	}
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
		fmt.Fprintln(os.Stderr, "set PATH:", err)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// TestScript runs the per-command testscript files against the built binary. Each
// script uses a vault under $WORK and points SPECSHELF_CONFIG at a file that does
// not exist, so no real configuration or vault is ever read.
func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
		Setup: func(env *testscript.Env) error {
			env.Setenv("SPECSHELF_CONFIG", filepath.Join(env.WorkDir, "config.toml"))
			return nil
		},
	})
}
