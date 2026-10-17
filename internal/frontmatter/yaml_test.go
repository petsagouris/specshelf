package frontmatter

import (
	"fmt"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// A third-party tool reads our frontmatter with a general YAML parser, so the
// writer has to emit YAML that is valid *and* that a parser reads back as the
// strings we put in. The quoting rules are what make the difference: without
// them "true" is a boolean, "123" a number, "#x" a comment and "- x" a sequence.
// Weakening a rule has to fail here, or the rules are decoration.
func TestEmittedFrontmatterIsValidYAML(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
	}{
		{"plain", Metadata{"title": "Quality analysis display contract"}},
		{"colons and a hash", Metadata{"title": "Fix #42: the thing"}},
		{"leading hash", Metadata{"title": "#1 priority"}},
		{"leading dash", Metadata{"title": "- dash start"}},
		{"leading asterisk", Metadata{"title": "*bold* start"}},
		{"inner quotes", Metadata{"title": `He said "hi"`}},
		{"a lone quote", Metadata{"title": `"`}},
		{"empty title", Metadata{"title": ""}},
		{"trailing colon", Metadata{"title": "ends:"}},
		{"leading and trailing space", Metadata{"title": "  padded  "}},
		{"a boolean word", Metadata{"title": "true"}},
		{"a number", Metadata{"title": "123"}},
		{"a float", Metadata{"title": "1.5"}},
		{"a null word", Metadata{"title": "null"}},
		{"a newline", Metadata{"title": "line1\nline2"}},
		{"a carriage return", Metadata{"title": "a\rb"}},
		{"a tab", Metadata{"title": "a\tb"}},
		{"a backslash", Metadata{"title": `a\b`}},
		{"non-ASCII", Metadata{"title": "质量分析展示契约"}},
		{"an emoji", Metadata{"title": "🚀 ship it"}},
		{"an exotic separator", Metadata{"title": "a\u2028b"}},
		{"a control character", Metadata{"title": "a\x07b"}},
		{"everything set", Metadata{
			"id": "spec-20261017-a-slug", "kind": "spec", "title": "Fix #42: the thing",
			"status": "active", "created": "2026-10-17", "updated": "2026-10-18",
			"repos": []string{"has #hash", "true"}, "tags": []string{"123"},
			"codex_sessions": []string{},
		}},
		{"an empty item", Metadata{"tags": []string{"", "real"}}},
		{"empty lists", Metadata{"repos": []string{}, "tags": []string{}, "codex_sessions": []string{}}},
		{"unknown keys", Metadata{
			"title": "t", "zzz": "a scalar that needs no quotes", "aaa": []string{"one", "two"},
		}},
		{"an unknown key with an unusual value", Metadata{"title": "t", "count": 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := decodeFrontmatter(t, emittedFrontmatter(t, tt.metadata))
			checkReadsBackAsWritten(t, mapping, tt.metadata)
		})
	}
}

// isListField reports whether a key is one of the known list fields, whose value
// has to be a sequence whether or not the metadata set one.
func isListField(key string) bool {
	for _, known := range listFields {
		if known == key {
			return true
		}
	}
	return false
}

// emittedFrontmatter returns the frontmatter block the writer produces for a
// metadata map, with an empty body.
func emittedFrontmatter(t *testing.T, metadata Metadata) []byte {
	t.Helper()
	text := string(Marshal(Document{Metadata: metadata}))
	if !strings.HasPrefix(text, "---\n") {
		t.Fatalf("the document does not open with a fence:\n%s", text)
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		t.Fatalf("the document has no closing fence:\n%s", text)
	}
	return []byte(rest[:end])
}

// decodeFrontmatter parses a block with a real YAML parser and returns its
// mapping node.
func decodeFrontmatter(t *testing.T, block []byte) *yaml.Node {
	t.Helper()
	var document yaml.Node
	if err := yaml.Unmarshal(block, &document); err != nil {
		t.Fatalf("the frontmatter we emitted is not valid YAML: %v\nblock:\n%s", err, block)
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		t.Fatalf("expected one document node, got kind %v with %d children", document.Kind, len(document.Content))
	}
	if mapping := document.Content[0]; mapping.Kind != yaml.MappingNode {
		t.Fatalf("expected a mapping, got kind %v", mapping.Kind)
	} else {
		return mapping
	}
	return nil
}

// checkReadsBackAsWritten compares the parsed mapping with the metadata it came
// from, and complains about anything the writer dropped.
func checkReadsBackAsWritten(t *testing.T, mapping *yaml.Node, metadata Metadata) {
	t.Helper()
	seen := make(map[string]bool, len(mapping.Content)/2)

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i].Value, mapping.Content[i+1]
		seen[key] = true

		if want, isList := metadata[key].([]string); isList || isListField(key) {
			if !isList {
				want = nil
			}
			if value.Kind != yaml.SequenceNode {
				t.Errorf("%s does not read back as a list: kind %v, tag %s", key, value.Kind, value.Tag)
				continue
			}
			got := make([]string, 0, len(value.Content))
			for _, item := range value.Content {
				got = append(got, item.Value)
			}
			if !sameStrings(got, want) {
				t.Errorf("%s reads back as %q, want %q", key, got, want)
			}
			continue
		}

		want := ""
		if raw, ok := metadata[key]; ok {
			want = fmt.Sprint(raw)
		}
		if value.Value != want {
			t.Errorf("%s reads back as %q, want %q", key, value.Value, want)
		}
		// created and updated are dates on purpose, and are the one pair of
		// fields we let a parser type; everything else has to stay a string.
		if key != "created" && key != "updated" && value.Tag != "!!str" {
			t.Errorf("%s reads back as %s, not as a string", key, value.Tag)
		}
	}

	for _, key := range append(append([]string{}, scalarFields...), listFields...) {
		if !seen[key] {
			t.Errorf("the writer did not emit the known field %s", key)
		}
	}
	for key := range metadata {
		if !seen[key] {
			t.Errorf("the writer dropped %s", key)
		}
	}
}
