package vault

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/petsagouris/specshelf/internal/frontmatter"
)

// Filters select documents for the collection commands. An empty field, and the
// kind "all", matches everything; the fields are ANDed.
type Filters struct {
	Kind         string
	Repo         string
	Tag          string
	CodexSession string
	Status       string
}

// matches reports whether a record passes every set filter.
func (f Filters) matches(r IndexRecord) bool {
	if f.Kind != "" && f.Kind != "all" && r.Kind != f.Kind {
		return false
	}
	if f.Repo != "" && !contains(r.Repos, f.Repo) {
		return false
	}
	if f.Tag != "" && !contains(r.Tags, f.Tag) {
		return false
	}
	if f.CodexSession != "" && !contains(r.CodexSessions, f.CodexSession) {
		return false
	}
	if f.Status != "" && r.Status != f.Status {
		return false
	}
	return true
}

// Query returns the records matching f, ordered by updated descending and then
// id descending, with one warning per unreadable document. It is the store behind
// `list` and `prime`.
func Query(root string, f Filters) ([]IndexRecord, []Warning, error) {
	records, warnings, err := List(root)
	if err != nil {
		return nil, nil, err
	}
	filtered := records[:0]
	for _, record := range records {
		if f.matches(record) {
			filtered = append(filtered, record)
		}
	}
	sortRecords(filtered)
	return filtered, warnings, nil
}

// sortRecords orders records by updated descending, then id descending.
func sortRecords(records []IndexRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Updated != records[j].Updated {
			return records[i].Updated > records[j].Updated
		}
		return records[i].ID > records[j].ID
	})
}

// Match is one search result: the document's identity, its vault relative path,
// the 1-based body line the query matched (0 when it matched the metadata), and
// the snippet to show.
type Match struct {
	ID      string
	Kind    string
	Path    string
	Line    int
	Snippet string
}

// Search scans the vault for documents whose metadata or body contains query,
// case-insensitively, narrowed by f and ordered like Query. A metadata match
// reports the title as its snippet with line 0; a body match reports the first
// matching line. The store is the only package that may read the body, which is
// why search lives here rather than in the command.
func Search(root, query string, f Filters) ([]Match, []Warning, error) {
	needle := strings.ToLower(query)
	type candidate struct {
		record IndexRecord
		body   string
	}
	var candidates []candidate
	var warnings []Warning
	err := walkDocuments(root, func(dir, name string) error {
		rel := dir + "/" + name
		data, err := os.ReadFile(filepath.Join(root, dir, name))
		if err != nil {
			warnings = append(warnings, warn(rel, err))
			return nil
		}
		doc, err := frontmatter.Parse(rel, data)
		if err != nil {
			warnings = append(warnings, warn(rel, err))
			return nil
		}
		record := newRecord(doc, rel)
		if f.matches(record) {
			candidates = append(candidates, candidate{record: record, body: doc.Body})
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i].record, candidates[j].record
		if a.Updated != b.Updated {
			return a.Updated > b.Updated
		}
		return a.ID > b.ID
	})

	var matches []Match
	for _, c := range candidates {
		record := c.record
		if strings.Contains(metadataText(record), needle) {
			matches = append(matches, Match{ID: record.ID, Kind: record.Kind, Path: record.Path, Snippet: record.Title})
			continue
		}
		for i, line := range splitLines(c.body) {
			if strings.Contains(strings.ToLower(line), needle) {
				matches = append(matches, Match{ID: record.ID, Kind: record.Kind, Path: record.Path, Line: i + 1, Snippet: strings.TrimSpace(line)})
				break
			}
		}
	}
	return matches, warnings, nil
}

// metadataText is the lowercased JSON of a document's metadata, the haystack
// searched before the body. It is built in the index record's field order, so the
// text a match is found in is the text the record holds.
//
// shortcut: only the known fields are searched; an unknown key in a document's
// frontmatter is not, because the parsed map has lost the file's key order.
// Upgrade if a vault starts carrying searchable unknown keys.
func metadataText(record IndexRecord) string {
	var b []byte
	b = append(b, '{')
	scalars := []struct{ key, value string }{
		{"id", record.ID}, {"kind", record.Kind}, {"title", record.Title},
		{"status", record.Status}, {"created", record.Created}, {"updated", record.Updated},
	}
	for i, s := range scalars {
		if i > 0 {
			b = append(b, ',', ' ')
		}
		b = appendJSONString(b, s.key)
		b = append(b, ':', ' ')
		b = appendJSONString(b, s.value)
	}
	lists := []struct {
		key   string
		items []string
	}{
		{"repos", record.Repos}, {"tags", record.Tags}, {"codex_sessions", record.CodexSessions},
	}
	for _, l := range lists {
		b = append(b, ',', ' ')
		b = appendJSONString(b, l.key)
		b = append(b, ':', ' ')
		b = appendJSONList(b, l.items)
	}
	b = append(b, '}')
	return strings.ToLower(string(b))
}

// splitLines splits a body into lines the same way summary breaks them, so a
// reported line number counts the lines a reader sees, and the empty tail after a
// trailing break is dropped.
func splitLines(body string) []string {
	var lines []string
	start := 0
	for i, r := range body {
		if !isLineBreak(r) {
			continue
		}
		lines = append(lines, body[start:i])
		start = i + len(string(r))
	}
	if start < len(body) {
		lines = append(lines, body[start:])
	}
	return lines
}

// contains reports whether items holds value.
func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
