# Vault format

A vault is a directory of Markdown documents. `specshelf` reads and writes it
byte-compatibly with the Python reference implementation, `agent-spec-vault`.
This file describes what is on disk. Where the two tools differ on purpose, the
difference is enumerated in [`DIVERGENCES.md`](DIVERGENCES.md) rather than
repeated here.

## Directory layout

```text
<vault>/
├── README.md
├── index.jsonl
├── decisions/
│   └── YYYY-MM-DD-<slug>.md
├── specs/
│   └── YYYY-MM-DD-<slug>.md
└── templates/
    ├── decision.md
    └── spec.md
```

`init` creates `specs/`, `decisions/` and `templates/`, then `templates/spec.md`,
`templates/decision.md` and `README.md`, each one only when it is missing, so a
user-edited template or README survives. `index.jsonl` is produced by the index
step, not by `init`.

## Frontmatter

A document opens with a frontmatter block: a `---` line, the fields, and a
closing `---` line. The fields are written in this order, and all nine are always
present, even when empty:

| Field | Shape |
| --- | --- |
| `id` | scalar |
| `kind` | scalar, `spec` or `decision` |
| `title` | scalar |
| `status` | scalar |
| `created` | scalar, `YYYY-MM-DD` |
| `updated` | scalar, `YYYY-MM-DD` |
| `repos` | list |
| `tags` | list |
| `codex_sessions` | list |

After the nine known fields, any unrecognised key is preserved and written after
them in name order. A list field is a block sequence, indented two spaces; an
empty list is written `[]`, never as a bare `key:` line. A scalar whose text
would otherwise change meaning is written double-quoted — an empty value, a value
with leading or trailing whitespace, one containing `: `, one starting with `#`,
one a YAML reader would take as another type, and so on. `created` and `updated`
are written bare. The result parses as valid YAML; the reader additionally
accepts the looser dialect the reference tool emits, and that tolerance is one of
the entries in `DIVERGENCES.md`.

### Sample document

The block below is a complete document in written form. It is the input to
`internal/vault/format_doc_test.go`, which parses it with the real reader, so it
is also a check on this file.

```markdown
---
id: spec-20261017-quality-analysis-display-contract
kind: spec
title: Quality analysis display contract
status: active
created: 2026-10-17
updated: 2026-10-18
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

The quality analysis screen shows each run's findings.

## Goals
```

A document this tool creates carries the same string as the title in its
frontmatter and as its body's first heading. That title is normalised —
surrounding whitespace removed and inner runs of whitespace collapsed to one
space — while the id and the file name are derived from the title as given, so
whitespace-only differences leave the id and file name alone; the divergence
this fixes in the heading is in `DIVERGENCES.md`. A document written by another
tool may have the two differ — the reference fixtures do — and the reader
reports each string exactly as it finds it.

## Document id

An id is `<prefix>-<YYYYMMDD>-<slug>`, where the prefix is `spec` or `decision`,
the date is the local creation date, and the slug is derived from the title as
given:

1. NFKD-normalise the title, then drop every non-ASCII character;
2. lowercase what is left;
3. replace every run of characters that is not `a`–`z` or `0`–`9` with a single
   `-`;
4. trim leading and trailing `-`;
5. if the result is longer than 72 characters, cut it to 72 and trim `-` again;
6. if that leaves nothing, use the first 10 hexadecimal characters of the
   SHA-1 of the UTF-8 title.

An id that collides with one already present gains a `-2`, `-3`, … suffix. The
slug algorithm is checked byte-for-byte against the Python implementation by a
committed corpus (`testdata/slug_corpus.json`).

## Document file name

A document lives at `<vault>/<specs|decisions>/<YYYY-MM-DD>-<slug>.md`, using the
local date at creation time. When that name is taken, the next free `-2`, `-3`, …
suffix is used and an existing file is never overwritten.

## Index record

`index.jsonl` holds one JSON object per line, one line per document, with eleven
fields in this order:

`id`, `kind`, `title`, `status`, `created`, `updated`, `repos`, `tags`,
`codex_sessions`, `path`, `summary`.

`path` is relative to the vault root. `summary` is the first body line that is
neither blank nor a heading, with surrounding whitespace removed; a line is a
heading when its first non-space character is `#`, indented or not. Documents are
ordered by directory — `specs` before `decisions` — and by file name within a
directory.

Each line is written the way the reference tool's
`json.dumps(record, ensure_ascii=False)` writes it: `": "` after a key, `", "`
between list items, the five control characters that have a JSON short escape
(`\b`, `\f`, `\n`, `\r`, `\t`) written as that escape, every other control
character escaped as `\u00xx`, and every other character — including non-ASCII —
emitted literally as UTF-8. `encoding/json` does not produce these bytes, so the
encoder is hand-rolled and checked against the reference fixtures. A record for
the sample document above looks like:

```json
{"id": "spec-20261017-quality-analysis-display-contract", "kind": "spec", "title": "Quality analysis display contract", "status": "active", "created": "2026-10-17", "updated": "2026-10-18", "repos": ["lippi-smart-customer", "aics-web-repos"], "tags": ["quality-analysis", "contract"], "codex_sessions": ["session-a", "session-b"], "path": "specs/2026-10-17-quality-analysis-display-contract.md", "summary": "The quality analysis screen shows each run's findings."}
```
