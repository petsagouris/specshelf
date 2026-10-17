// Package conformance proves that specshelf and the Python reference
// implementation stay interchangeable. The default suite compares the command
// surface against committed fixtures and needs no Python; the
// //go:build conformance differential suite runs both tools.
//
// The package is test-only. It builds the real binary and drives it against
// copies of the fixture vaults under temporary directories, so it never writes
// a vault through a second code path.
package conformance

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// binaryPath is the specshelf binary built once for the package, so the tests
// drive cmd/specshelf's real exit wiring rather than an in-process command tree.
var binaryPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "specshelf-conformance")
	if err != nil {
		panic("conformance: create binary directory: " + err.Error())
	}
	binaryPath = filepath.Join(dir, "specshelf"+exeSuffix())
	build := exec.Command("go", "build", "-o", binaryPath, "github.com/petsagouris/specshelf/cmd/specshelf")
	if out, err := build.CombinedOutput(); err != nil {
		_, _ = os.Stderr.WriteString("conformance: build specshelf: " + err.Error() + "\n" + string(out))
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// result is one command invocation's observable outcome.
type result struct {
	stdout []byte
	stderr []byte
	code   int
}

// runSpecshelf drives the built binary with an explicit vault home. The config
// file is pointed at a path that does not exist under t's temporary directory,
// so no real configuration or vault is ever read.
func runSpecshelf(t *testing.T, home string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binaryPath, append([]string{"--home", home}, args...)...)
	cmd.Env = append(os.Environ(), "SPECSHELF_CONFIG="+filepath.Join(t.TempDir(), "config.toml"))

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("run specshelf %v: %v", args, err)
		}
		code = exit.ExitCode()
	}
	return result{stdout: stdout.Bytes(), stderr: stderr.Bytes(), code: code}
}

// substituteOnce replaces every occurrence of from in b with to, failing unless
// from occurs exactly want times. Pinning the count is what keeps a tolerance
// narrow: a substitution can never quietly absorb a second, different
// difference.
func substituteOnce(t *testing.T, label string, b, from, to []byte, want int) []byte {
	t.Helper()
	if got := bytes.Count(b, from); got != want {
		t.Fatalf("%s: %q occurs %d times, want %d", label, from, got, want)
	}
	return bytes.ReplaceAll(b, from, to)
}
