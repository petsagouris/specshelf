package vault

import (
	"maps"
	"strings"
	"testing"
)

// A vault the tool created and maintains is healthy: nothing to read twice,
// nothing whose kind disagrees with its directory, and therefore no problems.
func TestValidateHealthyVault(t *testing.T) {
	t.Run("a healthy vault", func(t *testing.T) {
		root := t.TempDir()
		writeDoc(t, root, "specs/2026-10-17-a.md",
			"---\nid: spec-20261017-a\nkind: spec\ntitle: A\nstatus: active\n---\n# A\n")
		writeDoc(t, root, "decisions/2026-10-17-b.md",
			"---\nid: decision-20261017-b\nkind: decision\ntitle: B\nstatus: accepted\n---\n# B\n")
		// No kind field: the reader classifies it by its directory, so there is
		// nothing declared for the directory to disagree with.
		writeDoc(t, root, "specs/2026-10-17-c.md",
			"---\nid: spec-20261017-c\ntitle: C\n---\n# C\n")

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if v.Problems() != 0 {
			t.Errorf("Problems = %d, want 0 for a healthy vault: %+v", v.Problems(), v)
		}
		if len(v.Unreadable) != 0 || len(v.Mismatches) != 0 {
			t.Errorf("Validate = %+v, want no findings", v)
		}
	})

	t.Run("an empty vault", func(t *testing.T) {
		v, err := Validate(t.TempDir())
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if v.Problems() != 0 {
			t.Errorf("Problems = %d, want 0 for an empty vault", v.Problems())
		}
	})
}

// A document that cannot be parsed is a finding, named by path with its reason,
// while the readable documents are unaffected.
func TestValidateReportsUnreadableDocument(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-good.md",
		"---\nid: spec-20261017-good\nkind: spec\ntitle: Good\n---\n# Good\n")
	writeDoc(t, root, "specs/2026-10-17-broken.md",
		"---\nid: spec-20261017-broken\nkind: spec\ntitle: Broken\n")

	v, err := Validate(root)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(v.Unreadable) != 1 {
		t.Fatalf("Unreadable = %+v, want exactly one finding", v.Unreadable)
	}
	if got := v.Unreadable[0]; got.Path != "specs/2026-10-17-broken.md" || !strings.Contains(got.Reason, "unterminated frontmatter") {
		t.Errorf("finding = %+v, want the bad path and its parse reason", got)
	}
	if len(v.Mismatches) != 0 {
		t.Errorf("Mismatches = %+v, want none", v.Mismatches)
	}
	if v.Problems() != 1 {
		t.Errorf("Problems = %d, want 1 (doctor exits non-zero for it)", v.Problems())
	}
}

// A document whose declared kind disagrees with its directory is a finding. The
// proof that this reads the declaration rather than the reader's classification
// is the document with no kind field: it is classified as a spec by its
// directory but is not reported.
func TestValidateReportsKindMismatch(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-misplaced.md",
		"---\nid: spec-20261017-misplaced\nkind: decision\ntitle: Misplaced\n---\n# Misplaced\n")
	writeDoc(t, root, "decisions/2026-10-17-in-place.md",
		"---\nid: decision-20261017-in-place\nkind: decision\ntitle: In place\n---\n# In place\n")
	writeDoc(t, root, "specs/2026-10-17-undeclared.md",
		"---\nid: spec-20261017-undeclared\ntitle: Undeclared\n---\n# Undeclared\n")

	v, err := Validate(root)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(v.Mismatches) != 1 {
		t.Fatalf("Mismatches = %+v, want exactly the one misplaced document", v.Mismatches)
	}
	got := v.Mismatches[0]
	if got.Path != "specs/2026-10-17-misplaced.md" || got.Declared != "decision" || got.Directory != "specs" {
		t.Errorf("mismatch = %+v, want the misplaced document and both kinds", got)
	}
	for _, want := range []string{got.Path, `"decision"`, "specs directory"} {
		if !strings.Contains(got.Text(), want) {
			t.Errorf("Text = %q, want it to contain %q", got.Text(), want)
		}
	}
	if v.Problems() != 1 {
		t.Errorf("Problems = %d, want 1 (doctor exits non-zero for it)", v.Problems())
	}
}

// Stats counts documents grouped by kind, status, repo and tag. A document with
// several repos or tags counts in each group, an unreadable document is a
// warning and contributes nothing, and the status groups include the empty one.
func TestStatsGroupsCounts(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-a.md",
		"---\nid: spec-20261017-a\nkind: spec\ntitle: A\nstatus: active\nrepos:\n  - r1\n  - r2\ntags:\n  - t1\n---\n# A\n")
	writeDoc(t, root, "specs/2026-10-17-b.md",
		"---\nid: spec-20261017-b\nkind: spec\ntitle: B\nstatus: active\nrepos:\n  - r2\ntags:\n  - t1\n  - t2\n---\n# B\n")
	writeDoc(t, root, "decisions/2026-10-17-c.md",
		"---\nid: decision-20261017-c\nkind: decision\ntitle: C\nstatus: accepted\nrepos:\n  - r1\n---\n# C\n")
	writeDoc(t, root, "specs/2026-10-17-d.md",
		"---\nid: spec-20261017-d\nkind: spec\ntitle: D\n---\n# D\n")
	writeDoc(t, root, "specs/2026-10-17-broken.md",
		"---\nid: spec-20261017-broken\nkind: spec\ntitle: Broken\n")

	stats, warnings, err := Stats(root)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if len(warnings) != 1 || warnings[0].Path != "specs/2026-10-17-broken.md" {
		t.Errorf("warnings = %+v, want the one unreadable document", warnings)
	}
	if stats.Documents != 4 {
		t.Errorf("Documents = %d, want the 4 readable documents", stats.Documents)
	}
	if want := map[string]int{"spec": 3, "decision": 1}; !maps.Equal(stats.ByKind, want) {
		t.Errorf("ByKind = %v, want %v", stats.ByKind, want)
	}
	if want := map[string]int{"active": 2, "accepted": 1, "": 1}; !maps.Equal(stats.ByStatus, want) {
		t.Errorf("ByStatus = %v, want %v", stats.ByStatus, want)
	}
	if want := map[string]int{"r1": 2, "r2": 2}; !maps.Equal(stats.ByRepo, want) {
		t.Errorf("ByRepo = %v, want %v", stats.ByRepo, want)
	}
	if want := map[string]int{"t1": 2, "t2": 1}; !maps.Equal(stats.ByTag, want) {
		t.Errorf("ByTag = %v, want %v", stats.ByTag, want)
	}
}
