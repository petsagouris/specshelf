package vault

import (
	"testing"
)

// Query ANDs the filters and orders by updated descending, then id descending.
func TestQueryFiltersAndOrders(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-b.md", "---\nid: spec-20261017-b\nkind: spec\ntitle: B\nstatus: active\nupdated: 2026-10-17\nrepos:\n  - billing\ntags:\n  - tooling\ncodex_sessions:\n  - s1\n---\n# B\n")
	writeDoc(t, root, "specs/2026-10-17-a.md", "---\nid: spec-20261017-a\nkind: spec\ntitle: A\nstatus: active\nupdated: 2026-10-17\nrepos:\n  - billing\n---\n# A\n")
	writeDoc(t, root, "specs/2026-10-16-c.md", "---\nid: spec-20261016-c\nkind: spec\ntitle: C\nstatus: draft\nupdated: 2026-10-16\n---\n# C\n")
	writeDoc(t, root, "decisions/2026-10-17-d.md", "---\nid: decision-20261017-d\nkind: decision\ntitle: D\nstatus: active\nupdated: 2026-10-17\ntags:\n  - tooling\n---\n# D\n")

	all, warnings, err := Query(root, Filters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
	var ids []string
	for _, record := range all {
		ids = append(ids, record.ID)
	}
	// 2026-10-17 tie broken by id descending ("spec-" sorts above "decision-"),
	// then 2026-10-16.
	want := []string{"spec-20261017-b", "spec-20261017-a", "decision-20261017-d", "spec-20261016-c"}
	for i, id := range want {
		if ids[i] != id {
			t.Fatalf("order = %v, want %v", ids, want)
		}
	}

	cases := []struct {
		name string
		f    Filters
		want []string
	}{
		{"kind", Filters{Kind: "decision"}, []string{"decision-20261017-d"}},
		{"kind all", Filters{Kind: "all"}, want},
		{"repo", Filters{Repo: "billing"}, []string{"spec-20261017-b", "spec-20261017-a"}},
		{"tag", Filters{Tag: "tooling"}, []string{"spec-20261017-b", "decision-20261017-d"}},
		{"session", Filters{CodexSession: "s1"}, []string{"spec-20261017-b"}},
		{"status", Filters{Status: "draft"}, []string{"spec-20261016-c"}},
		{"anded", Filters{Kind: "spec", Status: "active"}, []string{"spec-20261017-b", "spec-20261017-a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			records, _, err := Query(root, tc.f)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, record := range records {
				got = append(got, record.ID)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ids = %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("ids = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// Search reports a metadata match against the title with line 0, and a body match
// at the body line it first appears on.
func TestSearchMetadataAndBody(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-quality.md", "---\nid: spec-20261017-quality\nkind: spec\ntitle: Quality analysis\nstatus: active\nupdated: 2026-10-17\n---\n# Quality analysis\n\nintro line\n\nThe needle sits here.\n")
	writeDoc(t, root, "decisions/2026-10-17-beads.md", "---\nid: decision-20261017-beads\nkind: decision\ntitle: Use Markdown vault instead of Beads\nstatus: active\nupdated: 2026-10-17\n---\n# Use Markdown vault instead of Beads\n")

	matches, _, err := Search(root, "quality", Filters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("matches = %+v, want one", matches)
	}
	if m := matches[0]; m.ID != "spec-20261017-quality" || m.Line != 0 || m.Snippet != "Quality analysis" {
		t.Errorf("metadata match = %+v, want the title snippet at line 0", m)
	}

	matches, _, err = Search(root, "needle", Filters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("matches = %+v, want one", matches)
	}
	if m := matches[0]; m.Line != 5 || m.Snippet != "The needle sits here." {
		t.Errorf("body match = %+v, want line 5", m)
	}

	matches, _, err = Search(root, "beads", Filters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != "decision-20261017-beads" || matches[0].Line != 0 {
		t.Errorf("matches = %+v, want the decision matched on its title", matches)
	}
}
