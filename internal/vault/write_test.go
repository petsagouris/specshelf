package vault

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAtomicWriteReplacesInOneStep(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "doc.md")
		if err := os.WriteFile(path, []byte("previous bytes\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := WriteDocument(path, []byte("complete new content\n")); err != nil {
			t.Fatalf("WriteDocument: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "complete new content\n" {
			t.Errorf("content = %q, want the complete new content", got)
		}
		assertOnlyFile(t, dir, "doc.md")
	})

	t.Run("absent file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "new.md")

		if err := WriteDocument(path, []byte("first content\n")); err != nil {
			t.Fatalf("WriteDocument: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "first content\n" {
			t.Errorf("content = %q, want the complete new content", got)
		}
		assertOnlyFile(t, dir, "new.md")
	})
}

// An interruption between writing the temporary file and renaming it must leave
// the previous content in place and no temporary file behind.
func TestAtomicWriteInterruptedRenameLeavesOriginal(t *testing.T) {
	t.Run("target exists", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "doc.md")
		previous := []byte("previous bytes\n")
		if err := os.WriteFile(path, previous, 0o644); err != nil {
			t.Fatal(err)
		}

		interruptRename(t)

		if err := WriteDocument(path, []byte("never lands\n")); err == nil {
			t.Fatal("WriteDocument succeeded although the rename failed")
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, previous) {
			t.Errorf("target content = %q, want the previous bytes %q", got, previous)
		}
		assertOnlyFile(t, dir, "doc.md")
	})

	t.Run("target absent", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "doc.md")

		interruptRename(t)

		if err := WriteDocument(path, []byte("never lands\n")); err == nil {
			t.Fatal("WriteDocument succeeded although the rename failed")
		}
		if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("target exists after the failed write: %v", err)
		}
		assertOnlyFile(t, dir) // an empty directory: no temporary file left behind
	})
}

func TestExclusiveCreateRetriesNextFreeName(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "2026-10-17-same-title.md")

	first, err := CreateDocument(base, []byte("first\n"))
	if err != nil {
		t.Fatalf("first CreateDocument: %v", err)
	}
	if first != base {
		t.Errorf("first path = %q, want %q", first, base)
	}

	second, err := CreateDocument(base, []byte("second\n"))
	if err != nil {
		t.Fatalf("second CreateDocument: %v", err)
	}
	if want := filepath.Join(dir, "2026-10-17-same-title-2.md"); second != want {
		t.Errorf("second path = %q, want the next free name %q", second, want)
	}

	for path, want := range map[string]string{first: "first\n", second: "second\n"} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	assertOnlyFile(t, dir, "2026-10-17-same-title.md", "2026-10-17-same-title-2.md")
}

// The exclusive step itself: an existing file is an error, never a truncation.
func TestExclusiveCreateRefusesToTruncate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taken.md")
	original := []byte("original\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeExclusive(path, []byte("replacement\n"))
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("writeExclusive over an existing file = %v, want fs.ErrExist", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("existing file = %q, want it untouched %q", got, original)
	}
}

// Two creators racing for the same title must both succeed, each under its own
// name, leaving the vault with two distinct documents. With an exclusive create
// this holds for any interleaving.
func TestConcurrentCreationOfTheSameTitle(t *testing.T) {
	const creators = 16
	dir := t.TempDir()
	base := filepath.Join(dir, "2026-10-17-racing-title.md")

	start := make(chan struct{})
	paths := make([]string, creators)
	errs := make([]error, creators)
	var wg sync.WaitGroup
	for i := 0; i < creators; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			paths[i], errs[i] = CreateDocument(base, []byte(fmt.Sprintf("document %d\n", i)))
		}(i)
	}
	close(start)
	wg.Wait()

	seen := make(map[string]bool, creators)
	for i, err := range errs {
		if err != nil {
			t.Errorf("creator %d failed: %v", i, err)
			continue
		}
		if seen[paths[i]] {
			t.Errorf("two creators returned the same path %q", paths[i])
		}
		seen[paths[i]] = true
		if got, err := os.ReadFile(paths[i]); err != nil || string(got) != fmt.Sprintf("document %d\n", i) {
			t.Errorf("%s holds %q, %v; want document %d", paths[i], got, err, i)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != creators {
		t.Errorf("the vault holds %d documents, want %d distinct ones", len(entries), creators)
	}
}

// interruptRename makes the next run of this test's writes fail at the rename.
func interruptRename(t *testing.T) {
	t.Helper()
	restore := renameOver
	renameOver = func(_, _ string) error { return errors.New("simulated interruption") }
	t.Cleanup(func() { renameOver = restore })
}

// assertOnlyFile checks that dir holds exactly the named files and nothing else,
// which is how a test proves no temporary file survived a write.
func assertOnlyFile(t *testing.T, dir string, names ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(names) {
		t.Fatalf("%s holds %v, want exactly %v", dir, entryNames(entries), names)
	}
	for _, entry := range entries {
		found := false
		for _, name := range names {
			if entry.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s holds unexpected file %q", dir, entry.Name())
		}
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}
