package frontmatter

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const fixtureSpec = `---
id: spec-20261017-quality-analysis-display-contract
kind: spec
title: Quality analysis display contract
status: active
created: 2026-10-17
updated: 2026-10-17
repos:
  - lippi-smart-customer
  - aics-web-repos
tags:
  - quality-analysis
  - contract
codex_sessions:
  - session-a
  - session-b
---
# Quality analysis display contract

## Background
`

func TestParseFixtureDocument(t *testing.T) {
	doc, err := Parse("specs/2026-10-17-quality-analysis-display-contract.md", []byte(fixtureSpec))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"ID", doc.ID(), "spec-20261017-quality-analysis-display-contract"},
		{"Kind", doc.Kind(), "spec"},
		{"Title", doc.Title(), "Quality analysis display contract"},
		{"Status", doc.Status(), "active"},
		{"Created", doc.Created(), "2026-10-17"},
		{"Updated", doc.Updated(), "2026-10-17"},
		{"Repos", doc.Repos(), []string{"lippi-smart-customer", "aics-web-repos"}},
		{"Tags", doc.Tags(), []string{"quality-analysis", "contract"}},
		{"CodexSessions", doc.CodexSessions(), []string{"session-a", "session-b"}},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s = %#v, want %#v", c.name, c.got, c.want)
		}
	}
	if !strings.Contains(doc.Body, "## Background") {
		t.Errorf("body = %q, want it to keep the document body", doc.Body)
	}
}

// The reference implementation writes titles a YAML parser would misread; the
// reader must hand them back unchanged.
func TestParseTitleThatLooksLikeYAML(t *testing.T) {
	for _, title := range []string{"#1 priority", "- dash start", "*bold* start", `He said "hi"`, "Fix #42: the thing"} {
		raw := "---\ntitle: " + title + "\n---\nbody\n"
		doc, err := Parse("specs/x.md", []byte(raw))
		if err != nil {
			t.Fatalf("Parse(%q): %v", title, err)
		}
		if got := doc.Title(); got != title {
			t.Errorf("Title() = %q, want %q", got, title)
		}
	}
}

func TestParseWithoutFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantStem string
		wantKind string
	}{
		{"spec directory", "specs/2026-10-17-orphan.md", "2026-10-17-orphan", "spec"},
		{"decision directory", "decisions/2026-10-17-orphan.md", "2026-10-17-orphan", "decision"},
		{"no directory at all", "orphan.md", "orphan", "spec"},
		{"another directory", "notes/2026-10-17-orphan.md", "2026-10-17-orphan", "spec"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse(tt.path, []byte("# A title\n\nbody\n"))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := doc.ID(); got != tt.wantStem {
				t.Errorf("ID() = %q, want %q", got, tt.wantStem)
			}
			if got := doc.Title(); got != tt.wantStem {
				t.Errorf("Title() = %q, want %q", got, tt.wantStem)
			}
			if got := doc.Kind(); got != tt.wantKind {
				t.Errorf("Kind() = %q, want %q", got, tt.wantKind)
			}
			if doc.Status() != "" || doc.Repos() != nil || doc.Tags() != nil {
				t.Error("a document without frontmatter must have no other fields")
			}
			if !strings.Contains(doc.Body, "# A title") {
				t.Errorf("body = %q, want the whole file", doc.Body)
			}
		})
	}
}

// An empty title is empty: not the file stem, and not the "[]" the reference
// implementation renders an empty list as.
func TestEmptyValuesAreEmpty(t *testing.T) {
	for _, raw := range []string{
		"---\ntitle:\n---\nbody\n",
		"---\ntitle: []\n---\nbody\n",
		"---\ntitle: \"\"\n---\nbody\n",
		"---\ntitle: ''\n---\nbody\n",
	} {
		doc, err := Parse("specs/2026-10-17-empty-title.md", []byte(raw))
		if err != nil {
			t.Fatalf("Parse(%q): %v", raw, err)
		}
		if got := doc.Title(); got != "" {
			t.Errorf("Title() for %q = %q, want empty", raw, got)
		}
	}
}

func TestAbsentFieldsAreEmpty(t *testing.T) {
	doc, err := Parse("specs/x.md", []byte("---\nid: spec-x\n---\nbody\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if doc.Status() != "" || doc.Created() != "" || doc.Updated() != "" {
		t.Error("absent scalar fields must read as empty")
	}
	if doc.Title() != "x" {
		t.Errorf("Title() = %q, want the file stem as the fallback", doc.Title())
	}
	if doc.Repos() != nil || doc.Tags() != nil || doc.CodexSessions() != nil {
		t.Error("absent list fields must read as nil")
	}
}

// A field whose shape does not match its meaning yields nothing rather than a
// guess; `doctor` is what reports the mismatch.
func TestShapeMismatchesYieldNothing(t *testing.T) {
	doc, err := Parse("specs/x.md", []byte("---\ntitle:\n  - a title that became a list\ntags: a single tag\n---\nbody\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := doc.Title(); got != "" {
		t.Errorf("Title() = %q, want empty for a list where a scalar belongs", got)
	}
	if got := doc.Tags(); got != nil {
		t.Errorf("Tags() = %#v, want nil for a scalar where a list belongs", got)
	}
}

func TestParseErrors(t *testing.T) {
	for _, raw := range []string{"---\nid: x\n", "---\n"} {
		doc, err := Parse("specs/broken.md", []byte(raw))
		if !errors.Is(err, ErrUnterminated) {
			t.Fatalf("Parse(%q) error = %v, want ErrUnterminated", raw, err)
		}
		if !reflect.DeepEqual(doc, Document{}) {
			t.Errorf("a failed parse must not return a partial document: %#v", doc)
		}
	}
}

func TestParseNormalisesInput(t *testing.T) {
	for _, tt := range []struct{ name, raw string }{
		{"crlf", "---\r\ntitle: windows\r\n---\r\nbody\r\n"},
		{"byte-order mark", "\ufeff---\ntitle: bom\n---\nbody\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse("specs/x.md", []byte(tt.raw))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			want := map[string]string{"crlf": "windows", "byte-order mark": "bom"}[tt.name]
			if got := doc.Title(); got != want {
				t.Errorf("Title() = %q, want %q", got, want)
			}
			if doc.Metadata == nil {
				t.Error("Metadata must never be nil")
			}
		})
	}
}
