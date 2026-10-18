package frontmatter

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// The rendered shape is a contract: every known field is always written, in a
// fixed order, and unknown keys follow in name order.
func TestMarshalFieldOrderAndShape(t *testing.T) {
	doc := Document{
		Metadata: Metadata{
			"title": "T", "id": "I", "kind": "spec", "zzz": "last", "aaa": "first",
		},
		Body: "body\n",
	}
	want := "---\n" +
		"id: I\n" +
		"kind: spec\n" +
		"title: T\n" +
		"status: \"\"\n" +
		"created: \"\"\n" +
		"updated: \"\"\n" +
		"repos: []\n" +
		"tags: []\n" +
		"codex_sessions: []\n" +
		"aaa: first\n" +
		"zzz: last\n" +
		"---\n" +
		"body\n"

	if got := string(Marshal(doc)); got != want {
		t.Errorf("Marshal rendered:\n%q\nwant:\n%q", got, want)
	}
}

func TestMarshalQuotesValuesThatWouldChangeMeaning(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"plain text", "Quality analysis display contract", "Quality analysis display contract"},
		{"empty", "", `""`},
		{"leading space", " lead", `" lead"`},
		{"trailing space", "trail ", `"trail "`},
		{"inner colons", "Fix #42: the thing", `"Fix #42: the thing"`},
		{"trailing colon", "ends:", `"ends:"`},
		{"leading hash", "#1 priority", `"#1 priority"`},
		{"leading dash", "- dash start", `"- dash start"`},
		{"leading asterisk", "*bold* start", `"*bold* start"`},
		{"leading ampersand", "&anchor", `"&anchor"`},
		{"inner comment start", "a #b", `"a #b"`},
		{"leading bracket", "[draft]", `"[draft]"`},
		{"a quote alone", `"`, `"\""`},
		{"inner quotes need no quoting", `He said "hi"`, `He said "hi"`},
		{"reserved true", "true", `"true"`},
		{"reserved mixed case", "TRUE", `"TRUE"`},
		{"reserved null", "null", `"null"`},
		{"reserved tilde", "~", `"~"`},
		{"reserved no", "no", `"no"`},
		{"integer", "42", `"42"`},
		{"negative", "-5", `"-5"`},
		{"float", "1.5", `"1.5"`},
		{"hex", "0x1f", `"0x1f"`},
		{"exponent", "1e3", `"1e3"`},
		{"infinity", ".inf", `".inf"`},
		{"tab", "a\tb", `"a\tb"`},
		{"newline", "line1\nline2", `"line1\nline2"`},
		{"carriage return", "a\rb", `"a\rb"`},
		{"exotic separator", "a\u2028b", `"a\u2028b"`},
		{"control character", "a\x07b", `"a\u0007b"`},
		{"backslash", `a\b`, `a\b`},
		{"slug", "spec-20261017-a-slug", "spec-20261017-a-slug"},
		{"version", "v0.1.0", "v0.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderScalar(tt.value); got != tt.want {
				t.Errorf("renderScalar(%q) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

// Dates stay bare on purpose: created and updated are written that way, and
// quoting every document's dates would trade a stable byte form for a type
// distinction nothing consumes.
func TestMarshalLeavesDatesUnquoted(t *testing.T) {
	for _, value := range []string{"2026-10-17", "2026-10-17T21:35:40Z"} {
		if got := renderScalar(value); got != value {
			t.Errorf("renderScalar(%q) = %s, want it unquoted", value, got)
		}
	}
}

// The real invariant: a value survives a trip through the writer and the reader.
func TestRoundTripThroughParse(t *testing.T) {
	values := []string{
		"", "plain", " lead", "trail ", "  both  ", "Fix #42: the thing", "#1 priority",
		"- dash start", "*bold* start", "&anchor", "a #b", "ends:", `He said "hi"`,
		`"`, `"quoted"`, "true", "null", "~", "42", "1.5", "0x1f", ".inf", "a\tb",
		"line1\nline2", "line1\r\nline2", `a\b`, `a\"b`, "2026-10-17",
		"spec-20261017-a-slug", "质量分析展示契约", "emoji 🚀 rocket", "a\u2028b", "a\x07b",
		"---", "a\n---\nb",
	}
	for _, value := range values {
		t.Run(strings.ReplaceAll(value, "\n", "\\n"), func(t *testing.T) {
			marshalled := Marshal(Document{Metadata: Metadata{"title": value}, Body: "body\n"})
			doc, err := Parse("specs/x.md", marshalled)
			if err != nil {
				t.Fatalf("Parse(%q): %v", marshalled, err)
			}
			if got := doc.Title(); got != value {
				t.Errorf("round trip lost the value:\n got: %q\nwant: %q", got, value)
			}
			if doc.Body != "body\n" {
				t.Errorf("round trip lost the body: %q", doc.Body)
			}
		})
	}
}

// A value cannot be allowed to break out of the frontmatter block it sits in.
func TestMarshalEscapesLineBreaksSoTheyCannotInjectFrontmatter(t *testing.T) {
	hostile := "innocent\n---\nid: injected"
	marshalled := string(Marshal(Document{Metadata: Metadata{"title": hostile}, Body: "body\n"}))

	if got := strings.Count(marshalled, "\n---\n"); got != 1 {
		t.Errorf("the document has %d closing fences, want 1:\n%s", got, marshalled)
	}
	doc, err := Parse("specs/x.md", []byte(marshalled))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := doc.Title(); got != hostile {
		t.Errorf("Title() = %q, want %q", got, hostile)
	}
	if got := doc.ID(); got == "injected" {
		t.Error("the injected id won")
	}
}

func TestMarshalPreservesUnknownKeys(t *testing.T) {
	doc := Document{
		Metadata: Metadata{
			"id":    "spec-x",
			"zzz":   "scalar",
			"aaa":   []string{"one", "two"},
			"empty": []string{},
		},
		Body: "",
	}
	got := string(Marshal(doc))

	for _, want := range []string{"aaa:\n  - one\n  - two\n", "zzz: scalar\n", "empty: []\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered document is missing %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "aaa:") > strings.Index(got, "zzz:") {
		t.Errorf("unknown keys are not in name order:\n%s", got)
	}

	back, err := Parse("specs/x.md", []byte(got))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !sameStrings(back.Metadata["aaa"].([]string), []string{"one", "two"}) {
		t.Errorf("the unknown list did not survive: %#v", back.Metadata["aaa"])
	}
	if back.Metadata["zzz"] != "scalar" {
		t.Errorf("the unknown scalar did not survive: %#v", back.Metadata["zzz"])
	}
}

// Writing twice must be byte-identical, including for values that were quoted.
func TestMarshalIsIdempotent(t *testing.T) {
	documents := []Document{
		{Metadata: Metadata{"title": "Fix #42: the thing", "id": "spec-x"}, Body: "body\n"},
		{Metadata: Metadata{"title": "", "tags": []string{"a #b", "true"}}, Body: ""},
		{Metadata: Metadata{"title": "plain", "zzz": "extra"}, Body: "no trailing newline"},
		{Metadata: Metadata{}, Body: "\n"},
	}
	for i, doc := range documents {
		once := Marshal(doc)
		parsed, err := Parse("specs/x.md", once)
		if err != nil {
			t.Fatalf("document %d: Parse: %v", i, err)
		}
		twice := Marshal(parsed)
		if string(once) != string(twice) {
			t.Errorf("document %d is not byte-stable:\nfirst:  %q\nsecond: %q", i, once, twice)
		}
	}
}

// A rewrite must not blank a document whose metadata lacks id, kind or title:
// Marshal writes what the accessors read, so the fallback survives the round trip.
func TestMarshalKeepsFallbackIdentity(t *testing.T) {
	tests := []struct {
		name string
		path string
		kind string
	}{
		{"spec directory", "specs/2026-10-17-example.md", "spec"},
		{"decision directory", "decisions/2026-10-17-example.md", "decision"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := Document{Name: tt.path, Metadata: Metadata{}, Body: "body\n"}
			marshalled := Marshal(before)

			for _, want := range []string{
				"id: 2026-10-17-example\n",
				"kind: " + tt.kind + "\n",
				"title: 2026-10-17-example\n",
			} {
				if !strings.Contains(string(marshalled), want) {
					t.Errorf("marshalled document is missing %q:\n%s", want, marshalled)
				}
			}

			after, err := Parse(tt.path, marshalled)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if after.ID() != before.ID() || after.Kind() != before.Kind() || after.Title() != before.Title() {
				t.Errorf("a rewrite changed the identity: id %q→%q, kind %q→%q, title %q→%q",
					before.ID(), after.ID(), before.Kind(), after.Kind(), before.Title(), after.Title())
			}
		})
	}
}

// A title that is present but empty stays empty: it must not gain the file stem.
func TestMarshalWritesPresentEmptyTitleAsEmpty(t *testing.T) {
	doc := Document{Name: "specs/2026-10-17-example.md", Metadata: Metadata{"title": ""}}
	if got := string(Marshal(doc)); !strings.Contains(got, "title: \"\"\n") {
		t.Errorf("empty title was not written as title: \"\":\n%s", got)
	}
}

// The writer must not emit a key line its own reader misreads: a key beginning
// "- " is quoted, so the unknown key "- 0" is not read back after the rewrite as
// an item of the list field before it. This is counterexample 65c2fd0e.
func TestMarshalQuotesAKeyItsReaderWouldMisread(t *testing.T) {
	before, err := Parse("specs/fuzz.md", []byte("---\n0\n- 0:\n---\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	marshalled := Marshal(before)
	if !strings.Contains(string(marshalled), `"- 0": []`+"\n") {
		t.Errorf("the dash-leading key was not quoted:\n%s", marshalled)
	}

	after, err := Parse("specs/fuzz.md", marshalled)
	if err != nil {
		t.Fatalf("re-parsing our own output: %v", err)
	}
	if got := after.Metadata["- 0"]; !reflect.DeepEqual(got, []string{}) {
		t.Errorf(`the unknown key "- 0" did not survive: metadata %#v`, after.Metadata)
	}
	if got := after.CodexSessions(); len(got) != 0 {
		t.Errorf("codex_sessions gained %q", got)
	}
}

// Rewriting a document must not lose any value — the bytes will differ where the
// writer quotes or spells things differently, but every field and the whole body
// have to come back the same, or a rewrite silently damages a vault.
func TestRewriteKeepsEveryValue(t *testing.T) {
	for i, doc := range sampleDocuments {
		t.Run(fmt.Sprintf("sample %d", i), func(t *testing.T) {
			before, err := Parse("specs/sample.md", []byte(doc))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			after, err := Parse("specs/sample.md", Marshal(before))
			if err != nil {
				t.Fatalf("re-parsing our own output: %v", err)
			}
			if got, want := fields(after), fields(before); !reflect.DeepEqual(got, want) {
				t.Errorf("a rewrite changed the document:\n got: %#v\nwant: %#v", got, want)
			}
		})
	}
}

// fields collects everything a caller can read from a document, which is what a
// rewrite has to preserve.
func fields(doc Document) map[string]any {
	return map[string]any{
		"id": doc.ID(), "kind": doc.Kind(), "title": doc.Title(), "status": doc.Status(),
		"created": doc.Created(), "updated": doc.Updated(), "repos": doc.Repos(),
		"tags": doc.Tags(), "sessions": doc.CodexSessions(), "body": doc.Body,
	}
}
