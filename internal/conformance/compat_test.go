package conformance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const (
	fixtureVault    = "../../testdata/python_vault"
	fixtureExpected = "../../testdata/python_vault_expected"

	// The scratch directories the recorded fixtures embed. The fixtures were
	// generated against copies of the vault under /tmp/opencode/pyspecv/
	// (testdata/README.md), so any absolute path in them is the generation
	// path, not the fixture's own. Comparing absolute-path output means putting
	// this run's vault directory in its place.
	recordedInitRoot  = "/tmp/opencode/pyspecv/expected_init"
	recordedVaultRoot = "/tmp/opencode/pyspecv/expected_vault"
)

// recordedCommands are the ten invocations testdata/python_vault_expected/
// records, in the order testdata/README.md lists. fresh runs against a new
// empty vault (init creates one); every other command runs against a copy of
// the fixture vault. pathRoot is the recorded absolute path the fixture embeds,
// or empty when the output contains none.
var recordedCommands = []struct {
	name     string
	args     []string
	fresh    bool
	pathRoot string
}{
	{"init", []string{"init"}, true, recordedInitRoot},
	{"list", []string{"list"}, false, ""},
	{"list_json", []string{"list", "--json"}, false, ""},
	{"list_kind_decision_tag_tooling", []string{"list", "--kind", "decision", "--tag", "tooling"}, false, ""},
	{"search_beads", []string{"search", "Beads"}, false, ""},
	{"search_quality", []string{"search", "quality"}, false, ""},
	{"show", []string{"show", "spec-20261017-quality-analysis-display-contract"}, false, ""},
	{"prime_repo", []string{"prime", "--repo", "lippi-smart-customer"}, false, recordedVaultRoot},
	{"index_print", []string{"index", "--print"}, false, recordedVaultRoot},
	{"path", []string{"path"}, false, recordedVaultRoot},
}

// TestCompatibilityFixtures drives specshelf against the committed reference
// fixtures with Python absent: every recorded command's stdout, stderr and exit
// code is compared byte for byte, and the resulting vault files are compared
// against the fixture vault. Only two substitutions are tolerated, each
// asserted and narrow: the generation scratch path the fixture embeds, and the
// one enumerated title divergence.
func TestCompatibilityFixtures(t *testing.T) {
	initHome := t.TempDir()
	home := t.TempDir()
	copyTree(t, fixtureVault, home)

	codes := readExitCodes(t)

	for _, c := range recordedCommands {
		t.Run(c.name, func(t *testing.T) {
			dir := home
			if c.fresh {
				dir = initHome
			}
			got := runSpecshelf(t, dir, c.args...)

			if code := codes[c.name]; got.code != code {
				t.Errorf("exit code = %d, want %d", got.code, code)
			}

			want := readFixture(t, filepath.Join(fixtureExpected, c.name+".stdout"))
			if c.pathRoot != "" {
				want = substituteOnce(t, c.name, want, []byte(c.pathRoot), []byte(dir), 1)
				want = []byte(filepath.ToSlash(string(want)))
			}
			want = tolerateEmptyScalarTitle(t, c.name, want)
			if gotStdout := []byte(filepath.ToSlash(string(got.stdout))); !bytes.Equal(gotStdout, want) {
				t.Errorf("stdout differs:\n got %q\nwant %q", gotStdout, want)
			}

			wantErr := readFixture(t, filepath.Join(fixtureExpected, c.name+".stderr"))
			if !bytes.Equal(got.stderr, wantErr) {
				t.Errorf("stderr differs:\n got %q\nwant %q", got.stderr, wantErr)
			}
		})
	}

	t.Run("init writes the reference files", func(t *testing.T) {
		want := map[string][]byte{}
		for _, rel := range []string{"README.md", "templates/spec.md", "templates/decision.md"} {
			want[rel] = readFixture(t, filepath.Join(fixtureVault, filepath.FromSlash(rel)))
		}
		assertTreesEqual(t, vaultTree(t, initHome), want)
	})

	t.Run("commands leave the reference files intact", func(t *testing.T) {
		got := vaultTree(t, home)
		want := vaultTree(t, fixtureVault)
		// index --print regenerates index.jsonl, so it is the one file the read
		// commands change; the reference's empty-title value and ours are the
		// enumerated divergence that also shows up in list and list --json.
		want["index.jsonl"] = tolerateEmptyScalarTitle(t, "index.jsonl", want["index.jsonl"])
		assertTreesEqual(t, got, want)
	})
}

// tolerateEmptyScalarTitle applies the one enumerated divergence the compared
// commands expose: the reference reader records an empty scalar as the text
// "[]", specshelf records it as empty (task 3.13, "empty scalar values reading
// as empty rather than as []"). It asserts the recorded form occurs exactly
// once, so the substitution can never hide another difference.
func tolerateEmptyScalarTitle(t *testing.T, label string, b []byte) []byte {
	t.Helper()
	switch label {
	case "list":
		return substituteOnce(t, label, b, []byte("\t[]\n"), []byte("\t\n"), 1)
	case "list_json", "index.jsonl":
		return substituteOnce(t, label, b, []byte(`"title": "[]"`), []byte(`"title": ""`), 1)
	default:
		return b
	}
}

// readExitCodes reads python_vault_expected/exit_codes.json.
func readExitCodes(t *testing.T) map[string]int {
	t.Helper()
	data := readFixture(t, filepath.Join(fixtureExpected, "exit_codes.json"))
	var codes map[string]int
	if err := json.Unmarshal(data, &codes); err != nil {
		t.Fatalf("parse exit_codes.json: %v", err)
	}
	return codes
}

// readFixture reads a committed fixture, failing when it cannot.
func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}

// vaultTree maps every regular file under dir to its bytes, keyed by a
// slash-separated path relative to dir.
func vaultTree(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	tree := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		tree[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return tree
}

// assertTreesEqual compares two file trees exactly, naming every file that is
// missing, extra or different.
func assertTreesEqual(t *testing.T, got, want map[string][]byte) {
	t.Helper()
	for rel, wantData := range want {
		gotData, ok := got[rel]
		if !ok {
			t.Errorf("%s is missing", rel)
			continue
		}
		if !bytes.Equal(gotData, wantData) {
			t.Errorf("%s differs:\n got %q\nwant %q", rel, gotData, wantData)
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			t.Errorf("%s was created but is not in the reference", rel)
		}
	}
}

// copyTree copies every regular file under src to dst, creating directories and
// preserving file modes.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copy %s to %s: %v", src, dst, err)
	}
}
