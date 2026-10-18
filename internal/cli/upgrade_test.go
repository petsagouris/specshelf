package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stubProxy points the module proxy at a local server that answers every
// request with status and body, and restores the endpoint afterwards, so the
// tests never reach the network.
func stubProxy(t *testing.T, status int, body string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	previous := proxyBaseURL
	proxyBaseURL = server.URL
	t.Cleanup(func() {
		proxyBaseURL = previous
		server.Close()
	})
}

// withVersion pins Version() for one test, restoring it afterwards.
func withVersion(t *testing.T, v string) {
	t.Helper()
	previous := version
	version = v
	t.Cleanup(func() { version = previous })
}

// withGOOS pins the platform upgrade acts as, so the Windows path is testable
// anywhere.
func withGOOS(t *testing.T, goos string) {
	t.Helper()
	previous := upgradeGOOS
	upgradeGOOS = goos
	t.Cleanup(func() { upgradeGOOS = previous })
}

// executablePath is the resolved path of the test binary, matching what
// upgrade reports for the running executable.
func executablePath(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe
}

// TestUpgradeCheck reports both versions from the stubbed proxy and exits 0
// whether or not an update is available.
func TestUpgradeCheck(t *testing.T) {
	stubProxy(t, http.StatusOK, `{"Version": "v9.9.9"}`)

	t.Run("update available", func(t *testing.T) {
		withVersion(t, "v1.0.0")
		stdout, stderr, code := runCLI(t, "upgrade", "--check")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr)
		}
		for _, want := range []string{"v1.0.0", "v9.9.9", "update is available"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("stdout = %q, want it to contain %q", stdout, want)
			}
		}
		if stderr != "" {
			t.Errorf("stderr = %q, want empty", stderr)
		}
	})

	t.Run("already up to date", func(t *testing.T) {
		withVersion(t, "v9.9.9")
		stdout, stderr, code := runCLI(t, "upgrade", "--check")
		if code != 0 {
			t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr)
		}
		if !strings.Contains(stdout, "already up to date") {
			t.Errorf("stdout = %q, want it to say the binary is current", stdout)
		}
		if !strings.Contains(stdout, "v9.9.9") {
			t.Errorf("stdout = %q, want both versions", stdout)
		}
	})
}

// TestUpgradeCheckProxyUnreachable fails non-zero naming the network failure,
// and makes no version claim on stdout.
func TestUpgradeCheckProxyUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close()
	previous := proxyBaseURL
	proxyBaseURL = url
	t.Cleanup(func() { proxyBaseURL = previous })

	stdout, stderr, code := runCLI(t, "upgrade", "--check")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\nstderr: %s", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want no version claim", stdout)
	}
	if !strings.HasPrefix(stderr, "specshelf: ") || !strings.Contains(stderr, "module proxy") {
		t.Errorf("stderr = %q, want it to name the module proxy failure", stderr)
	}
}

// TestUpgradeOutsideInstallDir prints the exact command and the running
// executable's path, and exits 0, when the binary is not where `go install`
// writes.
func TestUpgradeOutsideInstallDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows always takes the no-replace path")
	}
	t.Setenv("GOBIN", t.TempDir())
	exe := executablePath(t)

	stdout, stderr, code := runCLI(t, "upgrade")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, installCommand) {
		t.Errorf("stdout = %q, want the install command %q", stdout, installCommand)
	}
	if !strings.Contains(stdout, exe) {
		t.Errorf("stdout = %q, want the running executable %q", stdout, exe)
	}
}

// TestUpgradeWindowsNeverReplaces proves the Windows path prints the command
// even when the running executable is in the install directory.
func TestUpgradeWindowsNeverReplaces(t *testing.T) {
	withGOOS(t, "windows")
	t.Setenv("GOBIN", filepath.Dir(executablePath(t)))

	stdout, stderr, code := runCLI(t, "upgrade")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "cannot be replaced on Windows") {
		t.Errorf("stdout = %q, want the Windows no-replace message", stdout)
	}
	if !strings.Contains(stdout, installCommand) {
		t.Errorf("stdout = %q, want the install command", stdout)
	}
}

// TestUpgradeMissingToolchain reports the missing Go toolchain and prints the
// command, without modifying anything, when the binary is in the install
// directory but `go` is not on PATH.
func TestUpgradeMissingToolchain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows takes the no-replace path before the toolchain check")
	}
	t.Setenv("GOBIN", filepath.Dir(executablePath(t)))
	t.Setenv("PATH", t.TempDir())

	stdout, stderr, code := runCLI(t, "upgrade")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\nstderr: %s", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Go toolchain is required") || !strings.Contains(stderr, installCommand) {
		t.Errorf("stderr = %q, want the missing-toolchain message and the command", stderr)
	}
}
