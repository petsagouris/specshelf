package frontmatter

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The known fields, in the order they are written. Every one of them is written
// even when it holds nothing, so a document's shape does not depend on which
// fields happen to be set.
var (
	scalarFields = []string{"id", "kind", "title", "status", "created", "updated"}
	listFields   = []string{"repos", "tags", "codex_sessions"}
)

// Marshal renders a document: its frontmatter block followed by its body.
//
// The six scalar fields come first, then the three list fields, then any other
// keys sorted by name, which is what keeps a rewrite byte-stable.
func Marshal(doc Document) []byte {
	var b strings.Builder
	b.WriteString("---\n")

	for _, key := range scalarFields {
		b.WriteString(renderScalar(key))
		b.WriteString(": ")
		b.WriteString(renderScalar(scalarValue(doc, key)))
		b.WriteByte('\n')
	}
	for _, key := range listFields {
		writeList(&b, key, listValue(doc.Metadata, key))
	}
	for _, key := range otherKeys(doc.Metadata) {
		if items, ok := doc.Metadata[key].([]string); ok {
			writeList(&b, key, items)
			continue
		}
		b.WriteString(renderScalar(key))
		b.WriteString(": ")
		b.WriteString(renderScalar(fmt.Sprint(doc.Metadata[key])))
		b.WriteByte('\n')
	}

	b.WriteString("---\n")
	b.WriteString(doc.Body)
	return []byte(b.String())
}

// writeList writes a block sequence, or an empty flow sequence when there is
// nothing in it. A bare "key:" would be read as null by a YAML parser, and "[]"
// is read back as an empty list. The key is quoted like a value, so a key
// beginning "- " is not read back as an item.
func writeList(b *strings.Builder, key string, items []string) {
	if len(items) == 0 {
		b.WriteString(renderScalar(key))
		b.WriteString(": []\n")
		return
	}
	b.WriteString(renderScalar(key))
	b.WriteString(":\n")
	for _, item := range items {
		b.WriteString("  - ")
		b.WriteString(renderScalar(item))
		b.WriteByte('\n')
	}
}

// renderScalar returns a value safe to put after "key: ".
func renderScalar(value string) string {
	if needsQuoting(value) {
		return quote(value)
	}
	return value
}

// needsQuoting reports whether a value would be read as something other than the
// string it holds, or would break the line it sits on. Dates are deliberately not
// treated as needing quotes: created and updated are written bare, and quoting
// every document's dates would trade a stable byte form for a type distinction
// nobody consumes.
func needsQuoting(value string) bool {
	switch {
	case value == "":
		return true
	case strings.TrimSpace(value) != value:
		return true // leading or trailing whitespace is not preserved by a plain scalar
	case hasUnprintable(value):
		return true // cannot appear in a single-line scalar, and would break the block
	case strings.Contains(value, ": "), strings.HasSuffix(value, ":"):
		return true
	case strings.Contains(value, " #"):
		return true
	case strings.IndexByte(yamlIndicators, value[0]) >= 0:
		return true
	}
	if _, reserved := reservedScalars[strings.ToLower(value)]; reserved {
		return true
	}
	return looksNumeric(value)
}

// yamlIndicators are the characters that cannot begin a plain scalar. Quoting
// more than necessary is harmless; quoting less changes meaning.
const yamlIndicators = "-?:,[]{}#&*!|>'\"%@`"

// reservedScalars are words a YAML reader gives a non-string value to.
var reservedScalars = map[string]struct{}{
	"null": {}, "~": {}, "true": {}, "false": {}, "yes": {}, "no": {},
	"on": {}, "off": {}, "y": {}, "n": {}, ".inf": {}, ".nan": {},
	"-.inf": {}, "+.inf": {}, "-.nan": {}, "+.nan": {},
}

// looksNumeric reports whether a YAML reader would read the value as a number.
func looksNumeric(value string) bool {
	if _, err := strconv.ParseInt(value, 0, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseUint(value, 0, 64); err == nil {
		return true
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

// quote renders a double-quoted scalar, escaping the backslash, the quote and the
// line breaks a scalar cannot carry. Everything else is kept as written, so a
// value holding an unusual character still round-trips through Parse.
func quote(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if escapeRune(r) {
				fmt.Fprintf(&b, `\u%04x`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// hasUnprintable reports whether a value holds a character that cannot sit in a
// plain single-line scalar: a break, a tab, or a character YAML forbids.
func hasUnprintable(value string) bool {
	for _, r := range value {
		if r == '\n' || r == '\r' || r == '\t' || escapeRune(r) {
			return true
		}
	}
	return false
}

// escapeRune reports whether a character has to be written as a \uXXXX escape.
// YAML forbids the control characters outright, and the parsers in the wild read
// U+0085, U+2028 and U+2029 as line breaks, so leaving one raw would end the line
// the value sits on and change what the block means.
func escapeRune(r rune) bool {
	switch {
	case r < 0x20: // the wordy escapes for \n, \r and \t are handled before this
		return true
	case r == 0x7f:
		return true
	case r >= 0x80 && r <= 0x9f:
		return true
	case r == 0x2028 || r == 0x2029:
		return true
	}
	return false
}

// scalarValue returns the scalar field as text, which is what the accessors
// read: for id, kind and title that is the accessor itself, so a rewrite cannot
// blank a document whose metadata lacks one — a document that had no id gains
// the name-derived id `doctor` reports, not one the codec invents. A title that
// is present but empty stays empty, because that is what Title() returns.
func scalarValue(doc Document, key string) string {
	switch key {
	case "id":
		return doc.ID()
	case "kind":
		return doc.Kind()
	case "title":
		return doc.Title()
	}
	text, _ := doc.Metadata[key].(string)
	return text
}

// listValue returns a list field's items, or nothing when the field is absent or
// holds a scalar.
func listValue(metadata Metadata, key string) []string {
	items, _ := metadata[key].([]string)
	return items
}

// otherKeys returns the keys Marsal does not know, in name order.
func otherKeys(metadata Metadata) []string {
	known := make(map[string]struct{}, len(scalarFields)+len(listFields))
	for _, key := range scalarFields {
		known[key] = struct{}{}
	}
	for _, key := range listFields {
		known[key] = struct{}{}
	}
	var keys []string
	for key := range metadata {
		if _, isKnown := known[key]; !isKnown {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
