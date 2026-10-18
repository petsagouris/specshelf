package cli

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// formatVault lays down the two-document vault the format tests read: one spec
// carrying every list field, one decision. Both share an updated date, so the
// order is id descending the way the reference sorts.
func formatVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	docs := map[string]string{
		"specs/2026-10-17-alpha.md": "---\nid: spec-20261017-alpha\nkind: spec\ntitle: Alpha\nstatus: active\n" +
			"created: 2026-10-17\nupdated: 2026-10-17\nrepos:\n  - billing\ntags:\n  - tooling\n" +
			"codex_sessions:\n  - s1\n---\n# Alpha\n\nline two\n\nneedle here\n",
		"decisions/2026-10-17-gamma.md": "---\nid: decision-20261017-gamma\nkind: decision\ntitle: Gamma\nstatus: accepted\n" +
			"created: 2026-10-17\nupdated: 2026-10-17\ntags:\n  - tooling\n---\n# Gamma\n",
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
	t.Setenv("SPECSHELF_CONFIG", filepath.Join(t.TempDir(), "config.toml"))
	return root
}

// runCLI runs one invocation against explicit streams and returns stdout, stderr
// and the exit code.
func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

// jsonKeys returns a JSON object's keys in order, which is what "the same field
// names as the index" is asserted against.
func jsonKeys(t *testing.T, line string) []string {
	t.Helper()
	matches := regexp.MustCompile(`"([a-z_]+)": `).FindAllStringSubmatch(line, -1)
	keys := make([]string, len(matches))
	for i, m := range matches {
		keys[i] = m[1]
	}
	return keys
}

// indexFieldNames is the field set of an index.jsonl record, in its written
// order. It is duplicated here, not derived, so a change to the encoder shows up
// as a failure.
var indexFieldNames = []string{"id", "kind", "title", "status", "created", "updated", "repos", "tags", "codex_sessions", "path", "summary"}

// listFieldNames is the tabular column order, which csv and table must share.
var listFieldNames = []string{"id", "kind", "status", "updated", "repos", "tags", "codex_sessions", "path", "title"}

// TestListFormatFieldNameParity pins the contract: jsonl carries the index
// record's field names; csv and table carry those columns in the TSV order. The
// index file itself is the oracle for jsonl.
func TestListFormatFieldNameParity(t *testing.T) {
	root := formatVault(t)

	if _, stderr, code := runCLI(t, "--home", root, "index"); code != 0 {
		t.Fatalf("index exit = %d, stderr = %s", code, stderr)
	}
	indexBytes, err := os.ReadFile(filepath.Join(root, "index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	jsonl, stderr, code := runCLI(t, "--home", root, "list", "--json")
	if code != 0 {
		t.Fatalf("list --json exit = %d, stderr = %s", code, stderr)
	}
	if jsonl != string(indexBytes) {
		t.Errorf("list --json = %q, want the index file's bytes %q", jsonl, indexBytes)
	}
	for _, line := range strings.Split(strings.TrimSuffix(jsonl, "\n"), "\n") {
		if got := jsonKeys(t, line); !reflect.DeepEqual(got, indexFieldNames) {
			t.Errorf("list --json keys = %v, want %v", got, indexFieldNames)
		}
	}

	formatJSONLOut, _, code := runCLI(t, "--home", root, "list", "--format", "jsonl")
	if code != 0 {
		t.Fatalf("list --format jsonl exit = %d", code)
	}
	if formatJSONLOut != jsonl {
		t.Errorf("--format jsonl = %q, want it to equal --json %q", formatJSONLOut, jsonl)
	}

	csvOut, stderr, code := runCLI(t, "--home", root, "list", "--format", "csv")
	if code != 0 {
		t.Fatalf("list --format csv exit = %d, stderr = %s", code, stderr)
	}
	rows, err := csv.NewReader(strings.NewReader(csvOut)).ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v", err)
	}
	if len(rows) == 0 || !reflect.DeepEqual(rows[0], listFieldNames) {
		t.Errorf("csv header = %v, want %v", rows[0], listFieldNames)
	}
	// The four formats must agree on the values, so a CSV row is the TSV row
	// split on tabs.
	tsvOut, _, _ := runCLI(t, "--home", root, "list")
	tsvRows := strings.Split(strings.TrimSuffix(tsvOut, "\n"), "\n")
	if len(rows)-1 != len(tsvRows) {
		t.Fatalf("csv has %d data rows, tsv has %d", len(rows)-1, len(tsvRows))
	}
	for i, row := range rows[1:] {
		if want := strings.Split(tsvRows[i], "\t"); !reflect.DeepEqual(row, want) {
			t.Errorf("csv row %d = %v, want the tsv cells %v", i, row, want)
		}
	}

	tableOut, stderr, code := runCLI(t, "--home", root, "list", "--format", "table")
	if code != 0 {
		t.Fatalf("list --format table exit = %d, stderr = %s", code, stderr)
	}
	header, _, _ := strings.Cut(tableOut, "\n")
	for _, name := range listFieldNames {
		if !strings.Contains(header, name) {
			t.Errorf("table header %q does not name column %q", header, name)
		}
	}
}

// TestListDefaultFormatIsTSV pins the default the reference fixes: no flag and
// --format tsv are the same bytes, with nine tab-separated columns.
func TestListDefaultFormatIsTSV(t *testing.T) {
	root := formatVault(t)

	def, _, code := runCLI(t, "--home", root, "list")
	if code != 0 {
		t.Fatalf("list exit = %d", code)
	}
	explicit, _, _ := runCLI(t, "--home", root, "list", "--format", "tsv")
	if def != explicit {
		t.Errorf("default list = %q, want --format tsv %q", def, explicit)
	}
	for _, line := range strings.Split(strings.TrimSuffix(def, "\n"), "\n") {
		if got := strings.Count(line, "\t") + 1; got != 9 {
			t.Errorf("tsv line %q has %d columns, want 9", line, got)
		}
	}
}

// TestEmptyListPrintsNothing pins the empty-listing divergence: the reference
// prints a "No specs or decisions found." line on stdout, which a caller parsing
// one row per line would read as a document, while specshelf prints nothing
// (task 3.13).
func TestEmptyListPrintsNothing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SPECSHELF_CONFIG", filepath.Join(t.TempDir(), "config.toml"))

	for _, args := range [][]string{
		{"list"},
		{"list", "--json"},
	} {
		stdout, stderr, code := runCLI(t, append([]string{"--home", root}, args...)...)
		if code != 0 {
			t.Errorf("%v exit = %d, want 0\nstderr: %s", args, code, stderr)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q, want it empty", args, stdout)
		}
	}
}

