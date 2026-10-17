package frontmatter

import (
	"strings"
	"testing"
)

// A document saved with CRLF line endings is rewritten with LF, because the
// reader normalises the line endings before the writer ever sees them, and the
// writer only ever emits \n.
func TestRewriteCRLFDocumentUsesLF(t *testing.T) {
	const crlf = "---\r\nid: spec-20261017-x\r\nkind: spec\r\ntitle: X\r\n---\r\n# X\r\n\r\nbody line one\r\nbody line two\r\n"
	const lf = "---\nid: spec-20261017-x\nkind: spec\ntitle: X\n---\n# X\n\nbody line one\nbody line two\n"

	crlfDoc, err := Parse("specs/2026-10-17-x.md", []byte(crlf))
	if err != nil {
		t.Fatalf("Parse CRLF: %v", err)
	}
	lfDoc, err := Parse("specs/2026-10-17-x.md", []byte(lf))
	if err != nil {
		t.Fatalf("Parse LF: %v", err)
	}

	got := Marshal(crlfDoc)
	if strings.ContainsRune(string(got), '\r') {
		t.Errorf("rewriting a CRLF document produced a carriage return:\n%q", got)
	}
	if want := Marshal(lfDoc); string(got) != string(want) {
		t.Errorf("rewriting a CRLF document =\n%q\nwant the LF form\n%q", got, want)
	}
}
