package conformance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
			gotStdout := got.stdout
			if c.pathRoot != "" {
				// The output embeds the run's vault path, which the tool prints
				// with the platform separator while the fixture records "/", so
				// normalise both sides. Only these commands can carry a native
				// separator: the rest use vault-relative paths that are "/" on
				// every platform, and a blanket ToSlash on them also rewrites the
				// backslash JSON uses to escape a quote, which made list_json
				// differ on Windows.
				want = substituteOnce(t, c.name, want, []byte(c.pathRoot), []byte(dir), 1)
				want = []byte(filepath.ToSlash(string(want)))
				gotStdout = []byte(filepath.ToSlash(string(gotStdout)))
			}
			want = tolerateEmptyScalarTitle(t, c.name, want)
			if !bytes.Equal(gotStdout, want) {
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

// TestPythonWrittenVaultReadsWithoutWarnings is the default-suite direction of
// bidirectional interchange: a vault written only by the Python tool is listed,
// searched and shown by specshelf with its metadata intact and no warning on
// stderr. It needs no Python.
func TestPythonWrittenVaultReadsWithoutWarnings(t *testing.T) {
	home := t.TempDir()
	copyTree(t, fixtureVault, home)

	want := parseRecords(t, readFixture(t, filepath.Join(fixtureVault, "index.jsonl")))
	if len(want) != 10 {
		t.Fatalf("the fixture index lists %d documents, want 10", len(want))
	}

	t.Run("list --json", func(t *testing.T) {
		got := runSpecshelf(t, home, "list", "--json")
		if got.code != 0 {
			t.Errorf("exit code = %d, want 0", got.code)
		}
		if len(got.stderr) != 0 {
			t.Errorf("stderr = %q, want empty (no warnings)", got.stderr)
		}
		records := parseRecords(t, got.stdout)
		if len(records) != len(want) {
			t.Fatalf("listed %d documents, want %d", len(records), len(want))
		}
		byID := make(map[string]indexRecord, len(records))
		for _, record := range records {
			byID[record.ID] = record
		}
		for _, expected := range want {
			record, ok := byID[expected.ID]
			if !ok {
				t.Errorf("document %q is missing", expected.ID)
				continue
			}
			// The one enumerated divergence: the reference records an empty
			// scalar title as the text "[]", specshelf records it as empty.
			if expected.Title == "[]" {
				expected.Title = ""
			}
			if !sameRecord(record, expected) {
				t.Errorf("%s: metadata = %+v, want %+v", expected.ID, record, expected)
			}
		}
	})

	t.Run("show every document", func(t *testing.T) {
		for _, expected := range want {
			got := runSpecshelf(t, home, "show", expected.ID)
			if got.code != 0 {
				t.Errorf("show %s: exit code = %d, want 0", expected.ID, got.code)
			}
			if len(got.stderr) != 0 {
				t.Errorf("show %s: stderr = %q, want empty", expected.ID, got.stderr)
			}
			if !bytes.HasPrefix(got.stdout, []byte("---\n")) || !bytes.Contains(got.stdout, []byte("# ")) {
				t.Errorf("show %s: output does not look like a document:\n%q", expected.ID, got.stdout)
			}
		}
	})

	t.Run("search reaches every document", func(t *testing.T) {
		// Every document's body carries an "## " heading, so this finds all of
		// them without relying on any one template's section names.
		got := runSpecshelf(t, home, "search", "##")
		if got.code != 0 {
			t.Errorf("exit code = %d, want 0", got.code)
		}
		if len(got.stderr) != 0 {
			t.Errorf("stderr = %q, want empty", got.stderr)
		}
		found := map[string]bool{}
		for _, line := range strings.Split(strings.TrimSuffix(string(got.stdout), "\n"), "\n") {
			if id, _, ok := strings.Cut(line, "\t"); ok {
				found[id] = true
			}
		}
		for _, expected := range want {
			if !found[expected.ID] {
				t.Errorf("search did not find %q", expected.ID)
			}
		}
	})
}

// indexRecord is one line of index.jsonl. Both the reference index and the
// list --json output use the same keys, so one struct parses either.
type indexRecord struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	Created       string   `json:"created"`
	Updated       string   `json:"updated"`
	Repos         []string `json:"repos"`
	Tags          []string `json:"tags"`
	CodexSessions []string `json:"codex_sessions"`
	Path          string   `json:"path"`
}

// sameRecord compares two index records, including their list fields.
func sameRecord(a, b indexRecord) bool {
	return a.ID == b.ID && a.Kind == b.Kind && a.Title == b.Title && a.Status == b.Status &&
		a.Created == b.Created && a.Updated == b.Updated && a.Path == b.Path &&
		slices.Equal(a.Repos, b.Repos) && slices.Equal(a.Tags, b.Tags) &&
		slices.Equal(a.CodexSessions, b.CodexSessions)
}

// parseRecords parses newline-delimited index records, failing on a malformed
// line so a corrupt payload cannot pass as an empty one.
func parseRecords(t *testing.T, data []byte) []indexRecord {
	t.Helper()
	var records []indexRecord
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var record indexRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("parse index line %q: %v", line, err)
		}
		records = append(records, record)
	}
	return records
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
