package vault

import (
	"fmt"
	"path/filepath"

	"github.com/petsagouris/specshelf/internal/frontmatter"
)

// KindMismatch reports a document whose declared kind disagrees with the
// directory it lives in. The reader's Kind() substitutes the directory when the
// kind field is absent, so it cannot show the disagreement; the declaration is
// read from the document's own frontmatter instead.
type KindMismatch struct {
	Path      string // vault relative slash path
	Declared  string // the kind the document states
	Directory string // the directory the document lives in
}

// Text is the report line for the mismatch, naming the path, the declared kind
// and the directory. The store never prints it.
func (m KindMismatch) Text() string {
	return fmt.Sprintf("%s: kind %q does not match the %s directory", m.Path, m.Declared, m.Directory)
}

// Validation is a vault's report: the documents that could not be read, each
// with its reason, and the documents whose declared kind disagrees with their
// directory.
type Validation struct {
	Unreadable []Warning
	Mismatches []KindMismatch
}

// Problems is the number of findings. `doctor` exits non-zero when it is at
// least one, and exits zero for a healthy vault.
func (v Validation) Problems() int { return len(v.Unreadable) + len(v.Mismatches) }

// Validate walks the vault the way List does and reports what is wrong with it:
// each document that could not be read, with the reason, and each document whose
// declared kind disagrees with its directory. A directory that cannot be listed
// at all is the error; every other finding is returned in the report. A healthy
// vault reports nothing. It prints nothing and knows nothing about exit codes.
func Validate(root string) (Validation, error) {
	records, warnings, err := List(root)
	if err != nil {
		return Validation{}, err
	}
	v := Validation{Unreadable: warnings}
	for _, record := range records {
		// IndexRecord carries the classified kind, not the declared one, so the
		// document is read again through the shared reader to get its own words.
		doc, err := readDocument(filepath.Join(root, record.Path))
		if err != nil {
			v.Unreadable = append(v.Unreadable, warn(record.Path, err))
			continue
		}
		declared, stated := declaredKind(doc)
		if !stated {
			continue
		}
		directory := filepath.Dir(record.Path)
		if declared != impliedKind(directory) {
			v.Mismatches = append(v.Mismatches, KindMismatch{
				Path:      record.Path,
				Declared:  declared,
				Directory: directory,
			})
		}
	}
	return v, nil
}

// declaredKind returns the kind the document's frontmatter states, and whether
// it states one. An absent or empty field is not a declaration: the reader's
// Kind() classifies such a document by its directory, so there is nothing to
// disagree with. Metadata is the parsed frontmatter the reader already exposes,
// so this reads the document's own value without a second parser.
func declaredKind(doc frontmatter.Document) (string, bool) {
	text, ok := doc.Metadata["kind"].(string)
	if !ok || text == "" {
		return "", false
	}
	return text, true
}

// impliedKind is the kind a document's directory declares: decisions holds
// decisions, everything else specs.
func impliedKind(directory string) string {
	if filepath.Base(directory) == "decisions" {
		return "decision"
	}
	return "spec"
}

// Statistics is a vault's document counts grouped by kind, status, repo and tag.
// A document with several repos or tags counts once in each, so a group's total
// can exceed Documents. A document with no status counts in the empty status
// group, so the status groups sum to Documents.
type Statistics struct {
	Documents int
	ByKind    map[string]int
	ByStatus  map[string]int
	ByRepo    map[string]int
	ByTag     map[string]int
}

// Stats walks the vault the way List does and returns its counts grouped by
// kind, status, repo and tag, together with the warnings for the documents it
// could not read — those documents contribute no counts. It prints nothing and
// knows nothing about exit codes.
func Stats(root string) (Statistics, []Warning, error) {
	records, warnings, err := List(root)
	if err != nil {
		return Statistics{}, nil, err
	}
	stats := Statistics{
		Documents: len(records),
		ByKind:    make(map[string]int),
		ByStatus:  make(map[string]int),
		ByRepo:    make(map[string]int),
		ByTag:     make(map[string]int),
	}
	for _, record := range records {
		stats.ByKind[record.Kind]++
		stats.ByStatus[record.Status]++
		for _, repo := range record.Repos {
			stats.ByRepo[repo]++
		}
		for _, tag := range record.Tags {
			stats.ByTag[tag]++
		}
	}
	return stats, warnings, nil
}
