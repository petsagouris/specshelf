package vault

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A vault the tool created and maintains is healthy: the layout is present,
// nothing to read twice, nothing whose kind disagrees with its directory,
// nothing whose id or file name disagrees with its title, and therefore no
// problems.
func TestValidateHealthyVault(t *testing.T) {
	t.Run("a healthy vault", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
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
		if len(v.Missing) != 0 || len(v.Unreadable) != 0 || len(v.Mismatches) != 0 || len(v.Identities) != 0 {
			t.Errorf("Validate = %+v, want no findings", v)
		}
	})

	t.Run("an empty vault", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		v, err := Validate(root)
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
	if err := initVault(root); err != nil {
		t.Fatalf("initVault: %v", err)
	}
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
	if err := initVault(root); err != nil {
		t.Fatalf("initVault: %v", err)
	}
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

// A vault missing a directory or file that initialising a vault creates is a
// finding, counted by Problems; an extra directory or file the initialiser does
// not create is none of the validator's business.
func TestValidateReportsMissingLayout(t *testing.T) {
	t.Run("a missing README and templates are findings", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(filepath.Join(root, "templates")); err != nil {
			t.Fatal(err)
		}

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		want := map[string]bool{
			"templates":             false,
			"templates/spec.md":     false,
			"templates/decision.md": false,
			"README.md":             false,
		}
		if len(v.Missing) != len(want) {
			t.Fatalf("Missing = %+v, want %d findings", v.Missing, len(want))
		}
		for _, finding := range v.Missing {
			if _, ok := want[finding.Path]; !ok {
				t.Errorf("unexpected missing finding %q", finding.Path)
				continue
			}
			want[finding.Path] = true
			if !strings.Contains(finding.Text(), finding.Path) {
				t.Errorf("Text = %q, want it to name %q", finding.Text(), finding.Path)
			}
		}
		for path, seen := range want {
			if !seen {
				t.Errorf("%s was not reported as missing", path)
			}
		}
		if v.Problems() != 4 {
			t.Errorf("Problems = %d, want 4 (doctor exits non-zero for them)", v.Problems())
		}
	})

	t.Run("an extra directory or file is not a finding", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		writeDoc(t, root, "archive/2026-10-17-extra.md", "not a document\n")

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(v.Missing) != 0 {
			t.Errorf("Missing = %+v, want none: the validator owns only what the initialiser creates", v.Missing)
		}
	})
}

// A document whose declared id, or whose file name, does not match its title is
// a finding; the legitimate -N collision suffix the exclusive-create retry adds
// is not. The expectation is derived from slugify/documentID/documentPath, with
// the creation date taken from the file name.
func TestValidateReportsIdentityMismatch(t *testing.T) {
	newVault := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		return root
	}

	t.Run("an id that disagrees with the title", func(t *testing.T) {
		root := newVault(t)
		writeDoc(t, root, "specs/2026-10-17-note.md",
			"---\nid: spec-20261017-wrong\nkind: spec\ntitle: Note\n---\n# Note\n")

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(v.Identities) != 1 {
			t.Fatalf("Identities = %+v, want exactly the id finding", v.Identities)
		}
		got := v.Identities[0]
		if got.Field != "id" || got.Got != "spec-20261017-wrong" || got.Want != "spec-20261017-note" {
			t.Errorf("finding = %+v, want the id and the title-derived id", got)
		}
		for _, part := range []string{got.Path, "id", got.Got, got.Want} {
			if !strings.Contains(got.Text(), part) {
				t.Errorf("Text = %q, want it to contain %q", got.Text(), part)
			}
		}
		if v.Problems() != 1 {
			t.Errorf("Problems = %d, want 1 (doctor exits non-zero for it)", v.Problems())
		}
	})

	t.Run("a file name that disagrees with the title", func(t *testing.T) {
		root := newVault(t)
		writeDoc(t, root, "specs/2026-10-17-other.md",
			"---\nid: spec-20261017-note\nkind: spec\ntitle: Note\n---\n# Note\n")

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(v.Identities) != 1 {
			t.Fatalf("Identities = %+v, want exactly the file name finding", v.Identities)
		}
		got := v.Identities[0]
		if got.Field != "file name" || got.Got != "2026-10-17-other.md" || got.Want != "2026-10-17-note.md" {
			t.Errorf("finding = %+v, want the name and the title-derived name", got)
		}
		if v.Problems() != 1 {
			t.Errorf("Problems = %d, want 1 (doctor exits non-zero for it)", v.Problems())
		}
	})

	t.Run("a -N collision is not a finding", func(t *testing.T) {
		root := newVault(t)
		// A second document of one title gets -2 on both its id and its name.
		// That is the exclusive-create collision marker (D6), not a mismatch.
		writeDoc(t, root, "specs/2026-10-17-note-2.md",
			"---\nid: spec-20261017-note-2\nkind: spec\ntitle: Note\n---\n# Note\n")

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(v.Identities) != 0 {
			t.Errorf("Identities = %+v, want none for a -2 collision", v.Identities)
		}
	})

	t.Run("a file name with no date prefix is a finding", func(t *testing.T) {
		root := newVault(t)
		writeDoc(t, root, "specs/note.md",
			"---\nid: spec-20261017-note\nkind: spec\ntitle: Note\n---\n# Note\n")

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(v.Identities) != 1 || v.Identities[0].Field != "file name" || v.Identities[0].Want != "YYYY-MM-DD-note.md" {
			t.Fatalf("Identities = %+v, want the undated name reported against the dated form", v.Identities)
		}
	})

	t.Run("the date comes from the file name, not created", func(t *testing.T) {
		root := newVault(t)
		// The id and the name agree on 2026-10-16; created says otherwise and
		// is deliberately not consulted.
		writeDoc(t, root, "specs/2026-10-16-note.md",
			"---\nid: spec-20261016-note\nkind: spec\ntitle: Note\ncreated: 2026-10-17\n---\n# Note\n")

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(v.Identities) != 0 {
			t.Errorf("Identities = %+v, want none: created is not consulted", v.Identities)
		}
	})

	t.Run("a document that declares no id is not an identity finding", func(t *testing.T) {
		root := newVault(t)
		writeDoc(t, root, "specs/2026-10-17-note.md",
			"---\nkind: spec\ntitle: Note\n---\n# Note\n")

		v, err := Validate(root)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(v.Identities) != 0 {
			t.Errorf("Identities = %+v, want none: there is no declared id to disagree", v.Identities)
		}
	})
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
