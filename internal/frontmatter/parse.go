package frontmatter

import (
	"strconv"
	"strings"
)

// Metadata holds parsed frontmatter fields. A field is a string when it carried
// a value on its own line, and []string when it carried list items. Which shape
// a known field must take is decided by the typed view, not here.
type Metadata map[string]any

// parseBlock parses the frontmatter dialect of the reference implementation and
// is deliberately more tolerant, so that a document it mangles still reads:
//
//   - blank lines are ignored and do not close an open list
//   - a list item is a line whose first non-space, non-tab characters are "- ",
//     at any indentation; an item with no value is ignored rather than stored as
//     an empty string
//   - a line without a separator does not close an open list, while a line with
//     a separator and a value does
//   - an empty value opens a list ("key:" and "key: []" alike), and a
//     well-formed quoted value or key is unquoted
//
// Line endings must already be normalised, which is what splitFrontmatter does.
// A repeated key keeps its last value, and an item with no open key is ignored.
func parseBlock(raw string) Metadata {
	metadata := Metadata{}
	var listKey string

	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if item, ok := listItem(line); ok && listKey != "" {
			// An item with no text at all is ignored, but an explicitly empty
			// quoted item is a value the writer emitted and has to survive.
			if item != "" {
				items, _ := metadata[listKey].([]string)
				metadata[listKey] = append(items, unquote(item))
			}
			continue
		}
		key, value, ok := splitKeyValue(line)
		if !ok {
			continue
		}
		key = unquote(key)
		switch value {
		case "", "[]":
			metadata[key] = []string{}
			listKey = key
		default:
			metadata[key] = unquote(value)
			listKey = ""
		}
	}
	return metadata
}

// splitKeyValue splits a line into its key and value at the colon that separates
// them, trimming both. A key the writer emitted quoted ends at its closing quote,
// so a quoted key holding ": " splits where it was written, not inside its quotes.
func splitKeyValue(line string) (key, value string, ok bool) {
	if strings.HasPrefix(line, `"`) {
		if end := closingQuote(line); end > 0 && end+1 < len(line) && line[end+1] == ':' {
			return strings.TrimSpace(line[:end+1]), strings.TrimSpace(line[end+2:]), true
		}
	}
	key, value, ok = strings.Cut(line, ":")
	return strings.TrimSpace(key), strings.TrimSpace(value), ok
}

// closingQuote returns the index of the quote that closes a double-quoted token
// at the start of line, or -1 when there is none.
func closingQuote(line string) int {
	escaped := false
	for i := 1; i < len(line); i++ {
		switch {
		case escaped:
			escaped = false
		case line[i] == '\\':
			escaped = true
		case line[i] == '"':
			return i
		}
	}
	return -1
}

// listItem reports whether line is a list item and returns its trimmed value.
// The reference implementation only recognises exactly two leading spaces, which
// silently discards items under any other indentation.
func listItem(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "- ") {
		return "", false
	}
	return strings.TrimSpace(trimmed[len("- "):]), true
}

// unquote removes the surrounding quotes from a quoted value, which the
// reference implementation never does. A value whose inner text is not a
// well-formed scalar of that quoting style is returned unchanged, so a stray
// quote cannot make the reader mangle text it does not understand.
func unquote(value string) string {
	if len(value) < 2 {
		return value
	}
	switch {
	case value[0] == '"' && value[len(value)-1] == '"':
		inner := value[1 : len(value)-1]
		if hasUnescaped(inner, '"') {
			return value
		}
		return unescapeDouble(inner)
	case value[0] == '\'' && value[len(value)-1] == '\'':
		inner := value[1 : len(value)-1]
		if strings.Contains(strings.ReplaceAll(inner, "''", ""), "'") {
			return value
		}
		return strings.ReplaceAll(inner, "''", "'")
	}
	return value
}

// hasUnescaped reports whether s contains quote that is not preceded by an odd
// number of backslashes.
func hasUnescaped(s string, quote byte) bool {
	escaped := false
	for i := 0; i < len(s); i++ {
		switch {
		case escaped:
			escaped = false
		case s[i] == '\\':
			escaped = true
		case s[i] == quote:
			return true
		}
	}
	return false
}

// unescapeDouble resolves the escapes the writer emits inside a double quoted
// value: \\, \", \n, \r, \t and a four-digit \uXXXX. Any other escape is left as
// written, so a hand-written value is not silently reinterpreted.
func unescapeDouble(inner string) string {
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] != '\\' || i+1 >= len(inner) {
			b.WriteByte(inner[i])
			continue
		}
		switch inner[i+1] {
		case '\\', '"':
			b.WriteByte(inner[i+1])
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'u':
			code, ok := hex4(inner[i+2:])
			if !ok {
				b.WriteByte(inner[i])
				b.WriteByte(inner[i+1])
				break
			}
			b.WriteRune(rune(code))
			i += 4 // the 'u' and the four digits; the loop's own two steps make six
		default:
			b.WriteByte(inner[i])
			b.WriteByte(inner[i+1])
		}
		i++
	}
	return b.String()
}

// hex4 reads exactly four hexadecimal digits.
func hex4(rest string) (uint32, bool) {
	if len(rest) < 4 {
		return 0, false
	}
	code, err := strconv.ParseUint(rest[:4], 16, 32)
	if err != nil {
		return 0, false
	}
	return uint32(code), true
}
