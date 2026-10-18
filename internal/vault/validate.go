package vault

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

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

// MissingLayout reports a directory or file that initialising a vault creates
// and the vault root does not hold. It is the shape a missing `specs/` or a
// deleted `README.md` takes. An extra directory or file the initialiser does not
// create is none of the validator's business and is not reported.
type MissingLayout struct {
	Path string // vault relative slash path
}

// Text is the report line for the missing item. The store never prints it.
func (m MissingLayout) Text() string {
	return fmt.Sprintf("%s: missing from the vault layout", m.Path)
}

// IdentityMismatch reports a document whose declared id or whose file name does
// not match its title.
type IdentityMismatch struct {
	Path  string // vault relative slash path
	Field string // "id" or "file name"
	Got   string // the value the document carries
	Want  string // the value the naming functions derive from the title
}

// Text is the report line for the mismatch. The store never prints it.
func (m IdentityMismatch) Text() string {
	return fmt.Sprintf("%s: %s %q does not match the title (want %q)", m.Path, m.Field, m.Got, m.Want)
}

// Validation is a vault's report: the layout items that are missing, the
// documents that could not be read, each with its reason, the documents whose
// declared kind disagrees with their directory, and the documents whose declared
// id or file name disagrees with their title.
type Validation struct {
	Missing    []MissingLayout
	Unreadable []Warning
	Mismatches []KindMismatch
	Identities []IdentityMismatch
}

// Problems is the number of findings. `doctor` exits non-zero when it is at
// least one, and exits zero for a healthy vault.
func (v Validation) Problems() int {
	return len(v.Missing) + len(v.Unreadable) + len(v.Mismatches) + len(v.Identities)
}

// Validate walks the vault the way List does and reports what is wrong with it:
// the layout items initialising a vault creates that are missing, each document
// that could not be read, with the reason, each document whose declared kind
// disagrees with its directory, and each document whose declared id or file name
// does not match its title. A directory that cannot be listed at all is the
// error; every other finding is returned in the report. A healthy vault reports
// nothing. It prints nothing and knows nothing about exit codes.
func Validate(root string) (Validation, error) {
	records, warnings, err := List(root)
	if err != nil {
		return Validation{}, err
	}
	v := Validation{Missing: missingLayout(root), Unreadable: warnings}
	for _, record := range records {
		// IndexRecord carries the classified kind, not the declared one, so the
		// document is read again through the shared reader to get its own words.
		doc, err := readDocument(filepath.Join(root, record.Path))
		if err != nil {
			v.Unreadable = append(v.Unreadable, warn(record.Path, err))
			continue
		}
		directory := filepath.Dir(record.Path)
		if declared, stated := declaredKind(doc); stated && declared != impliedKind(directory) {
			v.Mismatches = append(v.Mismatches, KindMismatch{
				Path:      record.Path,
				Declared:  declared,
				Directory: directory,
			})
		}
		v.Identities = append(v.Identities, identityMismatches(record.Path, doc)...)
	}
	return v, nil
}

// missingLayout returns a finding for every directory and file the vault layout
// requires that root does not hold. Initialising the vault is what creates them,
// so a vault this tool set up has none; an extra item is not a finding.
func missingLayout(root string) []MissingLayout {
	var missing []MissingLayout
	for _, dir := range vaultDirs {
		if info, err := os.Stat(filepath.Join(root, dir)); err != nil || !info.IsDir() {
			missing = append(missing, MissingLayout{Path: dir})
		}
	}
	for _, f := range vaultFiles {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(f.rel))); err != nil {
			missing = append(missing, MissingLayout{Path: f.rel})
		}
	}
	return missing
}

// identityMismatches returns a finding for a document whose declared id or whose
// file name does not match its title.
//
// Both are derived from one title slug and one creation date: the file name is
// documentPath's `<YYYY-MM-DD>-<slug>.md` and the id is documentID's
// `<prefix>-<YYYYMMDD>-<slug>`. The date is taken from the file name, which is
// what documentPath wrote and is not user-editable; `created` is not consulted,
// so a corrected `created` does not read as an identity problem. A file name with
// no date prefix cannot be what documentPath produces, so it is reported on its
// own, with the date left as the placeholder YYYY-MM-DD. A `-N` suffix is the
// exclusive-create collision marker (D6) and is legitimate on either the id or
// the file name.
func identityMismatches(rel string, doc frontmatter.Document) []IdentityMismatch {
	dir := path.Dir(rel)
	name := path.Base(rel)
	slug := slugify(doc.Title())
	date := nameDate(name)

	if date == "" {
		return []IdentityMismatch{{
			Path:  rel,
			Field: "file name",
			Got:   name,
			Want:  filepath.Base(documentPath(dir, datePlaceholder, slug)),
		}}
	}

	var out []IdentityMismatch
	if want := filepath.Base(documentPath(dir, date, slug)); name != want && !collidesWith(name, want) {
		out = append(out, IdentityMismatch{Path: rel, Field: "file name", Got: name, Want: want})
	}
	wantID := documentID(impliedKind(dir), date, slug)
	if id, stated := declaredID(doc); stated && id != wantID && !collidesWith(id, wantID) {
		out = append(out, IdentityMismatch{Path: rel, Field: "id", Got: id, Want: wantID})
	}
	return out
}

// datePlaceholder stands in for the creation date of a file name that carries
// none, so the report can still show the shape documentPath produces.
const datePlaceholder = "YYYY-MM-DD"

// nameDate returns the YYYY-MM-DD date a document file name begins with, or ""
// when it begins with anything else. The dated form is the one documentPath
// writes.
func nameDate(name string) string {
	stem := strings.TrimSuffix(name, ".md")
	if len(stem) >= 11 && isDate(stem[:10]) && stem[10] == '-' {
		return stem[:10]
	}
	return ""
}

// isDate reports whether s is exactly YYYY-MM-DD.
func isDate(s string) bool {
	return len(s) == 10 && isDigits(s[0:4]) && s[4] == '-' && isDigits(s[5:7]) && s[7] == '-' && isDigits(s[8:10])
}

// isDigits reports whether s is all ASCII digits.
func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// collidesWith reports whether name is want with a `-N` suffix, N >= 2, inserted
// before want's extension — the name exclusive creation retries with when want is
// already taken (D6). It is what makes a legitimate `-2` collision pass.
func collidesWith(name, want string) bool {
	ext := filepath.Ext(want)
	stem := strings.TrimSuffix(want, ext)
	if !strings.HasPrefix(name, stem+"-") || !strings.HasSuffix(name, ext) {
		return false
	}
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, stem+"-"), ext))
	return err == nil && n >= 2
}

// declaredID returns the id the document's frontmatter states, and whether it
// states one. Mirroring declaredKind, an absent or empty field is not a
// declaration: the reader's ID() falls back to the file stem, so there is no
// document id to disagree with the title.
func declaredID(doc frontmatter.Document) (string, bool) {
	text, ok := doc.Metadata["id"].(string)
	if !ok || text == "" {
		return "", false
	}
	return text, true
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
