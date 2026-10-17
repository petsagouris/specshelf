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
