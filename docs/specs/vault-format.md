# vault-format Specification

## Purpose
Defines the on-disk contract of a vault: directory layout, document frontmatter
and body, document identity, file naming and the JSONL index. The contract is
what keeps a vault readable across versions: the reader tolerates what the writer
does not emit, and the writer only ever emits what the reader can read back.

## Requirements

### Requirement: Vault directory layout

A vault root SHALL contain the directories `specs/` and `decisions/`, the
directory `templates/` holding `spec.md` and `decision.md`, and the file
`README.md`. The index file `index.jsonl` SHALL be produced by the index step
rather than by initialisation. Initialising a vault SHALL create anything
missing and SHALL NOT overwrite a file that already exists, so user edits to
templates and README survive.

#### Scenario: Initialising an empty directory

- **WHEN** `specshelf init` runs against an empty or non-existent directory
- **THEN** `specs/`, `decisions/`, `templates/spec.md`, `templates/decision.md` and `README.md` exist, and the vault path is printed to stdout

#### Scenario: Initialising a vault that already has a customised template

- **WHEN** `specshelf init` runs on a vault whose `templates/spec.md` was edited by the user
- **THEN** the file keeps its edited content

### Requirement: Document frontmatter fields

Every document SHALL carry a frontmatter block delimited by `---` lines
containing at minimum `id`, `kind`, `title`, `status`, `created`, `updated`,
`repos`, `tags` and `codex_sessions`. `kind` SHALL be `spec` or `decision` and
SHALL match the directory the document lives in. List fields SHALL be rendered
as block sequences, and an empty list SHALL be written as `[]`: a bare `key:` is
null to a YAML reader and would not read back as an empty list.

#### Scenario: A created spec carries the full field set

- **WHEN** a spec is created with repos, tags and session ids
- **THEN** its frontmatter contains all nine fields, with `kind: spec`, and the list fields rendered as indented `- ` items

#### Scenario: Unknown keys are preserved

- **WHEN** a document whose frontmatter contains an unrecognised key is rewritten by any command
- **THEN** the unrecognised key and its value still appear in the resulting frontmatter

### Requirement: Tolerant frontmatter reading

The reader SHALL accept the frontmatter dialect the writer emits, and the looser
shapes a document in the wild may carry: unquoted scalars, quoted scalars, `key:`
lines that introduce empty list values, `  - ` list items, blank lines, and
scalar values containing `:` or `#`. A document with no frontmatter SHALL be
readable with empty metadata rather than rejected. An unterminated frontmatter
block SHALL be reported as an error naming the file.

#### Scenario: Reading a title containing a colon and a hash

- **WHEN** a document carries `title: Decision: use #1 approach`
- **THEN** the parsed title is exactly `Decision: use #1 approach`

#### Scenario: Empty values mean the empty scalar

- **WHEN** a document contains `title:` with nothing after the colon, or `title: []`
- **THEN** the title reads as an empty string, not as a list and not as the text `[]`

#### Scenario: Reading a document without frontmatter

- **WHEN** a Markdown file with no frontmatter is read
- **THEN** it is returned with empty metadata, its path as `id` and `title`, and `kind` derived from its directory

#### Scenario: Unterminated frontmatter

- **WHEN** a file begins with `---` and has no closing `---` line, or that line is not newline-terminated
- **THEN** reading it fails with an error that names the file path

### Requirement: Reader tolerance for documents the writer did not emit

A document that the writer did not emit exactly — written by another editor,
another tool, or an older version — SHALL read without losing data: line endings
and a byte-order mark are normalised, quoted values are unquoted, an empty
frontmatter block is readable, list items are recognised at any indentation, an
intervening line does not close an open list, list items with no value are
ignored, and values are not truncated at characters some parsers treat as line
breaks.

#### Scenario: CRLF line endings

- **WHEN** a document saved with Windows line endings opens with a frontmatter fence
- **THEN** its metadata is read normally, and a document written back later uses LF endings

#### Scenario: Byte-order mark

- **WHEN** a document opens with a byte-order mark followed by the fence
- **THEN** the frontmatter is read and the mark does not appear in the body

#### Scenario: Quoted values are unquoted

- **WHEN** a field carries a single- or double-quoted value, as the writer emits for values that need quoting
- **THEN** the value is read without its surrounding quotes

#### Scenario: Empty frontmatter block

- **WHEN** the closing fence immediately follows the opening fence
- **THEN** the document has empty metadata and its body is everything after the fence, rather than being unreadable

#### Scenario: List items at any indentation

- **WHEN** list items are indented with more than two spaces, with a tab, or with two spaces
- **THEN** every item is read

#### Scenario: A line between list items does not close the list

- **WHEN** a line carrying no key separator sits between two list items
- **THEN** the later item still belongs to the open list

#### Scenario: List items with no value are ignored

- **WHEN** an item is written as a dash with no value
- **THEN** it is ignored rather than stored as an empty string

#### Scenario: Exotic line separators do not truncate a value

