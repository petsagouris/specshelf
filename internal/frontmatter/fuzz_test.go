package frontmatter

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"
)

// FuzzReadFrontmatter asserts the reader's two invariants over arbitrary input:
// it never panics, and any document it reads re-emits frontmatter that it reads
// back with the same values a caller sees. An input it cannot read (an
// unterminated block) is a documented outcome, not a failure.
func FuzzReadFrontmatter(f *testing.F) {
	for _, vault := range []string{"python_vault", "python_vault_rewritten"} {
		root := filepath.Join("..", "..", "testdata", vault)
		for _, dir := range []string{"specs", "decisions"} {
			entries, err := os.ReadDir(filepath.Join(root, dir))
			if err != nil {
				f.Fatalf("reading %s: %v", dir, err)
			}
			for _, entry := range entries {
				data, err := os.ReadFile(filepath.Join(root, dir, entry.Name()))
				if err != nil {
					f.Fatalf("reading %s: %v", entry.Name(), err)
				}
				f.Add(data)
			}
		}
	}

	for _, seed := range []string{
		"---\n---\n",
		"---\ntitle: x\n",
		"title: \"a\\nb\"",
		"# a body-only document\n\nwith no frontmatter\n",
		"---\ntitle: [\ntags:\n\t- a tabbed item\n---\nbody\n",
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		before, err := Parse("specs/fuzz.md", data)
		if err != nil {
			return // an unreadable document is a documented, non-panicking outcome
		}

		marshalled := Marshal(before)
		after, err := Parse("specs/fuzz.md", marshalled)
		if err != nil {
			t.Fatalf("re-parsing our own output failed: %v\ninput: %q\nmarshalled: %q", err, data, marshalled)
		}

		// Invalid UTF-8 is replaced while quoting, so only the re-parse
		// invariant holds for it; the values are compared for valid input.
		if !utf8.Valid(data) {
			return
		}

		for _, c := range []struct {
			name      string
			got, want string
		}{
			{"ID", after.ID(), before.ID()},
			{"Kind", after.Kind(), before.Kind()},
			{"Title", after.Title(), before.Title()},
			{"Status", after.Status(), before.Status()},
			{"Created", after.Created(), before.Created()},
			{"Updated", after.Updated(), before.Updated()},
			{"Body", after.Body, before.Body},
		} {
			if c.got != c.want {
				t.Fatalf("%s changed across a rewrite: %q, want %q\ninput: %q\nmarshalled: %q", c.name, c.got, c.want, data, marshalled)
			}
		}
		for _, c := range []struct {
			name      string
			got, want []string
		}{
			{"Repos", after.Repos(), before.Repos()},
			{"Tags", after.Tags(), before.Tags()},
			{"CodexSessions", after.CodexSessions(), before.CodexSessions()},
		} {
			if !sameStrings(c.got, c.want) {
				t.Fatalf("%s changed across a rewrite: %q, want %q\ninput: %q\nmarshalled: %q", c.name, c.got, c.want, data, marshalled)
			}
		}
	})
}
