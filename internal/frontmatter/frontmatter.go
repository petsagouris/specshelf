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

// splitFrontmatter separates a document's frontmatter block from its body.
//
// Line endings are normalised to LF first, so a document saved with CRLF line
// endings — a Windows checkout running git with core.autocrlf, for instance —
// reads the same as one saved with LF. hasFrontmatter is false when the
// document does not open with a fence, in which case the whole input is the
// body. An unterminated block is an error naming the document.
func splitFrontmatter(name string, data []byte) (raw string, hasFrontmatter bool, body string, err error) {
	text := normaliseLineEndings(string(data))
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

// normaliseLineEndings converts CRLF and lone CR to LF, leaving text with no
// carriage return untouched.
func normaliseLineEndings(text string) string {
	if !strings.Contains(text, "\r") {
		return text
	}
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}