- **WHEN** a value contains a character some parsers treat as a line break, such as a vertical tab or U+2028
- **THEN** the value is read whole rather than truncated at that character

### Requirement: Valid YAML frontmatter writing

Frontmatter written by specshelf SHALL be parseable by a standard YAML parser.
Scalar values that would otherwise change meaning SHALL be quoted, including
values containing `: `, a leading or trailing space, a leading `#`, or an empty
value. Field order SHALL be the scalars `id`, `kind`, `title`, `status`,
`created`, `updated`, then the lists `repos`, `tags`, `codex_sessions`, then any
preserved unknown keys sorted by name.

#### Scenario: Titles that need quoting

- **WHEN** a document titled `Fix #42: the thing` is written
- **THEN** the emitted frontmatter parses with a standard YAML parser and round-trips the exact title

#### Scenario: Dates are written bare

- **WHEN** a document is written
- **THEN** `created` and `updated` are emitted unquoted, so the byte form of a document does not change between writes, even though a YAML reader types them as dates rather than strings

#### Scenario: Empty values are written unambiguously

- **WHEN** a document whose title is empty is written
- **THEN** the field is emitted as `title: ""` and reads back as an empty title, both here and in a standard YAML parser

#### Scenario: Field ordering is stable

- **WHEN** any document is rewritten
- **THEN** its frontmatter field order is the documented order, and rewriting twice produces byte-identical output

### Requirement: Document identity

A document id SHALL be `<kind-prefix>-<YYYYMMDD>-<slug>`, where the prefix is
`spec` or `decision` and the slug is derived from the title. The slug algorithm
SHALL be: NFKD normalise, drop non-ASCII characters, lowercase, replace runs of
non-alphanumeric ASCII with `-`, trim `-`, truncate to 72 characters and trim
again. If that yields an empty slug, the id segment SHALL be the first 10 hex
characters of the SHA-1 of the UTF-8 title. Colliding ids SHALL gain a `-2`,
`-3` suffix.

#### Scenario: ASCII title

- **WHEN** a spec titled `Quality analysis display contract` is created
- **THEN** its id begins `spec-` and ends with `-quality-analysis-display-contract`

#### Scenario: Non-ASCII title

- **WHEN** a spec titled `质量分析展示契约` is created
- **THEN** its id segment is the first 10 hexadecimal characters of the SHA-1 of the UTF-8 title

#### Scenario: Duplicate titles

- **WHEN** two documents with the same title are created on the same day
- **THEN** the second id ends with `-2` and the two documents are distinct files

### Requirement: Document file naming

A document SHALL live at `<vault>/<specs|decisions>/<YYYY-MM-DD>-<slug>.md` in
the local date at creation time. When that path is taken by a different
document, the next free `-2`, `-3` suffixed name SHALL be used, and an existing
file SHALL never be overwritten.

#### Scenario: Date prefix uses the creation date

- **WHEN** a decision is created
- **THEN** its file name starts with the local date in `YYYY-MM-DD` form and ends with `.md`

#### Scenario: Same-day, same-title collision

- **WHEN** a second document would produce the same file name
- **THEN** the file name gains a `-2` suffix and the first file is left untouched

### Requirement: Document body and templates

New documents SHALL be seeded from the vault's template for their kind — specs
from `templates/spec.md`, decisions from `templates/decision.md` — with the
title substituted into the template's first heading. When a template file is
missing it SHALL be recreated from the built-in default before use.

#### Scenario: New spec body

- **WHEN** a spec is created
- **THEN** its body starts with `# <title>` and contains the template's section headings

#### Scenario: Missing template

- **WHEN** a document is created in a vault whose `templates/spec.md` was deleted
- **THEN** the default template is written back and the new document uses it

#### Scenario: Title whitespace is normalised consistently

- **WHEN** a document is created from a title carrying surrounding or repeated whitespace
- **THEN** the frontmatter title and the body heading both carry the same normalised title, with surrounding whitespace removed and inner runs collapsed to one space, while the id and file name are the same as they would be for that title unnormalised

### Requirement: Index file content

`index.jsonl` SHALL contain one JSON object per line, one line per document,
with the fields `id`, `kind`, `title`, `status`, `created`, `updated`, `repos`,
`tags`, `codex_sessions`, `path` and `summary`, where `path` is relative to the
vault root and `summary` is the first non-empty, non-heading line of the body.
Documents SHALL be ordered by directory (`specs` before `decisions`) and by file
name within a directory, and non-ASCII characters SHALL be emitted literally
rather than escaped.

#### Scenario: Index lines for a mixed vault

- **WHEN** a vault holds one spec and one decision and `specshelf index` runs
- **THEN** the file has two lines, the spec first, each a complete JSON object with all eleven fields

#### Scenario: Non-ASCII content stays literal

- **WHEN** a document's title contains non-ASCII characters
- **THEN** the index line contains those characters as UTF-8 rather than `\u` escapes
