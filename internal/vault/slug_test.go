package vault

import (
	"fmt"
	"testing"
)

// TestSlugify pins the slug rules in docs/FORMAT.md over the titles that have
// always been the awkward ones: accents, CJK, punctuation-only, over-long, and
// empty. A slug is part of a document's id and file name, so these values are
// contract — changing one renames documents already on disk.
func TestSlugify(t *testing.T) {
	cases := []struct{ title, slug string }{
		{"Quality analysis display contract", "quality-analysis-display-contract"},
		{"质量分析展示契约", "992681441c"},
		{"不用 Beads 管历史 spec", "beads-spec"},
		{"Über Straße", "uber-strae"},
		{"Fix #42: the thing", "fix-42-the-thing"},
		{"Café résumé naïve", "cafe-resume-naive"},
		{"Ångström Øresund", "angstrom-resund"},
		{"Привет мир", "6b49ee5b05"},
		{"日本語 mixed Latin", "mixed-latin"},
		{"emoji 🚀 rocket", "emoji-rocket"},
		{"!!!???", "8b8a79815e"},
		{"Trailing   spaces   and---dashes", "trailing-spaces-and-dashes"},
		{
			"a very long title that goes on and on and on and on and on and on and on and on and on",
			"a-very-long-title-that-goes-on-and-on-and-on-and-on-and-on-and-on-and-on",
		},
		{"", "da39a3ee5e"},
		{"Ünïcödé mixed with trailing punctuation!!", "unicode-mixed-with-trailing-punctuation"},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%02d/%s", i, c.title), func(t *testing.T) {
			if got := slugify(c.title); got != c.slug {
				t.Errorf("slugify(%q) = %q, want %q", c.title, got, c.slug)
			}
		})
	}
}
