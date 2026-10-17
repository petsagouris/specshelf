package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExportJSONAndJSONL pins the task's contract: every document appears exactly
// once, with the index record's fields, in both the JSON array and the JSONL
// form. The JSONL form is byte-identical to index.jsonl because it is the same
// encoder.
func TestExportJSONAndJSONL(t *testing.T) {
	root := formatVault(t)

	if _, stderr, code := runCLI(t, "--home", root, "index"); code != 0 {
		t.Fatalf("index exit = %d, stderr = %s", code, stderr)
	}
	indexBytes, err := os.ReadFile(filepath.Join(root, "index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	indexLines := strings.Split(strings.TrimSuffix(string(indexBytes), "\n"), "\n")

	jsonl, stderr, code := runCLI(t, "--home", root, "export", "--format", "jsonl")
	if code != 0 {
		t.Fatalf("export --format jsonl exit = %d, stderr = %s", code, stderr)
	}
	if jsonl != string(indexBytes) {
		t.Errorf("export --format jsonl = %q, want the index file %q", jsonl, indexBytes)
	}

	arr, stderr, code := runCLI(t, "--home", root, "export")
	if code != 0 {
		t.Fatalf("export exit = %d, stderr = %s", code, stderr)
	}
	var records []struct {
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
		Summary       string   `json:"summary"`
	}
	if err := json.Unmarshal([]byte(arr), &records); err != nil {
		t.Fatalf("export did not produce a JSON array: %v\n%s", err, arr)
	}
	if len(records) != len(indexLines) {
		t.Errorf("export has %d records, want %d (one per document)", len(records), len(indexLines))
	}
	seen := make(map[string]bool, len(records))
	for _, record := range records {
		if seen[record.ID] {
			t.Errorf("document %s exported twice", record.ID)
		}
		seen[record.ID] = true
	}
	for _, line := range indexLines {
		if !strings.Contains(arr, line) {
			t.Errorf("export array is missing the index record %q", line)
		}
	}

	if _, stderr, code := runCLI(t, "--home", root, "export", "--format", "tsv"); code != exitUsage {
		t.Errorf("export --format tsv exit = %d, want %d", code, exitUsage)
	} else if !strings.Contains(stderr, "must be json or jsonl") {
		t.Errorf("export --format tsv stderr = %q, want the format error", stderr)
	}
}
