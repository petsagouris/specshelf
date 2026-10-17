package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/petsagouris/specshelf/internal/docs"
	"github.com/petsagouris/specshelf/internal/frontmatter"
)

func TestNormaliseTitle(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"surrounding whitespace", "  trim me  ", "trim me"},
		{"inner run collapse", "a   b\t\tc", "a b c"},
		{"surrounding and inner together", "\n  a  b  \t\n", "a b"},
		{"already normal", "Quality analysis", "Quality analysis"},
		{"whitespace only", " \t\n ", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := normaliseTitle(c.in); got != c.want {
				t.Errorf("normaliseTitle(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestSeedBody(t *testing.T) {
	t.Run("spec body from the spec template", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		body, err := seedBody(root, "spec", "Quality analysis")
		if err != nil {
			t.Fatalf("seedBody: %v", err)
		}
		want := strings.ReplaceAll(docs.SpecTemplate, "{title}", "Quality analysis")
		if body != want {
			t.Errorf("spec body = %q, want %q", body, want)
		}
	})

	t.Run("decision body from the decision template", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		body, err := seedBody(root, "decision", "Adopt cobra")
		if err != nil {
			t.Fatalf("seedBody: %v", err)
		}
		want := strings.ReplaceAll(docs.DecisionTemplate, "{title}", "Adopt cobra")
		if body != want {
			t.Errorf("decision body = %q, want %q", body, want)
		}
	})

	t.Run("deleted template is recreated and used", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		path := filepath.Join(root, "templates", "spec.md")
		if err := os.Remove(path); err != nil {
			t.Fatalf("remove template: %v", err)
		}

		body, err := seedBody(root, "spec", "Restored")
		if err != nil {
			t.Fatalf("seedBody: %v", err)
		}
		want := strings.ReplaceAll(docs.SpecTemplate, "{title}", "Restored")
		if body != want {
			t.Errorf("body = %q, want the default template %q", body, want)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read recreated template: %v", err)
		}
		if string(got) != docs.SpecTemplate {
			t.Errorf("recreated template = %q, want %q", got, docs.SpecTemplate)
		}
	})

	t.Run("edited template survives and is used", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		const custom = "CUSTOM {title}\n"
		path := filepath.Join(root, "templates", "spec.md")
		if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
			t.Fatal(err)
		}

		body, err := seedBody(root, "spec", "Mine")
		if err != nil {
			t.Fatalf("seedBody: %v", err)
		}
		if body != "CUSTOM Mine\n" {
			t.Errorf("body = %q, want %q", body, "CUSTOM Mine\n")
		}
	})

	t.Run("whitespace title normalised for heading and frontmatter, raw for id and name", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		const raw = "  Quality  analysis\tdisplay contract \n"
		const wantTitle = "Quality analysis display contract"
		const date = "2026-10-17"
		dir := filepath.Join(root, "specs")

		body, err := seedBody(root, "spec", raw)
		if err != nil {
			t.Fatalf("seedBody: %v", err)
		}
		heading, _, _ := strings.Cut(body, "\n")
		if heading != "# "+wantTitle {
			t.Errorf("body heading = %q, want %q", heading, "# "+wantTitle)
		}

		// The frontmatter value the creation path writes, rendered and read back
		// with the real codec, must carry the same normalised title.
		doc := frontmatter.Document{
			Metadata: frontmatter.Metadata{
				"id":    documentID("spec", date, slugify(raw)),
				"kind":  "spec",
				"title": normaliseTitle(raw),
			},
			Body: body,
		}
		parsed, err := frontmatter.Parse("sample.md", frontmatter.Marshal(doc))
		if err != nil {
			t.Fatalf("parse rendered document: %v", err)
		}
		if parsed.Title() != wantTitle {
			t.Errorf("frontmatter title = %q, want %q", parsed.Title(), wantTitle)
		}
		if parsed.Title() != strings.TrimPrefix(heading, "# ") {
			t.Errorf("frontmatter title %q and heading %q differ", parsed.Title(), heading)
		}

		// The id and the file name come from the title as given.
		if got := slugify(raw); got != "quality-analysis-display-contract" {
			t.Errorf("slugify(%q) = %q, want %q", raw, got, "quality-analysis-display-contract")
		}
		if got := documentID("spec", date, slugify(raw)); got != "spec-20261017-quality-analysis-display-contract" {
			t.Errorf("documentID = %q, want spec-20261017-quality-analysis-display-contract", got)
		}
		wantPath := filepath.Join(dir, "2026-10-17-quality-analysis-display-contract.md")
		if got := documentPath(dir, date, slugify(raw)); got != wantPath {
			t.Errorf("documentPath = %q, want %q", got, wantPath)
		}
	})

	t.Run("unknown kind is an error", func(t *testing.T) {
		root := t.TempDir()
		if err := initVault(root); err != nil {
			t.Fatalf("initVault: %v", err)
		}
		if _, err := seedBody(root, "note", "Nope"); err == nil {
			t.Error("seedBody with an unknown kind = nil error, want one")
		}
	})
}
