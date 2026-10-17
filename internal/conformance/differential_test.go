//go:build conformance

package conformance

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// pythonDir is the upstream checkout the fixtures were generated from
// (testdata/README.md). Upstream documents running the tool from a checkout as
// `python3 -m specv` with that directory as the working directory.
const pythonDir = "/tmp/opencode/pyspecv/agent_spec_vault-0.1.2"

// pythonTool is the Python reference implementation. requirePython returns one,
// or skips the test when Python or the package is absent, so a machine without
// it reports the comparison as skipped rather than failing the build.
type pythonTool struct{ python string }

func requirePython(t *testing.T) pythonTool {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not on PATH: skipping the differential comparison")
	}
	if _, err := os.Stat(pythonDir); err != nil {
		t.Skipf("the reference implementation is not at %s: skipping", pythonDir)
	}
	probe := exec.Command(python, "-m", "specv", "--version")
	probe.Dir = pythonDir
	if out, err := probe.CombinedOutput(); err != nil {
		t.Skipf("python3 -m specv is unavailable (%v): %s", err, out)
	}
	return pythonTool{python: python}
}

// run executes one reference invocation against an explicit vault home. HOME is
// redirected to a temporary directory so the reference cannot read or write the
// real ~/.agents/spec-vault behind --home's back.
func (p pythonTool) run(t *testing.T, home string, args ...string) result {
	t.Helper()
	cmd := exec.Command(p.python, append([]string{"-m", "specv", "--home", home}, args...)...)
	cmd.Dir = pythonDir
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("run reference specv %v: %v", args, err)
		}
		code = exit.ExitCode()
	}
	return result{stdout: stdout.Bytes(), stderr: stderr.Bytes(), code: code}
}

// diffCase is one command run against two equivalent vaults, one per tool.
type diffCase struct {
	name  string
	setup func(t *testing.T, py pythonTool) (ourHome, pyHome string)
	args  []string
	// ignoreIndex excludes index.jsonl from the file comparison: specshelf
	// refreshes the index automatically and the reference does so only when
	// asked (tasks.md 3.13, "automatic index refresh").
	ignoreIndex bool
	// stdout tolerates the enumerated divergences in the reference's stdout.
	stdout func(*testing.T, []byte) []byte
	// files tolerates the enumerated divergences in the reference's files.
	files func(*testing.T, string, []byte) []byte
}

// TestDifferentialAgainstPython runs both tools against equivalent vaults and
// compares stdout, exit codes and the resulting files. A difference that no
// enumerated divergence explains fails the test and is named.
func TestDifferentialAgainstPython(t *testing.T) {
	py := requirePython(t)

	cases := []diffCase{
		{
			name:  "init",
			setup: freshDirs,
			args:  []string{"init"},
		},
		{
			name:  "list",
			setup: fixtureDirs,
			args:  []string{"list"},
			stdout: func(t *testing.T, b []byte) []byte {
				return substituteOnce(t, "list", b, []byte("\t[]\n"), []byte("\t\n"), 1)
			},
		},
		{
			name:  "list --json",
			setup: fixtureDirs,
			args:  []string{"list", "--json"},
			stdout: func(t *testing.T, b []byte) []byte {
				return substituteOnce(t, "list --json", b, []byte(`"title": "[]"`), []byte(`"title": ""`), 1)
			},
		},
		{
			name:  "list --kind decision --tag tooling",
			setup: fixtureDirs,
			args:  []string{"list", "--kind", "decision", "--tag", "tooling"},
		},
		{
			name:  "search Beads",
			setup: fixtureDirs,
			args:  []string{"search", "Beads"},
		},
		{
			name:  "search quality",
			setup: fixtureDirs,
			args:  []string{"search", "quality"},
		},
		{
			name:  "show",
			setup: fixtureDirs,
			args:  []string{"show", "spec-20261017-quality-analysis-display-contract"},
		},
		{
			name:  "prime --repo",
			setup: fixtureDirs,
			args:  []string{"prime", "--repo", "lippi-smart-customer"},
		},
		{
			name:  "index --print",
			setup: fixtureDirs,
			args:  []string{"index", "--print"},
			files: indexEmptyScalar,
		},
		{
			name:  "path",
			setup: fixtureDirs,
			args:  []string{"path"},
		},
		{
			name:        "new",
			setup:       initDirs,
			args:        []string{"new", "Alpha spec"},
			ignoreIndex: true,
			files:       emptyListFiles,
		},
		{
			name:        "new reserved title",
			setup:       initDirs,
			args:        []string{"new", "#1 priority"},
			ignoreIndex: true,
			// The reference writes the title bare, which is not valid YAML;
			// specshelf quotes it (tasks.md 3.13, "valid-YAML emission").
			files: func(t *testing.T, rel string, ref []byte) []byte {
				ref = emptyListFields(ref)
				if strings.HasPrefix(rel, "specs/") {
					ref = substituteOnce(t, rel, ref, []byte("title: #1 priority"), []byte(`title: "#1 priority"`), 1)
				}
				return ref
			},
		},
		{
			name:        "decision",
			setup:       initDirs,
			args:        []string{"decision", "Adopt Markdown", "--status", "accepted", "--tags", "alpha"},
			ignoreIndex: true,
			files:       emptyListFiles,
		},
		{
			name:        "link-session",
			setup:       fixtureDirs,
			args:        []string{"link-session", "spec-20261017-quality-analysis-display-contract", "--codex-session", "roundtrip-1"},
			ignoreIndex: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ourHome, pyHome := c.setup(t, py)
			ours := runSpecshelf(t, ourHome, c.args...)
			theirs := py.run(t, pyHome, c.args...)

			compareOutput(t, "stdout", normaliseVaultPath(ours.stdout, ourHome), tolerate(t, c.stdout, theirs.stdout, pyHome))
			compareOutput(t, "stderr", normaliseVaultPath(ours.stderr, ourHome), normaliseVaultPath(theirs.stderr, pyHome))
			if ours.code != theirs.code {
				t.Errorf("exit code = %d, the reference exits %d", ours.code, theirs.code)
			}

			got, want := vaultTree(t, ourHome), vaultTree(t, pyHome)
			if c.ignoreIndex {
				delete(got, "index.jsonl")
				delete(want, "index.jsonl")
			}
			for rel, ref := range want {
				if c.files != nil {
					ref = c.files(t, rel, ref)
				}
				want[rel] = ref
			}
			assertTreesEqual(t, got, want)
		})
	}
}

