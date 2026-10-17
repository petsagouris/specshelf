package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunExitCodes pins the three exit codes and the stderr contract: a usage
// failure exits 2, a runtime failure exits 1, and every error line starts with
// the specshelf prefix.
func TestRunExitCodes(t *testing.T) {
	vault := filepath.Join(t.TempDir(), "vault")

	tests := []struct {
		name   string
		args   []string
		code   int
		stderr string
	}{
		{name: "success", args: []string{"--home", vault, "path"}, code: 0},
		{name: "unknown command", args: []string{"nope"}, code: 2, stderr: "specshelf: unknown command"},
		{name: "unknown flag", args: []string{"--home", vault, "list", "--nope"}, code: 2, stderr: "specshelf: unknown flag"},
		{name: "missing argument", args: []string{"--home", vault, "show"}, code: 2, stderr: "specshelf: accepts 1 arg(s), received 0"},
		{name: "conflicting selectors", args: []string{"--home", vault, "--vault", "work", "list"}, code: 2, stderr: "specshelf: --home and --vault are mutually exclusive"},
		{name: "invalid kind", args: []string{"--home", vault, "list", "--kind", "bogus"}, code: 2, stderr: `specshelf: invalid argument "bogus"`},
		{name: "unknown reference", args: []string{"--home", vault, "show", "missing"}, code: 1, stderr: `specshelf: unknown reference "missing"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SPECSHELF_CONFIG", filepath.Join(t.TempDir(), "config.toml"))
			var stdout, stderr bytes.Buffer
			if got := Run(tt.args, &stdout, &stderr); got != tt.code {
				t.Errorf("Run(%v) exit code = %d, want %d\nstderr: %s", tt.args, got, tt.code, stderr.String())
			}
			if tt.stderr == "" {
				if stderr.Len() != 0 {
					t.Errorf("unexpected stderr: %q", stderr.String())
				}
				return
			}
			if !strings.HasPrefix(stderr.String(), tt.stderr) {
				t.Errorf("stderr = %q, want it to start with %q", stderr.String(), tt.stderr)
			}
		})
	}
}
