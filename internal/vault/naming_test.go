package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalDate(t *testing.T) {
	// 23:30 UTC is already the next day in UTC+13, so the local date and the
	// UTC date must differ.
	utc := time.Date(2026, 10, 17, 23, 30, 0, 0, time.UTC)
	if got := localDate(utc); got != "2026-10-17" {
		t.Errorf("localDate(%v) = %q, want %q", utc, got, "2026-10-17")
	}

	plus13 := time.FixedZone("UTC+13", 13*60*60)
	if got := localDate(utc.In(plus13)); got != "2026-10-18" {
		t.Errorf("localDate(%v) = %q, want %q", utc.In(plus13), got, "2026-10-18")
	}
}

func TestCompactDate(t *testing.T) {
	if got := compactDate("2026-10-17"); got != "20261017" {
		t.Errorf("compactDate(%q) = %q, want %q", "2026-10-17", got, "20261017")
	}
}

func TestDocumentID(t *testing.T) {
	cases := []struct {
		kind, date, slug, want string
	}{
		{"spec", "2026-10-17", "quality-analysis-display-contract", "spec-20261017-quality-analysis-display-contract"},
		{"decision", "2026-10-17", "adopt-cobra", "decision-20261017-adopt-cobra"},
		// The sha1 fallback slug, as slugify returns it for a non-ASCII title.
		{"spec", "2026-10-17", "992681441c", "spec-20261017-992681441c"},
	}
	for _, c := range cases {
		if got := documentID(c.kind, c.date, c.slug); got != c.want {
			t.Errorf("documentID(%q, %q, %q) = %q, want %q", c.kind, c.date, c.slug, got, c.want)
		}
	}
}

func TestUniqueID(t *testing.T) {
	cases := []struct {
		name  string
		base  string
		taken map[string]bool
		want  string
	}{
		{"no collision", "spec-20261017-a-slug", map[string]bool{}, "spec-20261017-a-slug"},
		{"one collision", "spec-20261017-a-slug", map[string]bool{"spec-20261017-a-slug": true}, "spec-20261017-a-slug-2"},
		{"three collisions", "spec-20261017-a-slug", map[string]bool{
			"spec-20261017-a-slug":   true,
			"spec-20261017-a-slug-2": true,
			"spec-20261017-a-slug-3": true,
		}, "spec-20261017-a-slug-4"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := uniqueID(c.base, func(s string) bool { return c.taken[s] })
			if got != c.want {
				t.Errorf("uniqueID(%q, taken) = %q, want %q", c.base, got, c.want)
			}
		})
	}
}

func TestDocumentPath(t *testing.T) {
	datadir := filepath.Join("vault", "specs")
	want := filepath.Join(datadir, "2026-10-17-a-slug.md")
	if got := documentPath(datadir, "2026-10-17", "a-slug"); got != want {
		t.Errorf("documentPath(%q, %q, %q) = %q, want %q", datadir, "2026-10-17", "a-slug", got, want)
	}
}

func TestNextFreePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.md")

	// An absent path comes back unchanged and nothing is created.
	if got := nextFreePath(path); got != path {
		t.Fatalf("nextFreePath(%q) = %q, want %q", path, got, path)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("nextFreePath created %q: stat err = %v", path, err)
	}

	first := []byte("first\n")
	if err := os.WriteFile(path, first, 0o644); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(dir, "x-2.md")
	if got := nextFreePath(path); got != second {
		t.Fatalf("nextFreePath(%q) = %q, want %q", path, got, second)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != string(first) {
		t.Fatalf("pre-existing %q changed: %q, %v", path, got, err)
	}

	secondBytes := []byte("second\n")
	if err := os.WriteFile(second, secondBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	third := filepath.Join(dir, "x-3.md")
	if got := nextFreePath(path); got != third {
		t.Fatalf("nextFreePath(%q) = %q, want %q", path, got, third)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != string(first) {
		t.Fatalf("pre-existing %q changed: %q, %v", path, got, err)
	}
	if got, err := os.ReadFile(second); err != nil || string(got) != string(secondBytes) {
		t.Fatalf("pre-existing %q changed: %q, %v", second, got, err)
	}
}
