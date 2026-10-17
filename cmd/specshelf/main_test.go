package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestGoInstallReportsDevelopmentVersion covers the version reporting contract
// for binaries built by `go install`, where no version is injected at build
// time: the binary must report a development marker rather than an empty or
// wrong version.
func TestGoInstallReportsDevelopmentVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary with go install")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	gobin := t.TempDir()
	install := exec.Command("go", "install", ".")
	install.Env = append(os.Environ(), "GOBIN="+gobin)
	if out, err := install.CombinedOutput(); err != nil {
		t.Fatalf("go install .: %v\n%s", err, out)
	}

	name := "specshelf"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	out, err := exec.Command(filepath.Join(gobin, name), "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("%s --version: %v\n%s", name, err, out)
	}
	if got := string(out); !strings.Contains(got, "dev") {
		t.Errorf("--version output %q does not report the development marker", got)
	}
}
