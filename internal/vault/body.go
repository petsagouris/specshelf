package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/petsagouris/specshelf/internal/docs"
)

// normaliseTitle is the one definition of the title a document is written with:
// surrounding whitespace removed and inner runs of whitespace collapsed to a
// single space. The id and file name are derived from the title as given, not
// from this, so normalising cannot move a document.
func normaliseTitle(title string) string {
	return strings.Join(strings.Fields(title), " ")
}

// docTemplates maps a document kind to its template file name under
// templates/ and the built-in default used when that file is missing.
var docTemplates = map[string]struct {
	file    string
	builtin string
}{
	"spec":     {"spec.md", docs.SpecTemplate},
	"decision": {"decision.md", docs.DecisionTemplate},
}

// seedBody returns the body of a new document of kind: the vault's template for
// that kind with its {title} placeholder replaced by the normalised title. A
// missing template file is written back from the built-in default first, so an
// edited template survives and a deleted one is restored; a vault with no
// template directory is left for the caller's own initialisation to fix.
func seedBody(root, kind, title string) (string, error) {
	t, ok := docTemplates[kind]
	if !ok {
		return "", fmt.Errorf("no template for kind %q", kind)
	}
	path := filepath.Join(root, "templates", t.file)
	if err := writeIfMissing(path, t.builtin); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return strings.ReplaceAll(string(data), "{title}", normaliseTitle(title)), nil
}
