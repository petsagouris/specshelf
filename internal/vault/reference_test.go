package vault

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceResolution(t *testing.T) {
	root := t.TempDir()
	const (
		id       = "spec-20261017-alpha"
		fileName = "2026-10-17-alpha.md"
		stemName = "2026-10-17-alpha"
		rel      = "specs/" + fileName
	)
	writeDoc(t, root, rel, "---\nid: "+id+"\nkind: spec\ntitle: Alpha\n---\n# Alpha\n")
	full := filepath.Join(root, filepath.FromSlash(rel))

	cases := []struct{ name, ref string }{
		{"existing path", full},
		{"id", id},
		{"file name", fileName},
		{"stem", stemName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc, err := ResolveReference(root, c.ref)
			if err != nil {
				t.Fatalf("ResolveReference(%q): %v", c.ref, err)
			}
			if doc.ID() != id || doc.Title() != "Alpha" {
				t.Errorf("ResolveReference(%q) = id %q title %q, want %q and %q",
					c.ref, doc.ID(), doc.Title(), id, "Alpha")
			}
		})
	}

	t.Run("unknown reference", func(t *testing.T) {
		_, err := ResolveReference(root, "no-such-reference")
		var unknown *UnknownReferenceError
		if !errors.As(err, &unknown) {
			t.Fatalf("ResolveReference error = %v, want *UnknownReferenceError", err)
		}
		if unknown.Reference != "no-such-reference" || !strings.Contains(err.Error(), "no-such-reference") {
			t.Errorf("error %q must name the reference", err.Error())
		}
	})

	t.Run("ambiguous reference", func(t *testing.T) {
		// The same stem in both directories matches at the id level for neither
		// document's own file name alone, so it is genuinely ambiguous.
		other := t.TempDir()
		writeDoc(t, other, "specs/x.md", "# x\n")
		writeDoc(t, other, "decisions/x.md", "# x\n")

		_, err := ResolveReference(other, "x")
		var ambiguous *AmbiguousReferenceError
		if !errors.As(err, &ambiguous) {
			t.Fatalf("ResolveReference error = %v, want *AmbiguousReferenceError", err)
		}
		if len(ambiguous.Matches) != 2 || !strings.Contains(err.Error(), "x") {
			t.Errorf("Matches = %v (error %q), want two matches naming the reference", ambiguous.Matches, err.Error())
		}
	})
}
