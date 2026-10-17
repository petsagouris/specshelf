//go:build conformance

package conformance

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// TestSpecshelfWrittenVaultReadableByPython is the reverse direction of
// bidirectional interchange: a vault specshelf wrote is listed, shown and
// searched by the Python tool with its metadata intact.
func TestSpecshelfWrittenVaultReadableByPython(t *testing.T) {
	py := requirePython(t)
	home := t.TempDir()

	for _, args := range [][]string{
		{"init"},
		{"new", "Mixed authorship from specshelf", "--tags", "alpha,beta"},
		{"decision", "Adopt Markdown vault", "--status", "accepted", "--tags", "tooling"},
	} {
		if got := runSpecshelf(t, home, args...); got.code != 0 {
			t.Fatalf("specshelf %v exited %d: %s", args, got.code, got.stderr)
		}
	}

	// The id embeds the current date, so the expectations are keyed by title
	// and specshelf's own listing supplies the ids to compare and to show.
	want := map[string]struct {
		kind string
		tags []string
	}{
		"Mixed authorship from specshelf": {"spec", []string{"alpha", "beta"}},
		"Adopt Markdown vault":            {"decision", []string{"tooling"}},
	}
	ourRecords := parseRecords(t, runSpecshelf(t, home, "list", "--json").stdout)
	if len(ourRecords) != len(want) {
		t.Fatalf("specshelf listed %d documents, want %d", len(ourRecords), len(want))
	}
	for _, record := range ourRecords {
		expected, ok := want[record.Title]
		if !ok {
			t.Errorf("specshelf wrote an unexpected document %q", record.Title)
			continue
		}
		if record.Kind != expected.kind || !slices.Equal(record.Tags, expected.tags) {
			t.Errorf("specshelf read %q back as kind %q tags %q, want kind %q tags %q",
				record.Title, record.Kind, record.Tags, expected.kind, expected.tags)
		}
	}

	listed := py.run(t, home, "list", "--json")
	if listed.code != 0 {
		t.Fatalf("reference list exited %d: %s", listed.code, listed.stderr)
	}
	if len(listed.stderr) != 0 {
		t.Errorf("reference list stderr = %q, want empty", listed.stderr)
	}
	theirs := parseRecords(t, listed.stdout)
	if len(theirs) != len(ourRecords) {
		t.Fatalf("the reference listed %d documents, want %d", len(theirs), len(ourRecords))
	}
	byID := make(map[string]indexRecord, len(theirs))
	for _, record := range theirs {
		byID[record.ID] = record
	}
	for _, our := range ourRecords {
		record, ok := byID[our.ID]
		if !ok {
			t.Errorf("the reference does not see %q (id %q)", our.Title, our.ID)
			continue
		}
		if record.Kind != our.Kind || record.Title != our.Title || !slices.Equal(record.Tags, our.Tags) {
			t.Errorf("%s: the reference read %+v, want %+v", our.ID, record, our)
		}
	}

	for _, record := range ourRecords {
		shown := py.run(t, home, "show", record.ID)
		if shown.code != 0 {
			t.Errorf("reference show %s exited %d: %s", record.ID, shown.code, shown.stderr)
		}
		if !bytes.Contains(shown.stdout, []byte("# ")) {
			t.Errorf("reference show %s: output does not look like a document:\n%q", record.ID, shown.stdout)
		}
	}

	found := py.run(t, home, "search", "Mixed")
	if found.code != 0 {
		t.Errorf("reference search exited %d: %s", found.code, found.stderr)
	}
	if !bytes.Contains(found.stdout, []byte("Mixed authorship from specshelf")) {
		t.Errorf("reference search output does not contain the document:\n%q", found.stdout)
	}
}

// TestMixedAuthorship has both tools add documents to the same vault and checks
// that each sees the other's documents and neither writes a file the other
// cannot read.
func TestMixedAuthorship(t *testing.T) {
	py := requirePython(t)
	home := t.TempDir()

	if got := runSpecshelf(t, home, "init"); got.code != 0 {
		t.Fatalf("specshelf init exited %d: %s", got.code, got.stderr)
	}
	if got := runSpecshelf(t, home, "new", "Specshelf wrote this"); got.code != 0 {
		t.Fatalf("specshelf new exited %d: %s", got.code, got.stderr)
	}
	added := py.run(t, home, "new", "Python wrote this")
	if added.code != 0 {
		t.Fatalf("reference new exited %d: %s", added.code, added.stderr)
	}

	ours := runSpecshelf(t, home, "list", "--json")
	if ours.code != 0 {
		t.Fatalf("specshelf list exited %d: %s", ours.code, ours.stderr)
	}
	theirs := py.run(t, home, "list", "--json")
	if theirs.code != 0 {
		t.Fatalf("reference list exited %d: %s", theirs.code, theirs.stderr)
	}
	for _, record := range []struct {
		tool  string
		title string
		out   []byte
	}{
		{"specshelf", "Specshelf wrote this", ours.stdout},
		{"specshelf", "Python wrote this", ours.stdout},
		{"python", "Specshelf wrote this", theirs.stdout},
		{"python", "Python wrote this", theirs.stdout},
	} {
		if !bytes.Contains(record.out, []byte(record.title)) {
			t.Errorf("%s does not see the document %q", record.tool, record.title)
		}
	}
}
