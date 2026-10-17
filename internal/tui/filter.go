package tui

import (
	"strings"

	"github.com/petsagouris/specshelf/internal/vault"
)

// Filter narrows the loaded documents in memory. The zero value matches every
// document. Query is a case-insensitive substring matched against a document's
// metadata text; the named fields must match exactly and are ANDed.
type Filter struct {
	Query  string
	Kind   string
	Status string
	Repo   string
	Tag    string
}

// matches reports whether record passes every set part of the filter.
func (f Filter) matches(record vault.IndexRecord) bool {
	if f.Kind != "" && record.Kind != f.Kind {
		return false
	}
	if f.Status != "" && record.Status != f.Status {
		return false
	}
	if f.Repo != "" && !contains(record.Repos, f.Repo) {
		return false
	}
	if f.Tag != "" && !contains(record.Tags, f.Tag) {
		return false
	}
	if query := strings.ToLower(strings.TrimSpace(f.Query)); query != "" {
		if !strings.Contains(f.haystack(record), query) {
			return false
		}
	}
	return true
}

// haystack is the lowercased text a query is matched against: the document's
// metadata values, one per line.
func (f Filter) haystack(record vault.IndexRecord) string {
	parts := []string{
		record.ID, record.Kind, record.Title, record.Status,
		record.Created, record.Updated, record.Path, record.Summary,
	}
	parts = append(parts, record.Repos...)
	parts = append(parts, record.Tags...)
	return strings.ToLower(strings.Join(parts, "\n"))
}

// summary is the one-line description of the active filter, blank when the
// filter keeps every document.
func (f Filter) summary() string {
	var parts []string
	for _, part := range []struct{ name, value string }{
		{"", f.Query}, {"kind", f.Kind}, {"status", f.Status},
		{"repo", f.Repo}, {"tag", f.Tag},
	} {
		if part.value == "" {
			continue
		}
		if part.name == "" {
			parts = append(parts, `"`+part.value+`"`)
		} else {
			parts = append(parts, part.name+"="+part.value)
		}
	}
	return strings.Join(parts, " ")
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

// options returns the distinct non-empty values of a field across records, in
// first-seen order, with the empty "all" option first.
func options(records []vault.IndexRecord, pick func(vault.IndexRecord) []string) []string {
	out := []string{""}
	seen := map[string]bool{}
	for _, record := range records {
		for _, value := range pick(record) {
			if value != "" && !seen[value] {
				seen[value] = true
				out = append(out, value)
			}
		}
	}
	return out
}

// next returns the value after current in opts, wrapping around. A current value
// that is no longer present starts again at the first option.
func next(opts []string, current string) string {
	for i, option := range opts {
		if option == current {
			return opts[(i+1)%len(opts)]
		}
	}
	return opts[0]
}

// kindOptions and friends are the values the filter keys cycle through.
func kindOptions(records []vault.IndexRecord) []string {
	return options(records, func(r vault.IndexRecord) []string { return []string{r.Kind} })
}

func statusOptions(records []vault.IndexRecord) []string {
	return options(records, func(r vault.IndexRecord) []string { return []string{r.Status} })
}

func repoOptions(records []vault.IndexRecord) []string {
	return options(records, func(r vault.IndexRecord) []string { return r.Repos })
}

func tagOptions(records []vault.IndexRecord) []string {
	return options(records, func(r vault.IndexRecord) []string { return r.Tags })
}
