// Package frontmatter reads and writes the frontmatter block of a vault
// document: tolerant on the way in, always parseable YAML on the way out.
package frontmatter

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnterminated reports a document whose frontmatter opens with a fence and
// never closes it.
var ErrUnterminated = errors.New("unterminated frontmatter")

const (
	openFence  = "---\n"
	closeFence = "\n---\n"
)

// bom is the UTF-8 byte-order mark, which some editors put in front of a file.
const bom = "\ufeff"

// splitFrontmatter separates a document's frontmatter block from its body.
//
// The bytes are normalised first (see normalise), so a document saved with CRLF
// line endings or with a byte-order mark reads the same as a clean one.
// hasFrontmatter is false when the document does not open with a fence, in which
// case the whole input is the body. An unterminated block is an error naming the
// document.
func splitFrontmatter(name string, data []byte) (raw string, hasFrontmatter bool, body string, err error) {
	text := normalise(string(data))
	if !strings.HasPrefix(text, openFence) {
		return "", false, text, nil
	}
	// The closing fence is searched from one byte before the opening fence ends,
	// so a closing fence on the very next line is an empty block rather than an
	// unterminated one.
	from := len(openFence) - 1
	at := strings.Index(text[from:], closeFence)
	if at == -1 {
		return "", false, "", fmt.Errorf("%w: %s", ErrUnterminated, name)
	}
	end := from + at
	if end < len(openFence) {
		return "", true, text[end+len(closeFence):], nil
	}
	return text[len(openFence):end], true, text[end+len(closeFence):], nil
}

// normalise prepares raw document bytes for parsing: a leading byte-order mark is
// dropped, and CRLF and lone CR become LF. The reference implementation seeds a
// BOM'd document to "no frontmatter" and therefore loses all of its metadata.
func normalise(text string) string {
	text = strings.TrimPrefix(text, bom)
	if !strings.Contains(text, "\r") {
		return text
	}
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}
