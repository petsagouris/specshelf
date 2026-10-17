package vault

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/petsagouris/specshelf/internal/frontmatter"
)

// IndexRecord is one line of index.jsonl: a document's metadata plus its vault
// relative path and summary, in the field order the reference implementation
// writes.
type IndexRecord struct {
	ID            string
	Kind          string
	Title         string
	Status        string
	Created       string
	Updated       string
	Repos         []string
	Tags          []string
	CodexSessions []string
	Path          string
	Summary       string
}

// MarshalJSON renders the record as the reference tool's
// json.dumps(record, ensure_ascii=False) does: a fixed key order, ": " after a
// key and ", " between list items, and every character the reference leaves
// alone written as itself. encoding/json cannot produce these bytes — it escapes
// `<`, `>`, `&`, U+2028 and U+2029 and puts no space after a colon.
func (r IndexRecord) MarshalJSON() ([]byte, error) { return r.appendJSON(nil), nil }

// EncodeIndex renders records as the whole content of index.jsonl: one JSON
// object per line with a newline after every line, including the last.
func EncodeIndex(records []IndexRecord) []byte {
	var out []byte
	for _, record := range records {
		out = record.appendJSON(out)
		out = append(out, '\n')
	}
	return out
}

// Index reads every document in the vault and returns its records in index
// order: the specs directory before decisions, and file name within a
// directory. A missing directory is skipped, so a vault without one is not an
// error; a document that cannot be read or parsed is, naming the file. Index
// only reads — writing the file atomically belongs to the caller.
func Index(root string) ([]IndexRecord, error) {
	var records []IndexRecord
	for _, dir := range []string{"specs", "decisions"} {
		path := root + "/" + dir
		entries, err := os.ReadDir(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		// os.ReadDir already returns entries sorted by file name.
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".md") {
				continue
			}
			rec, err := readIndexRecord(path, dir, name)
			if err != nil {
				return nil, err
			}
			records = append(records, rec)
		}
	}
	return records, nil
}

// WriteIndex regenerates index.jsonl from the vault's documents and writes it
// atomically, so a reader never sees a half-written index. A document it cannot
// read fails the whole refresh; List is the tolerant alternative. Every
// mutating command calls this before it exits.
func WriteIndex(root string) error {
	records, err := Index(root)
	if err != nil {
		return err
	}
	return WriteDocument(filepath.Join(root, "index.jsonl"), EncodeIndex(records))
}

// readIndexRecord reads one document and builds its record. A file that cannot
// be read or parsed is an error naming it; deciding to skip and warn is the
// caller's job.
func readIndexRecord(path, dir, name string) (IndexRecord, error) {
	full := path + "/" + name
	data, err := os.ReadFile(full)
	if err != nil {
		return IndexRecord{}, fmt.Errorf("read %s: %w", full, err)
	}
	doc, err := frontmatter.Parse(full, data)
	if err != nil {
		return IndexRecord{}, err
	}
	return IndexRecord{
		ID:            doc.ID(),
		Kind:          doc.Kind(),
		Title:         doc.Title(),
		Status:        doc.Status(),
		Created:       doc.Created(),
		Updated:       doc.Updated(),
		Repos:         doc.Repos(),
		Tags:          doc.Tags(),
		CodexSessions: doc.CodexSessions(),
		Path:          dir + "/" + name,
		Summary:       summary(doc.Body),
	}, nil
}

// summary is the first body line that is neither blank nor a heading, with
// surrounding whitespace removed — the reference implementation's
// first_content_line, which strips a line before deciding whether it is a
// heading, so an indented "#" line is a heading too.
func summary(body string) string {
	for _, line := range strings.Split(body, "\n") {
		text := strings.TrimSpace(line)
		if text != "" && !strings.HasPrefix(text, "#") {
			return text
		}
	}
	return ""
}

// appendJSON writes the record's line, without its newline, onto dst.
func (r IndexRecord) appendJSON(dst []byte) []byte {
	dst = append(dst, `{"id": `...)
	dst = appendJSONString(dst, r.ID)
	dst = append(dst, `, "kind": `...)
	dst = appendJSONString(dst, r.Kind)
	dst = append(dst, `, "title": `...)
	dst = appendJSONString(dst, r.Title)
	dst = append(dst, `, "status": `...)
	dst = appendJSONString(dst, r.Status)
	dst = append(dst, `, "created": `...)
	dst = appendJSONString(dst, r.Created)
	dst = append(dst, `, "updated": `...)
	dst = appendJSONString(dst, r.Updated)
	dst = append(dst, `, "repos": `...)
	dst = appendJSONList(dst, r.Repos)
	dst = append(dst, `, "tags": `...)
	dst = appendJSONList(dst, r.Tags)
	dst = append(dst, `, "codex_sessions": `...)
	dst = appendJSONList(dst, r.CodexSessions)
	dst = append(dst, `, "path": `...)
	dst = appendJSONString(dst, r.Path)
	dst = append(dst, `, "summary": `...)
	dst = appendJSONString(dst, r.Summary)
	return append(dst, '}')
}

// appendJSONString writes s as a JSON string the way CPython's
// json.dumps(..., ensure_ascii=False) does: escape the two mandatory
// characters, the five short control escapes, and any other control character
// below 0x20 as \u00xx, leaving everything else — including 0x7F and all
// non-ASCII — as its own bytes.
func appendJSONString(dst []byte, s string) []byte {
	const hex = "0123456789abcdef"
	dst = append(dst, '"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\\':
			dst = append(dst, '\\', c)
		case '\b':
			dst = append(dst, '\\', 'b')
		case '\f':
			dst = append(dst, '\\', 'f')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			if c < 0x20 {
				dst = append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xf])
				continue
			}
			dst = append(dst, c)
		}
	}
	return append(dst, '"')
}

// appendJSONList writes items as a JSON array with ", " between them. A nil
// slice is the empty array, matching the reference's empty list.
func appendJSONList(dst []byte, items []string) []byte {
	dst = append(dst, '[')
	for i, item := range items {
		if i > 0 {
			dst = append(dst, ',', ' ')
		}
		dst = appendJSONString(dst, item)
	}
	return append(dst, ']')
}
