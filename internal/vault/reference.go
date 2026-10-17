package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/petsagouris/specshelf/internal/frontmatter"
)

// UnknownReferenceError reports a reference that names no document. It carries
// no candidates: an unknown reference lists nothing rather than guessing.
type UnknownReferenceError struct {
	Reference string
}

func (e *UnknownReferenceError) Error() string {
	return fmt.Sprintf("unknown reference %q", e.Reference)
}

// AmbiguousReferenceError reports a reference that names more than one document
// at the same precedence level.
type AmbiguousReferenceError struct {
	Reference string
	Matches   []string
}

func (e *AmbiguousReferenceError) Error() string {
	return fmt.Sprintf("ambiguous reference %q: matches %s", e.Reference, strings.Join(e.Matches, ", "))
}

// OutsideVaultError reports a resolved document whose path is not inside the
// vault root. A deletion refuses it rather than touching a file the vault does
// not own, and names both the path and the root so the caller can report them.
type OutsideVaultError struct {
	Path string
	Root string
}

func (e *OutsideVaultError) Error() string {
	return fmt.Sprintf("%s is outside the vault %s", e.Path, e.Root)
}

// withinVault reports whether path lies inside root. Symbolic links are resolved
// on both, so a directory symlink inside the vault that escapes it is refused,
// while a symlink that itself lives inside the vault is allowed: removing it
// unlinks the link, not what it points at. A path whose symbolic links cannot be
// resolved falls back to its absolute form.
func withinVault(root, path string) bool {
	rel, err := filepath.Rel(resolvePath(root), resolvePath(filepath.Dir(path)))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolvePath makes path absolute and resolves its symbolic links, falling back
// to the absolute path when it cannot — a path that does not exist has nothing
// to delete, and the lexical form still bounds it.
func resolvePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// ResolveReference returns the document a user reference names, trying an
// existing path first, then a document id, then a file name, then a file stem.
// A reference matching no document is an UnknownReferenceError; one matching
// several documents at the same level is an AmbiguousReferenceError.
func ResolveReference(root, reference string) (frontmatter.Document, error) {
	if info, err := os.Stat(reference); err == nil && info.Mode().IsRegular() {
		return readDocument(reference)
	}
	records, err := Index(root)
	if err != nil {
		return frontmatter.Document{}, err
	}

	levels := []func(IndexRecord) string{
		func(r IndexRecord) string { return r.ID },
		func(r IndexRecord) string { return filepath.Base(r.Path) },
		func(r IndexRecord) string { return stem(filepath.Base(r.Path)) },
	}
	for _, key := range levels {
		var matches []IndexRecord
		for _, record := range records {
			if key(record) == reference {
				matches = append(matches, record)
			}
		}
		switch len(matches) {
		case 0:
			continue
		case 1:
			return readDocument(filepath.Join(root, matches[0].Path))
		default:
			paths := make([]string, len(matches))
			for i, match := range matches {
				paths[i] = filepath.Join(root, match.Path)
			}
			return frontmatter.Document{}, &AmbiguousReferenceError{Reference: reference, Matches: paths}
		}
	}
	return frontmatter.Document{}, &UnknownReferenceError{Reference: reference}
}

// readDocument reads and parses one document through the shared reader.
func readDocument(path string) (frontmatter.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return frontmatter.Document{}, fmt.Errorf("read %s: %w", path, err)
	}
	return frontmatter.Parse(path, data)
}

// stem is a file name without its extension.
func stem(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}
