package docs

import (
	"os"
	"path/filepath"
	"testing"
)

// The constants are the bytes a new vault is seeded with, so they are user-visible
// content: a change to a template changes every vault created afterwards. These
// pinned copies make such a change show up as a diff in review instead of
// shipping silently.
func TestTemplatesMatchPinnedBytes(t *testing.T) {
	tests := []struct {
		name   string
		pinned string
		got    string
	}{
		{"spec template", "spec-template.md", SpecTemplate},
		{"decision template", "decision-template.md", DecisionTemplate},
		{"vault readme", "vault-readme.md", VaultReadme},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", tt.pinned))
			if err != nil {
				t.Fatalf("read pinned bytes: %v", err)
			}
			if tt.got != string(want) {
				t.Errorf("constant differs from testdata/%s\ngot  %q\nwant %q", tt.pinned, tt.got, want)
			}
		})
	}
}
