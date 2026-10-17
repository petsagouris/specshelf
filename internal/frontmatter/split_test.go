package frontmatter

import (
	"errors"
	"strings"
	"testing"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantRaw  string
		wantHas  bool
		wantBody string
	}{
		{
			name:     "no frontmatter",
			input:    "just a body\n",
			wantBody: "just a body\n",
		},
		{
			name:     "blank line before the fence is not frontmatter",
			input:    "\n---\nid: x\n---\nbody",
			wantBody: "\n---\nid: x\n---\nbody",
		},
		{
			name:     "a bare fence is not a frontmatter block",
			input:    "---",
			wantBody: "---",
		},
		{
			name:     "standard document",
			input:    "---\nid: x\n---\n\nbody\n",
			wantRaw:  "id: x",
			wantHas:  true,
			wantBody: "\nbody\n",
		},
		{
			name:     "several fields",
			input:    "---\nid: x\ntitle: hello\n---\nbody",
			wantRaw:  "id: x\ntitle: hello",
			wantHas:  true,
			wantBody: "body",
		},
		{
			name:     "empty block",
			input:    "---\n---\nbody",
			wantHas:  true,
			wantBody: "body",
		},
		{
			name:     "body containing a horizontal rule",
			input:    "---\nid: x\n---\nbody\n---\nmore\n",
			wantRaw:  "id: x",
			wantHas:  true,
			wantBody: "body\n---\nmore\n",
		},
		{
			name:     "windows line endings",
			input:    "---\r\nid: x\r\n---\r\n\r\nbody\r\n",
			wantRaw:  "id: x",
			wantHas:  true,
			wantBody: "\nbody\n",
		},
		{
			name:     "lone carriage returns",
			input:    "---\rid: x\r---\rbody",
			wantRaw:  "id: x",
			wantHas:  true,
			wantBody: "body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, has, body, err := splitFrontmatter("specs/example.md", []byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if raw != tt.wantRaw {
				t.Errorf("raw = %q, want %q", raw, tt.wantRaw)
			}
			if has != tt.wantHas {
				t.Errorf("hasFrontmatter = %v, want %v", has, tt.wantHas)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestSplitFrontmatterUnterminated(t *testing.T) {
	tests := []struct{ name, input string }{
		{"fence with a field and no closing fence", "---\nid: x\n"},
		{"fence at the end of the file", "---\n"},
		{"fence with a final field and no newline", "---\nid: x"},
		{"windows line endings and no closing fence", "---\r\nid: x\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := splitFrontmatter("specs/broken.md", []byte(tt.input))
			if !errors.Is(err, ErrUnterminated) {
				t.Fatalf("error = %v, want ErrUnterminated", err)
			}
			if !strings.Contains(err.Error(), "specs/broken.md") {
				t.Errorf("error %q does not name the document", err)
			}
		})
	}
}
