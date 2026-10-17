package frontmatter

import (
	"reflect"
	"testing"
)

func TestParseBlock(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Metadata
	}{
		{"plain scalar", "title: plain value", Metadata{"title": "plain value"}},
		{"colons and hashes inside a value", "title: Decision: use #1 approach", Metadata{"title": "Decision: use #1 approach"}},
		{"value starting with a dash", "title: - dash start", Metadata{"title": "- dash start"}},
		{"value starting with an asterisk", "title: *bold* start", Metadata{"title": "*bold* start"}},
		{"value starting with a hash", "title: #1 priority", Metadata{"title": "#1 priority"}},
		{"value that looks like a tag", "title: &anchor", Metadata{"title": "&anchor"}},
		{"empty value opens a list", "title:", Metadata{"title": []string{}}},
		{"empty value followed by another key", "title:\nstatus: active", Metadata{"title": []string{}, "status": "active"}},
		{"empty flow sequence means an empty value", "title: []", Metadata{"title": []string{}}},
		{"quoted empty value is an empty string", `title: ""`, Metadata{"title": ""}},
		{"quoted empty single value", "title: ''", Metadata{"title": ""}},
		{"double quoted value", `title: "double quoted"`, Metadata{"title": "double quoted"}},
		{"escaped quote inside a double quoted value", `title: "he said \"hi\""`, Metadata{"title": `he said "hi"`}},
		{"escaped backslash inside a double quoted value", `title: "a\\b"`, Metadata{"title": `a\b`}},
		{"single quoted value", "title: 'single quoted'", Metadata{"title": "single quoted"}},
		{"doubled quote inside a single quoted value", "title: 'it''s fine'", Metadata{"title": "it's fine"}},
		{"malformed double quoting is left alone", `title: "a"b"`, Metadata{"title": `"a"b"`}},
		{"malformed single quoting is left alone", "title: 'a'b'", Metadata{"title": "'a'b'"}},
		{"null is plain text", "title: null", Metadata{"title": "null"}},
		{"tilde is plain text", "title: ~", Metadata{"title": "~"}},
		{"key case and spacing are trimmed", "   title   :    spaced value   ", Metadata{"title": "spaced value"}},
		{"an indented key still parses", "  nested: value", Metadata{"nested": "value"}},
		{"inner whitespace is preserved", "title: a  b", Metadata{"title": "a  b"}},
		{"trailing whitespace is dropped", "title: trailing   ", Metadata{"title": "trailing"}},
		{"a key with a space in it", "my key: value", Metadata{"my key": "value"}},
		{"an empty key is kept", ":value", Metadata{"": "value"}},
		{"a line without a separator is ignored", "---", Metadata{}},
		{"an empty block", "", Metadata{}},
		{"a whitespace-only block", "   \n\t\n", Metadata{}},

		{"block sequence", "tags:\n  - alpha\n  - beta", Metadata{"tags": []string{"alpha", "beta"}}},
		{"four space indentation still reads", "tags:\n    - alpha\n    - beta", Metadata{"tags": []string{"alpha", "beta"}}},
		{"tab indentation still reads", "tags:\n\t- alpha\n\t- beta", Metadata{"tags": []string{"alpha", "beta"}}},
		{"no indentation still reads", "tags:\n- alpha\n- beta", Metadata{"tags": []string{"alpha", "beta"}}},
		{"blank line inside a sequence", "tags:\n  - alpha\n\n  - beta", Metadata{"tags": []string{"alpha", "beta"}}},
		{"blank line after a sequence", "tags:\n  - alpha\n\ntitle: after", Metadata{"tags": []string{"alpha"}, "title": "after"}},
		{"a line without a separator does not close the list", "tags:\n  - alpha\nplain line\n  - beta", Metadata{"tags": []string{"alpha", "beta"}}},
		{"a key with a value closes the list", "tags:\n  - alpha\nstatus: active\n  - beta", Metadata{"tags": []string{"alpha"}, "status": "active"}},
		{"an orphan item is ignored", "  - orphan", Metadata{}},
		{"an orphan item before a key", "  - orphan\ntitle: real", Metadata{"title": "real"}},
		{"an item with no value is ignored", "tags:\n  - ", Metadata{"tags": []string{}}},
		{"a bare dash is not an item", "tags:\n  -", Metadata{"tags": []string{}}},
		{"an item value containing a separator", "tags:\n  - key: value", Metadata{"tags": []string{"key: value"}}},
		{"an item value containing two separators", "tags:\n  - a:b:c", Metadata{"tags": []string{"a:b:c"}}},
		{"item values are trimmed", "tags:\n  -   padded   ", Metadata{"tags": []string{"padded"}}},
		{"an item value with a hash", "tags:\n  - fix #42", Metadata{"tags": []string{"fix #42"}}},
		{"an item value that looks like an item", "tags:\n  - - inner", Metadata{"tags": []string{"- inner"}}},
		{"a quoted item is unquoted", "tags:\n  - \"a #b\"\n  - \"true\"", Metadata{"tags": []string{"a #b", "true"}}},
		{"a quoted numeric item is unquoted", "tags:\n  - \"123\"", Metadata{"tags": []string{"123"}}},
		{"an explicitly empty quoted item survives", "tags:\n  - \"\"", Metadata{"tags": []string{""}}},

		{"a repeated key keeps the last value", "title: first\ntitle: second", Metadata{"title": "second"}},
		{"a scalar replaces an earlier list", "x: scalar\nx:\n  - a\n  - b", Metadata{"x": []string{"a", "b"}}},
		{"a list replaces an earlier scalar", "x:\n  - a\n  - b\nx: scalar", Metadata{"x": "scalar"}},
		{"a reopened list replaces the earlier items", "x:\n  - a\nx:\n  - b", Metadata{"x": []string{"b"}}},

		{"an exotic separator does not truncate a value", "title: a\u2028b", Metadata{"title": "a\u2028b"}},
		{"a vertical tab does not truncate a value", "title: a\vb", Metadata{"title": "a\vb"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBlock(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseBlock(%q)\n got: %#v\nwant: %#v", tt.raw, got, tt.want)
			}
		})
	}
}
