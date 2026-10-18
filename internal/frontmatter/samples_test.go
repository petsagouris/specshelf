package frontmatter

import "slices"

// sameStrings reports whether two string slices hold the same values in order.
func sameStrings(a, b []string) bool { return slices.Equal(a, b) }

// sampleDocuments are complete documents in the written form: what a vault
// actually holds, including the awkward values the writer has to quote and the
// keys a document may carry beyond the nine known ones. They feed the rewrite
// test and the fuzz seeds, so the reader is exercised on documents shaped like
// the ones on disk rather than on fragments alone.
var sampleDocuments = []string{
	// Every field populated, with block-sequence lists.
	"---\nid: spec-20261017-quality-analysis-display-contract\nkind: spec\ntitle: Quality analysis display contract\nstatus: active\ncreated: 2026-10-17\nupdated: 2026-10-18\nrepos:\n  - billing\n  - web\ntags:\n  - reliability\ncodex_sessions:\n  - session-a\n  - session-b\n---\n# Quality analysis display contract\n\n## Background\n\nThe summary line.\n",

	// A title a YAML reader would misread, quoted by the writer, and empty lists.
	"---\nid: spec-20261017-fix-42-the-thing\nkind: spec\ntitle: \"Fix #42: the thing\"\nstatus: draft\ncreated: 2026-10-17\nupdated: 2026-10-17\nrepos: []\ntags: []\ncodex_sessions: []\n---\n# Fix #42: the thing\n\nBody.\n",

	// A non-ASCII title, written back literally.
	"---\nid: spec-20261017-cjk\nkind: spec\ntitle: 质量分析展示契约\nstatus: draft\ncreated: 2026-10-17\nupdated: 2026-10-17\nrepos:\ntags:\ncodex_sessions:\n---\n# 质量分析展示契约\n\n中文摘要行。\n",

	// A decision carrying keys beyond the nine known ones.
	"---\nid: decision-20261017-use-markdown\nkind: decision\ntitle: Use Markdown\nstatus: accepted\ncreated: 2026-10-17\nupdated: 2026-10-17\nrepos:\ntags:\ncodex_sessions:\n  - session-a\nx_custom: kept\n\"a: b\": \"quoted: value\"\n---\n# Use Markdown\n",

	// An empty title, which stays empty rather than decaying into a list.
	"---\nid: spec-20261017-empty-title\nkind: spec\ntitle:\nstatus:\ncreated: 2026-10-17\nupdated: 2026-10-17\nrepos: []\ntags: []\ncodex_sessions: []\n---\n# An empty title document\n",

	// CRLF line endings, which the reader normalises to LF.
	"---\r\nid: spec-20261017-windows\r\nkind: spec\r\ntitle: Written on Windows\r\nstatus: draft\r\ncreated: 2026-10-17\r\nupdated: 2026-10-17\r\nrepos: []\r\ntags: []\r\ncodex_sessions: []\r\n---\r\n# Written on Windows\r\n\r\nBody with CRLF.\r\n",
}
