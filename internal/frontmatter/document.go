package frontmatter

import (
	"path/filepath"
	"strings"
)

// Document is one vault document: the metadata parsed out of its frontmatter, its
// body, and the name it was read from — which the accessors fall back to when a
// field is absent, the way the reference implementation does.
type Document struct {
	Name     string
	Metadata Metadata
	Body     string
}

// Parse reads a document from its bytes. name is the document's path or file
// name; it is used in error messages and as the fallback for absent fields, and
// it does not have to exist on disk. The returned Document always has a non-nil
// Metadata, so callers can read it without a nil check.
func Parse(name string, data []byte) (Document, error) {
	raw, hasFrontmatter, body, err := splitFrontmatter(name, data)
	if err != nil {
		return Document{}, err
	}
	doc := Document{Name: name, Metadata: Metadata{}, Body: body}
	if hasFrontmatter {
		doc.Metadata = parseBlock(raw)
	}
	return doc, nil
}

// ID returns the document's id, falling back to the file stem for a document
// with no frontmatter.
func (d Document) ID() string { return d.scalar("id", d.stem()) }

// Kind returns "spec" or "decision". A document that does not say is classified
// by the directory it sits in, as the reference implementation does.
func (d Document) Kind() string {
	if kind := d.scalar("kind", ""); kind != "" {
		return kind
	}
	if filepath.Base(filepath.Dir(d.Name)) == "decisions" {
		return "decision"
	}
	return "spec"
}

// Title returns the document's title, falling back to the file stem when the
// field is absent. A title that is present but empty is empty — never a stem,
// and never the text "[]" the reference implementation produces.
func (d Document) Title() string { return d.scalar("title", d.stem()) }

// Status returns the frontmatter status, empty when absent.
func (d Document) Status() string { return d.scalar("status", "") }

// Created returns the creation date as written, empty when absent.
func (d Document) Created() string { return d.scalar("created", "") }

// Updated returns the last update date as written, empty when absent.
func (d Document) Updated() string { return d.scalar("updated", "") }

// Repos returns the repositories the document concerns.
func (d Document) Repos() []string { return d.list("repos") }

// Tags returns the document's tags.
func (d Document) Tags() []string { return d.list("tags") }

// CodexSessions returns the session ids linked to the document.
func (d Document) CodexSessions() []string { return d.list("codex_sessions") }

// scalar returns a field's string value, the fallback when the field is absent,
// and the empty string when the field holds a list: a field whose shape does not
// match its meaning is a document problem for `doctor` to report, not something
// to guess at here.
func (d Document) scalar(key, fallback string) string {
	value, ok := d.Metadata[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

// list returns a list field's items, or nil when the field is absent or holds a
// scalar. Mirroring scalar, a shape mismatch yields nothing rather than a guess.
func (d Document) list(key string) []string {
	items, ok := d.Metadata[key].([]string)
	if !ok {
		return nil
	}
	return items
}

// stem is the file name without its extension.
func (d Document) stem() string {
	base := filepath.Base(d.Name)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