// TestJSONConflictsWithFormat pins that --json and a disagreeing --format is a
// usage error rather than one silently winning.
func TestJSONConflictsWithFormat(t *testing.T) {
	root := formatVault(t)

	for _, format := range []string{"tsv", "csv", "table"} {
		_, stderr, code := runCLI(t, "--home", root, "list", "--json", "--format", format)
		if code != exitUsage {
			t.Errorf("list --json --format %s exit = %d, want %d", format, code, exitUsage)
		}
		if !strings.Contains(stderr, "cannot be combined") {
			t.Errorf("list --json --format %s stderr = %q, want the conflict message", format, stderr)
		}
	}
	if _, stderr, code := runCLI(t, "--home", root, "list", "--json", "--format", "jsonl"); code != 0 {
		t.Errorf("list --json --format jsonl exit = %d, stderr = %s", code, stderr)
	}
}

// TestTableFormatColourGating pins the colour rule at the table: with a
// non-terminal writer the default emits no escapes, and --color always forces
// them onto the header only.
func TestTableFormatColourGating(t *testing.T) {
	root := formatVault(t)

	def, _, code := runCLI(t, "--home", root, "list", "--format", "table")
	if code != 0 {
		t.Fatalf("table exit = %d", code)
	}
	if strings.Contains(def, "\x1b") {
		t.Errorf("piped table carries escapes: %q", def)
	}

	forced, _, code := runCLI(t, "--home", root, "list", "--format", "table", "--color", "always")
	if code != 0 {
		t.Fatalf("table --color always exit = %d", code)
	}
	if !strings.Contains(forced, "\x1b[1m") {
		t.Errorf("table --color always carries no bold header: %q", forced)
	}
}

// TestSearchFormats pins the search-shaped output: jsonl carries the index names
// id, kind and path plus line and snippet; csv shares the TSV cells.
func TestSearchFormats(t *testing.T) {
	root := formatVault(t)

	jsonl, stderr, code := runCLI(t, "--home", root, "search", "needle", "--json")
	if code != 0 {
		t.Fatalf("search --json exit = %d, stderr = %s", code, stderr)
	}
	want := `{"id":"spec-20261017-alpha","kind":"spec","path":"specs/2026-10-17-alpha.md","line":5,"snippet":"needle here"}` + "\n"
	if jsonl != want {
		t.Errorf("search --json = %q, want %q", jsonl, want)
	}

	csvOut, _, code := runCLI(t, "--home", root, "search", "needle", "--format", "csv")
	if code != 0 {
		t.Fatalf("search --format csv exit = %d", code)
	}
	rows, err := csv.NewReader(strings.NewReader(csvOut)).ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v", err)
	}
	if want := []string{"id", "kind", "location", "snippet"}; !reflect.DeepEqual(rows[0], want) {
		t.Errorf("search csv header = %v, want %v", rows[0], want)
	}
	if want := []string{"spec-20261017-alpha", "spec", "specs/2026-10-17-alpha.md:5", "needle here"}; len(rows) < 2 || !reflect.DeepEqual(rows[1], want) {
		t.Errorf("search csv row = %v, want %v", rows[1:], want)
	}
}

// TestShowAndPathJSON pin the single-object commands: show --json is the
// document's index record, path --json names the selector that won (D8).
func TestShowAndPathJSON(t *testing.T) {
	root := formatVault(t)

	if _, stderr, code := runCLI(t, "--home", root, "index"); code != 0 {
		t.Fatalf("index exit = %d, stderr = %s", code, stderr)
	}
	indexBytes, err := os.ReadFile(filepath.Join(root, "index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	wantLines := strings.Split(strings.TrimSuffix(string(indexBytes), "\n"), "\n")
	sort.Strings(wantLines)

	show, stderr, code := runCLI(t, "--home", root, "show", "spec-20261017-alpha", "--json")
	if code != 0 {
		t.Fatalf("show --json exit = %d, stderr = %s", code, stderr)
	}
	if trimmed := strings.TrimSuffix(show, "\n"); !containsLine(wantLines, trimmed) {
		t.Errorf("show --json = %q, which is not an index record", show)
	}

	path, stderr, code := runCLI(t, "--home", root, "path", "--json")
	if code != 0 {
		t.Fatalf("path --json exit = %d, stderr = %s", code, stderr)
	}
	want := `{"root":"` + root + `","source":"--home"}` + "\n"
	if path != want {
		t.Errorf("path --json = %q, want %q", path, want)
	}
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}
