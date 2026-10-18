package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"exitcode": cmdExitCode,
			"cmppath":  cmdCmpPath,
		},
	})
}

// cmdExitCode runs a command and asserts its exact exit status, which the
// builtin exec cannot do: it only checks success or failure. `exitcode 2
// specshelf ...` makes the 0/1/2 contract visible in the script. The output is
// captured through Exec, so a following stdout or stderr assertion sees it.
func cmdExitCode(ts *testscript.TestScript, _ bool, args []string) {
	if len(args) < 2 {
		ts.Fatalf("usage: exitcode <code> <command> [args...]")
	}
	want, err := strconv.Atoi(args[0])
	if err != nil {
		ts.Fatalf("exitcode: %v", err)
	}
	runErr := ts.Exec(args[1], args[2:]...)
	got := 0
	if runErr != nil {
		var exit *exec.ExitError
		if !errors.As(runErr, &exit) {
			ts.Fatalf("exitcode: %v", runErr)
		}
		got = exit.ExitCode()
	}
	if got != want {
		ts.Fatalf("exit code = %d, want %d", got, want)
	}
}

// cmdCmpPath compares two files exactly as cmpenv does — environment expansion
// applied to the recorded file — but normalises filesystem separators on both
// sides first, so a recorded path written with "/" matches the native separator
// the tool prints on Windows. Runs of backslashes collapse to one "/" as well,
// because encoding/json escapes a path's separators ("C:\\dir"), so path --json
// needs the same treatment as a bare path.
func cmdCmpPath(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("usage: cmppath file1 file2")
	}
	got := sepToSlash(ts.ReadFile(args[0]))
	want := sepToSlash(os.Expand(ts.ReadFile(args[1]), ts.Getenv))
	if (got == want) == neg {
		ts.Fatalf("%s and %s differ\n--- got ---\n%s\n--- want ---\n%s", args[0], args[1], got, want)
	}
}

var backslashRun = regexp.MustCompile(`\\+`)

func sepToSlash(s string) string { return backslashRun.ReplaceAllString(s, "/") }

// TestSepToSlash pins the normalisation cmppath relies on, which this Linux CI
// cannot exercise through the scripts: a native path and its JSON-escaped form
// both collapse to slashes.
func TestSepToSlash(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"/tmp/work/vault", "/tmp/work/vault"},
		{`C:\work\vault`, "C:/work/vault"},
		{`{"root":"C:\\work\\vault"}`, `{"root":"C:/work/vault"}`},
		{`\\server\share`, "/server/share"},
	} {
		if got := sepToSlash(tt.in); got != tt.want {
			t.Errorf("sepToSlash(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
