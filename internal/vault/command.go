package vault

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/petsagouris/specshelf/internal/frontmatter"
)

// Init creates the vault's directories, built-in README and templates under
// root, each only when it is missing, so a user edit survives. It never writes
// index.jsonl. This is the store half of `specshelf init`; the command prints
// the path.
func Init(root string) error { return initVault(root) }

// NewDocument is the input to Create: everything a new document carries that a
// command can set. Kind is "spec" or "decision".
type NewDocument struct {
	Kind     string
	Title    string
	Repos    []string
	Tags     []string
	Sessions []string
	Status   string
}

// Create composes a document of in.Kind from the vault's template and writes it
// exclusively at the first name whose id and file name are both free, then
// refreshes the index. It returns the path it created. A document already at a
// candidate name is never overwritten, so two concurrent creations of one title
// both succeed under distinct names, each with its own -N suffixed id.
func Create(root string, in NewDocument) (string, error) {
	if err := initVault(root); err != nil {
		return "", err
	}
	dirName, prefix := "specs", "spec"
	if in.Kind == "decision" {
		dirName, prefix = "decisions", "decision"
	}
	date := localDate(time.Now())
	slug := slugify(in.Title)
	body, err := seedBody(root, in.Kind, in.Title)
	if err != nil {
		return "", err
	}
	records, _, err := List(root)
	if err != nil {
		return "", err
	}
	taken := make(map[string]bool, len(records))
	for _, record := range records {
		taken[record.ID] = true
	}

	dir := filepath.Join(root, dirName)
	for n := 1; ; n++ {
		suffix := ""
		if n > 1 {
			suffix = "-" + strconv.Itoa(n)
		}
		id := documentID(prefix, date, slug+suffix)
		if taken[id] {
			continue
		}
		path := documentPath(dir, date, slug+suffix)
		doc := newDocument(id, in, body, date, path)
		if err := writeExclusive(path, frontmatter.Marshal(doc)); err != nil {
			// A name taken between the check and the create costs the next one.
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return "", err
		}
		if err := WriteIndex(root); err != nil {
			return "", err
		}
		return path, nil
	}
}

// newDocument is the metadata a freshly created document carries, in the shape
// the typed view reads: scalars as strings, list fields as []string.
func newDocument(id string, in NewDocument, body, date, name string) frontmatter.Document {
	return frontmatter.Document{
		Name: name,
		Metadata: frontmatter.Metadata{
			"id":             id,
			"kind":           in.Kind,
			"title":          normaliseTitle(in.Title),
			"status":         in.Status,
			"created":        date,
			"updated":        date,
			"repos":          in.Repos,
			"tags":           in.Tags,
			"codex_sessions": in.Sessions,
		},
		Body: body,
	}
}

// LinkSessions resolves ref, appends sessions to its codex_sessions, stamps
// updated with today's date, rewrites the document atomically and refreshes the
// index. It returns the document's path. Unknown keys and the body are kept, so
// linking never discards what it did not set. An empty sessions list is an error,
// as in the reference tool.
func LinkSessions(root, ref string, sessions []string) (string, error) {
	merged := mergeUnique(nil, sessions)
	if len(merged) == 0 {
		return "", errors.New("at least one Codex session id is required")
	}
	doc, err := ResolveReference(root, ref)
	if err != nil {
		return "", err
	}
	metadata := copyMetadata(doc.Metadata)
	metadata["codex_sessions"] = mergeUnique(doc.CodexSessions(), sessions)
	metadata["updated"] = localDate(time.Now())
	out := frontmatter.Document{Name: doc.Name, Metadata: metadata, Body: doc.Body}
	if err := WriteDocument(doc.Name, frontmatter.Marshal(out)); err != nil {
		return "", err
	}
	if err := WriteIndex(root); err != nil {
		return "", err
	}
	return doc.Name, nil
}

