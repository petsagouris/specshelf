package frontmatter

import "strings"

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
//     well-formed quoted value is unquoted
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
			if item != "" {
				items, _ := metadata[listKey].([]string)
				metadata[listKey] = append(items, item)
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch {
		case value == "" || value == "[]":
			metadata[key] = []string{}
			listKey = key
		default:
			metadata[key] = unquote(value)
			listKey = ""
		}
	}
	return metadata
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

// unescapeDouble resolves the two escapes the writer emits inside a double
// quoted value: \\ and \".
func unescapeDouble(inner string) string {
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) && (inner[i+1] == '\\' || inner[i+1] == '"') {
			b.WriteByte(inner[i+1])
			i++
			continue
		}
		b.WriteByte(inner[i])
	}
	return b.String()
}
