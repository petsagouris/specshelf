package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// The index must follow create, edit and remove on its own: each mutation is
// followed by the refresh a command runs, never by a separate `specshelf index`.
func TestWriteIndexFollowsMutation(t *testing.T) {
	root := t.TempDir()
	first := "specs/2026-10-17-first.md"
	second := "specs/2026-10-17-second.md"
	writeDoc(t, root, first, "---\nid: spec-20261017-first\nkind: spec\ntitle: First\n---\n# First\n\nThe first document.\n")
	writeDoc(t, root, second, "---\nid: spec-20261017-second\nkind: spec\ntitle: Second\n---\n# Second\n\nThe second document.\n")

	t.Run("create", func(t *testing.T) {
		if err := WriteIndex(root); err != nil {
			t.Fatalf("WriteIndex: %v", err)
		}
		assertIndexMatchesVault(t, root)
		assertIndexContains(t, root, "spec-20261017-first", "spec-20261017-second")
	})

	t.Run("edit", func(t *testing.T) {
		edited := "---\nid: spec-20261017-first\nkind: spec\ntitle: First\nupdated: 2026-10-18\n---\n# First\n\nA rewritten summary.\n"
		if err := WriteDocument(filepath.Join(root, first), []byte(edited)); err != nil {
			t.Fatalf("WriteDocument: %v", err)
		}
		if err := WriteIndex(root); err != nil {
			t.Fatalf("WriteIndex: %v", err)
		}
		assertIndexMatchesVault(t, root)
		assertIndexContains(t, root, "A rewritten summary.")
	})

	t.Run("remove", func(t *testing.T) {
		if err := os.Remove(filepath.Join(root, second)); err != nil {
			t.Fatal(err)
		}
		if err := WriteIndex(root); err != nil {
			t.Fatalf("WriteIndex: %v", err)
		}
		assertIndexMatchesVault(t, root)
		if got := string(mustRead(t, filepath.Join(root, "index.jsonl"))); bytes.Contains([]byte(got), []byte("spec-20261017-second")) {
			t.Errorf("the removed document is still indexed:\n%s", got)
		}
	})
}

// Regenerating the index is itself an atomic write, so a failure leaves the
// previous index readable and no temporary file behind.
func TestWriteIndexIsAtomic(t *testing.T) {
	root := t.TempDir()
	writeDoc(t, root, "specs/2026-10-17-one.md", "---\nid: spec-20261017-one\nkind: spec\ntitle: One\n---\n# One\n\nBody.\n")
	if err := WriteIndex(root); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}
	indexPath := filepath.Join(root, "index.jsonl")
	previous := mustRead(t, indexPath)

	interruptRename(t)

	if err := WriteIndex(root); err == nil {
		t.Fatal("WriteIndex succeeded although the rename failed")
	}
	if got := mustRead(t, indexPath); !bytes.Equal(got, previous) {
		t.Errorf("index after the failed refresh = %q, want the previous %q", got, previous)
	}
	assertOnlyFile(t, root, "specs", "index.jsonl")
}

func assertIndexMatchesVault(t *testing.T, root string) {
	t.Helper()
	records, err := Index(root)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	want := EncodeIndex(records)
	got := mustRead(t, filepath.Join(root, "index.jsonl"))
	if !bytes.Equal(got, want) {
		t.Errorf("index.jsonl does not match the vault:\n got %q\nwant %q", got, want)
	}
}

func assertIndexContains(t *testing.T, root string, substrings ...string) {
	t.Helper()
	got := mustRead(t, filepath.Join(root, "index.jsonl"))
	for _, want := range substrings {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("index.jsonl does not contain %q:\n%s", want, got)
		}
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