// copyMetadata returns a shallow copy of a document's metadata, so a rewrite can
// set fields without mutating the document it read. Unknown keys ride along and
// keep their document's shape.
func copyMetadata(in frontmatter.Metadata) frontmatter.Metadata {
	out := make(frontmatter.Metadata, len(in)+1)
	for key, value := range in {
		out[key] = value
	}
	return out
}

// mergeUnique concatenates two lists, trims each item, drops empty ones and
// de-duplicates while preserving first-seen order — the reference tool's
// merge_unique.
func mergeUnique(existing, additions []string) []string {
	var out []string
	seen := make(map[string]bool, len(existing)+len(additions))
	for _, list := range [2][]string{existing, additions} {
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item == "" || seen[item] {
				continue
			}
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

// Found is a document resolved by reference: its path and its raw bytes.
type Found struct {
	Path string
	Data []byte
}

// Find resolves a reference and reads the document's raw bytes, which is what
// `show` prints and what `edit` and `rm` will need. The reader is the same
// reference resolver every read command uses.
func Find(root, ref string) (Found, error) {
	doc, err := ResolveReference(root, ref)
	if err != nil {
		return Found{}, err
	}
	data, err := os.ReadFile(doc.Name)
	if err != nil {
		return Found{}, fmt.Errorf("read %s: %w", doc.Name, err)
	}
	return Found{Path: doc.Name, Data: data}, nil
}

// Editor opens one document for interactive editing and returns when the editor
// has exited. A command supplies the implementation, so tests can inject a fake
// and the session reuses the same seam (D11).
type Editor interface {
	Edit(path string) error
}

// Edit resolves ref, hands the document to editor, and — only when the editor
// left different bytes behind — stamps updated with today's date, rewrites the
// document atomically and refreshes the index. It returns the document's path.
// An editor that changed nothing leaves the file and the index exactly as they
// were, because nothing on disk changed. Unknown keys, the body, id, kind and
// created are the editor's to keep: the rewrite re-reads the file and re-emits
// it through the shared writer.
func Edit(root, ref string, editor Editor) (string, error) {
	found, err := Find(root, ref)
	if err != nil {
		return "", err
	}
	if err := editor.Edit(found.Path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(found.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", found.Path, err)
	}
	if bytes.Equal(data, found.Data) {
		return found.Path, nil
	}
	doc, err := frontmatter.Parse(found.Path, data)
	if err != nil {
		return "", err
	}
	metadata := copyMetadata(doc.Metadata)
	metadata["updated"] = localDate(time.Now())
	out := frontmatter.Document{Name: doc.Name, Metadata: metadata, Body: doc.Body}
	if err := WriteDocument(found.Path, frontmatter.Marshal(out)); err != nil {
		return "", err
	}
	if err := WriteIndex(root); err != nil {
		return "", err
	}
	return found.Path, nil
}

// SetStatus sets the document's frontmatter status, stamps updated with today's
// date, rewrites it atomically and refreshes the index. It returns the path.
func SetStatus(root, ref, status string) (string, error) {
	doc, err := ResolveReference(root, ref)
	if err != nil {
		return "", err
	}
	metadata := copyMetadata(doc.Metadata)
	metadata["status"] = status
	metadata["updated"] = localDate(time.Now())
	out := frontmatter.Document{Name: doc.Name, Metadata: metadata, Body: doc.Body}
	if err := WriteDocument(doc.Name, frontmatter.Marshal(out)); err != nil {
		return "", err
	}
	if err := WriteIndex(root); err != nil {
		return "", err
	}
	return doc.Name, nil
}

// Remove deletes the document ref names and refreshes the index, which is
// written atomically like every other refresh. It returns the path it removed.
func Remove(root, ref string) (string, error) {
	doc, err := ResolveReference(root, ref)
	if err != nil {
		return "", err
	}
	if err := os.Remove(doc.Name); err != nil {
		return "", fmt.Errorf("remove %s: %w", doc.Name, err)
	}
	if err := WriteIndex(root); err != nil {
		return "", err
	}
	return doc.Name, nil
}
