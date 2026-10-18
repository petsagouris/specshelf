package cli

import (
	"bytes"
	"os"
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

// TestRunClassifiesTypedErrors pins the exit code for every typed error the
// store returns. A bad selector value — an undeclared --vault name, a --project
// the configuration disables — is a usage failure (2); a problem found while
// reading the vault is a runtime failure (1). OutsideVaultError is the runtime
// case even though it is a refusal, because the command line was valid.
func TestRunClassifiesTypedErrors(t *testing.T) {
	noConfig := func(t *testing.T) {
		t.Helper()
		t.Setenv("SPECSHELF_CONFIG", filepath.Join(t.TempDir(), "config.toml"))
	}
	assertExit := func(t *testing.T, code, want int, stderr string) {
		t.Helper()
		if code != want {
			t.Errorf("exit code = %d, want %d\nstderr: %s", code, want, stderr)
		}
	}

	t.Run("unknown vault name is usage", func(t *testing.T) {
		noConfig(t)
		_, stderr, code := runCLI(t, "--vault", "nope", "list")
		assertExit(t, code, 2, stderr)
		if !strings.HasPrefix(stderr, `specshelf: unknown vault "nope"`) {
			t.Errorf("stderr = %q", stderr)
		}
	})

	t.Run("project disabled by config is usage", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("project = false\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SPECSHELF_CONFIG", path)
		_, stderr, code := runCLI(t, "--project", "list")
		assertExit(t, code, 2, stderr)
		if !strings.Contains(stderr, "disabled by the project setting") {
			t.Errorf("stderr = %q", stderr)
		}
	})

	t.Run("no project vault is runtime", func(t *testing.T) {
		noConfig(t)
		t.Chdir(t.TempDir())
		_, stderr, code := runCLI(t, "--project", "list")
		assertExit(t, code, 1, stderr)
		if !strings.HasPrefix(stderr, "specshelf: no project vault") {
			t.Errorf("stderr = %q", stderr)
		}
	})

	t.Run("ambiguous reference is runtime", func(t *testing.T) {
		noConfig(t)
		root := t.TempDir()
		docs := map[string]string{
			"specs/2026-10-17-dup.md":     "---\nid: spec-20261017-dup\nkind: spec\ntitle: Dup\nstatus: active\ncreated: 2026-10-17\nupdated: 2026-10-17\n---\n# Dup\n",
			"decisions/2026-10-17-dup.md": "---\nid: decision-20261017-dup\nkind: decision\ntitle: Dup\nstatus: active\ncreated: 2026-10-17\nupdated: 2026-10-17\n---\n# Dup\n",
		}
		for name, data := range docs {
			path := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		_, stderr, code := runCLI(t, "--home", root, "show", "2026-10-17-dup.md")
		assertExit(t, code, 1, stderr)
		if !strings.HasPrefix(stderr, `specshelf: ambiguous reference "2026-10-17-dup.md"`) {
			t.Errorf("stderr = %q", stderr)
		}
	})

	t.Run("outside-vault reference is runtime", func(t *testing.T) {
		noConfig(t)
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "outside.md")
		if err := os.WriteFile(outside, []byte("---\nid: spec-outside\nkind: spec\ntitle: Outside\n---\n# Outside\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, stderr, code := runCLI(t, "--home", root, "rm", outside)
		assertExit(t, code, 1, stderr)
		if !strings.Contains(stderr, "is outside the vault") {
			t.Errorf("stderr = %q", stderr)
		}
		if _, err := os.Stat(outside); err != nil {
			t.Errorf("the outside file was touched: %v", err)
		}
	})
}

// A hand-picked --slug the store refuses is a usage failure, and each rejection
// leaves the vault untouched: the slug is validated before the vault is created
// (task 4.12). An explicitly empty value is a rejection too, even though an
// absent --slug means "derive from the title".
func TestInvalidSlugIsUsageError(t *testing.T) {
	for _, slug := range []string{"", "a/b", `a\b`, ".", "..", "../escape"} {
		t.Run(slug, func(t *testing.T) {
			t.Setenv("SPECSHELF_CONFIG", filepath.Join(t.TempDir(), "config.toml"))
			root := filepath.Join(t.TempDir(), "vault")
			_, stderr, code := runCLI(t, "--home", root, "new", "Some Title", "--slug", slug)
			if code != 2 {
				t.Errorf("exit code = %d, want 2\nstderr: %s", code, stderr)
			}
			if !strings.HasPrefix(stderr, "specshelf: invalid slug") {
				t.Errorf("stderr = %q, want it to start with the invalid-slug message", stderr)
			}
			if entries, err := os.ReadDir(root); err == nil && len(entries) != 0 {
				t.Errorf("the rejection touched the vault: %v", entries)
			}
		})
	}
}