// tolerate applies the case's stdout tolerance to the reference output.
func tolerate(t *testing.T, f func(*testing.T, []byte) []byte, ref []byte, home string) []byte {
	if f == nil {
		return normaliseVaultPath(ref, home)
	}
	return normaliseVaultPath(f(t, ref), home)
}

// normaliseVaultPath replaces the run's vault directory with a fixed token and
// converts path separators, so the two tools' absolute paths compare equal on
// every platform. The fixtures embed a Unix scratch path for the same reason
// (testdata/README.md).
func normaliseVaultPath(b []byte, home string) []byte {
	b = bytes.ReplaceAll(b, []byte(home), []byte("<vault>"))
	return []byte(filepath.ToSlash(string(b)))
}

// compareOutput fails with both values when they differ, naming the first line
// that does.
func compareOutput(t *testing.T, label string, got, want []byte) {
	t.Helper()
	if bytes.Equal(got, want) {
		return
	}
	gotLines, wantLines := strings.Split(string(got), "\n"), strings.Split(string(want), "\n")
	for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
		var g, w string
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if g != w {
			t.Errorf("%s line %d differs:\n got %q\nwant %q", label, i+1, g, w)
			break
		}
	}
	t.Errorf("%s differs:\n got %q\nwant %q", label, got, want)
}

// emptyListFiles tolerates specshelf writing empty list fields as [] where the
// reference writes a bare key (tasks.md 3.13).
func emptyListFiles(_ *testing.T, _ string, ref []byte) []byte { return emptyListFields(ref) }

// indexEmptyScalar tolerates the empty-title field in a regenerated index.jsonl.
func indexEmptyScalar(t *testing.T, rel string, ref []byte) []byte {
	if rel != "index.jsonl" {
		return ref
	}
	return substituteOnce(t, rel, ref, []byte(`"title": "[]"`), []byte(`"title": ""`), 1)
}

// emptyListFields rewrites the reference's bare empty list keys to the "[]"
// form specshelf writes, matching only a whole line for one of the three list
// keys and only when no list item follows it, so a list with items is left
// untouched.
func emptyListFields(b []byte) []byte {
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		switch line {
		case "repos:", "tags:", "codex_sessions:":
			if i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "- ") {
				continue
			}
			lines[i] = line + " []"
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// freshDirs returns two empty vault directories.
func freshDirs(t *testing.T, _ pythonTool) (string, string) {
	return t.TempDir(), t.TempDir()
}

// fixtureDirs returns two copies of the committed reference vault.
func fixtureDirs(t *testing.T, _ pythonTool) (string, string) {
	ourHome, pyHome := t.TempDir(), t.TempDir()
	copyTree(t, fixtureVault, ourHome)
	copyTree(t, fixtureVault, pyHome)
	return ourHome, pyHome
}

// initDirs returns two vaults initialised by their own tool.
func initDirs(t *testing.T, py pythonTool) (string, string) {
	ourHome, pyHome := t.TempDir(), t.TempDir()
	if got := runSpecshelf(t, ourHome, "init"); got.code != 0 {
		t.Fatalf("specshelf init exited %d: %s", got.code, got.stderr)
	}
	if got := py.run(t, pyHome, "init"); got.code != 0 {
		t.Fatalf("reference init exited %d: %s", got.code, got.stderr)
	}
	return ourHome, pyHome
}
