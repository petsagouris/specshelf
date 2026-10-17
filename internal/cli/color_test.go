package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTerminal stands in for a real terminal, which a test process does not
// have, for the duration of one test. No test runs in parallel, so the swap is
// safe.
func withTerminal(t *testing.T, isTTY bool) {
	t.Helper()
	original := isTerminal
	isTerminal = func(io.Writer) bool { return isTTY }
	t.Cleanup(func() { isTerminal = original })
}

// unsetEnv removes an environment variable and restores it afterwards, which
// t.Setenv cannot do: t.Setenv only sets, so NO_COLOR must be unset this way to
// test that its absence permits colour.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}

// TestPaintEnabled pins the gating rules: a machine format never colours, and
// auto colours only a terminal with NO_COLOR unset and TERM other than dumb.
func TestPaintEnabled(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		machine  bool
		terminal bool
		noColor  bool
		term     string
		want     bool
	}{
		{name: "always colours a non-terminal writer", mode: colorAlways, want: true},
		{name: "never does not colour a terminal", mode: colorNever, terminal: true},
		{name: "auto does not colour a non-terminal writer", mode: colorAuto},
		{name: "auto colours a terminal", mode: colorAuto, terminal: true, want: true},
		{name: "auto with NO_COLOR does not colour", mode: colorAuto, terminal: true, noColor: true},
		{name: "auto with TERM=dumb does not colour", mode: colorAuto, terminal: true, term: "dumb"},
		{name: "a machine format never colours", mode: colorAlways, machine: true, terminal: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTerminal(t, tt.terminal)
			if tt.noColor {
				t.Setenv("NO_COLOR", "1")
			} else {
				unsetEnv(t, "NO_COLOR")
			}
			t.Setenv("TERM", tt.term)

			out := &bytes.Buffer{} // a non-terminal writer
			if got := paintEnabled(tt.mode, out, tt.machine); got != tt.want {
				t.Errorf("paintEnabled(%q, machine=%v, terminal=%v) = %v, want %v",
					tt.mode, tt.machine, tt.terminal, got, tt.want)
			}
		})
	}
}

// TestStyleBold pins that a disabled style is the identity and an enabled one
// frames its text in escapes.
func TestStyleBold(t *testing.T) {
	if got := (style{}).bold("x"); got != "x" {
		t.Errorf("disabled style bold = %q, want %q", got, "x")
	}
	if got := (style{on: true}).bold("x"); got != "\x1b[1mx\x1b[0m" {
		t.Errorf("enabled style bold = %q, want %q", got, "\x1b[1mx\x1b[0m")
	}
}

// writeVaultDoc lays down a one-document vault and returns its root.
func writeVaultDoc(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "specs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "---\nid: spec-20261017-doc\nkind: spec\ntitle: Doc\nstatus: active\n" +
		"created: 2026-10-17\nupdated: 2026-10-17\n---\n# Doc\n"
	if err := os.WriteFile(filepath.Join(dir, "2026-10-17-doc.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestMachineFormatsNeverCarryEscapes is the guarantee the spec states: --json,
// --format tsv and the single-value stdout never carry escapes, even with
// --color always. The writer is a non-terminal buffer, so auto would not colour
// either; always is what makes these cases meaningful.
func TestMachineFormatsNeverCarryEscapes(t *testing.T) {
	root := writeVaultDoc(t)
	t.Setenv("SPECSHELF_CONFIG", filepath.Join(t.TempDir(), "config.toml"))

	tests := []struct {
		name string
		args []string
	}{
		{name: "prime --json", args: []string{"--home", root, "prime", "--json", "--color", "always"}},
		{name: "list --format tsv", args: []string{"--home", root, "list", "--format", "tsv", "--color", "always"}},
		{name: "path", args: []string{"--home", root, "path", "--color", "always"}},
		{name: "new", args: []string{"--home", root, "new", "Coloured", "--color", "always"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(tt.args, &stdout, &stderr); code != 0 {
				t.Fatalf("Run(%v) exit = %d, stderr = %s", tt.args, code, stderr.String())
			}
			if strings.Contains(stdout.String(), "\x1b") {
				t.Errorf("Run(%v) stdout carries escapes: %q", tt.args, stdout.String())
			}
		})
	}
}

// TestColorFromConfig covers the config-spec scenario: with no flag, the config
// file's color setting takes effect, and a flag overrides it.
func TestColorFromConfig(t *testing.T) {
	root := writeVaultDoc(t)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("color = \"always\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPECSHELF_CONFIG", configPath)

	run := func(t *testing.T, args ...string) string {
		t.Helper()
		var stdout, stderr bytes.Buffer
		if code := Run(append([]string{"--home", root}, args...), &stdout, &stderr); code != 0 {
			t.Fatalf("Run(%v) exit != 0, stderr = %s", args, stderr.String())
		}
		return stdout.String()
	}

	if got := run(t, "prime"); !strings.Contains(got, "\x1b[1m") {
		t.Errorf("config color=always did not colour prime: %q", got)
	}
	if got := run(t, "--color", "never", "prime"); strings.Contains(got, "\x1b") {
		t.Errorf("--color never did not override config color=always: %q", got)
	}
}

// TestInvalidColorIsAnError pins both bad-value paths: an unusable flag value is
// a usage error, and an unusable configuration value is a runtime error naming
// the mode rather than a silent fallback.
func TestInvalidColorIsAnError(t *testing.T) {
	root := writeVaultDoc(t)
	t.Setenv("SPECSHELF_CONFIG", filepath.Join(t.TempDir(), "config.toml"))

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--home", root, "--color", "purple", "path"}, &stdout, &stderr); code != exitUsage {
		t.Errorf("--color purple exit = %d, want %d (stderr %q)", code, exitUsage, stderr.String())
	}
	if got, want := stderr.String(), "specshelf: invalid argument \"purple\" for \"--color\" flag: must be auto, always, or never\n"; got != want {
		t.Errorf("--color purple stderr = %q, want %q", got, want)
	}

	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("color = \"purple\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SPECSHELF_CONFIG", configPath)
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"--home", root, "prime"}, &stdout, &stderr); code != exitError {
		t.Errorf("config color=purple exit = %d, want %d (stderr %q)", code, exitError, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "specshelf: ") {
		t.Errorf("stderr = %q, want the specshelf prefix", stderr.String())
	}
}
