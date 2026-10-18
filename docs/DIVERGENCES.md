# Divergences from the reference tool

`specshelf` is a Go port of the Python reference implementation
[`agent-spec-vault`](https://github.com/xixifast/agent-spec-vault) v0.1.2, and the
reference stays the compatibility target for the vault format: the same
directory, the same document bytes where it is safe to match, and the same
document ids, so one vault is usable by both tools in both directions. The
format itself is [`FORMAT.md`](FORMAT.md) and the package boundaries are
[`ARCHITECTURE.md`](ARCHITECTURE.md); this file is only the record of the places
where `specshelf` deliberately does something else.

An entry is intentional. An unexplained difference is a bug: the differential
suite in `internal/conformance` runs both tools against equivalent vaults and
fails on a difference no entry here explains. Every entry below names the test
that pins its behaviour, in the package whose source it describes.

The reference implementation is judged from its source (`specv/vault.py` and
`specv/cli.py`) and from the committed fixtures in [`testdata/`](../testdata/),
which are its recorded output. "The reference" below means that tool; "we",
"ours" and `specshelf` mean this port.

## How the conformance suite encodes part of this list

The default suite compares the command surface against the committed fixtures
without needing Python; the `//go:build conformance` suite runs the reference
tool live. Neither may hide a difference: where a tolerance is applied it is a
narrow substitution that occurs an exact, asserted number of times
(`substituteOnce` in `internal/conformance/main_test.go`). Four entries below are
encoded that way, and are worded here exactly as the suite words them:

| Entry | Where the suite encodes it |
| --- | --- |
| "valid-YAML emission" | differential case `new reserved title` substitutes the reference's bare `title: #1 priority` for our quoted form |
| "automatic index refresh" | `diffCase.ignoreIndex` excludes `index.jsonl` from the file comparison, because we refresh it and the reference does only when asked |
| "empty list fields written `[]` where the reference writes a bare key" | `emptyListFields` rewrites the reference's bare `repos:`/`tags:`/`codex_sessions:` lines to `[]` |
| "empty scalar values reading as empty rather than as `[]`" | `tolerateEmptyScalarTitle` substitutes the reference's recorded empty-title field, asserted to occur exactly once |

## Writing a document

### valid-YAML emission

**Reference.** Its writer emits `key: value` with no quoting
(`render_frontmatter` in `specv/vault.py`). A title such as `#1 priority`,
`- dash start`, `*bold* start`, `Fix #42: the thing`, `true` or `123` is written
raw, so a standard YAML parser reads it as a comment, a sequence, an alias, a
mapping or a value that is not a string. The frontmatter is a private line
dialect, not YAML.

**specshelf.** The writer quotes any scalar whose text would otherwise change
meaning: an empty value, one with leading or trailing whitespace, one containing
`: ` or a trailing `:`, one containing ` #`, one starting with a YAML indicator
character (`-?:,[]{}#&*!|>'"%@\``), one a YAML reader would take for another
type (`true`, `null`, `~`, `42`, `1.5`, `.inf`, …), and anything holding a
character that cannot sit on the line (a break, a tab, a control character,
U+0085/U+2028/U+2029). The escapes the writer emits are `\\`, `\"`, `\n`, `\r`,
`\t` and `\uXXXX`. `created` and `updated` stay bare, so those two lines keep the
reference's exact bytes. The rules are in [`FORMAT.md`](FORMAT.md).

**Why.** Decision D2: the reader must accept the reference's looser dialect,
because that is what real vaults contain, but the writer must emit YAML a
third-party parser accepts. Weakening a quoting rule has to fail a test or the
rules are decoration.

**Test.** `TestEmittedFrontmatterIsValidYAML` and
`TestMarshalQuotesValuesThatWouldChangeMeaning` in
`internal/frontmatter/render_test.go`; the emitted bytes are checked with the
test-only `go.yaml.in/yaml/v3` dependency.

### atomic writes

**Reference.** `write_doc` calls `Path.write_text`, which truncates the document
and rewrites it in place. An interruption — a crash, a full disk, a killed
process — can leave a truncated document.

**specshelf.** `WriteDocument` writes the complete new bytes to a temporary file
in the destination directory, flushes it to durable storage (`Sync`) and renames
it over the target. A failure before the rename leaves the previous bytes, or no
file, and removes the temporary file. The index refresh is the same write, so it
is atomic too.

**Why.** Decision D6: data loss is the failure mode this port exists to fix. The
reference's non-atomic write is why the two tools cannot simply share a write
path.

**Test.** `TestAtomicWriteReplacesInOneStep`,
`TestAtomicWriteInterruptedRenameLeavesOriginal` and `TestWriteIndexIsAtomic` in
`internal/vault/write_test.go` and `internal/vault/refresh_test.go`.

### exclusive creation

**Reference.** `unique_path` checks whether a name exists and then the caller
writes it, so two creators can both see a name free and the second overwrites the
first.

**specshelf.** Creation uses an exclusive create
(`os.O_CREATE|os.O_EXCL`); a name taken between the check and the create wraps
`fs.ErrExist` and the store retries the next free `-2`, `-3`, … name. An
existing file is never truncated.

**Why.** Decision D6, and the vault-store spec's concurrent-creation scenario.

**Test.** `TestExclusiveCreateRetriesNextFreeName`,
`TestExclusiveCreateRefusesToTruncate` and
`TestConcurrentCreationOfTheSameTitle` in `internal/vault/write_test.go`.

### empty list fields written `[]` where the reference writes a bare key

**Reference.** `render_frontmatter` writes `repos:`, `tags:` and
`codex_sessions:` with nothing after the colon when the list is empty. To a YAML
reader that is null, not an empty sequence.

**specshelf.** An empty list field is written `key: []`, and a preserved unknown
key holding an empty list is written the same way. `[]` reads back as an empty
list in this reader, in the reference's reader and in a standard YAML parser
alike.

**Why.** [`FORMAT.md`](FORMAT.md) requires the nine fields always to be present
and an empty list to be unambiguous, and the conformance suite's
`emptyListFields` tolerance exists only for this difference.

**Test.** `TestMarshalFieldOrderAndShape` and `TestMarshalPreservesUnknownKeys`
in `internal/frontmatter/render_test.go`.

### preserved keys keep a fixed, name-sorted position

**Reference.** An unrecognised key survives a rewrite: `parse_frontmatter` keeps
every key it reads, and `render_frontmatter` emits the unknown keys
`sorted(metadata)` after the six scalars and three lists.

**specshelf.** The same visible order: the nine known fields in the documented
order, then every unrecognised key sorted by name. The writer is also the only
writer — the store calls it for every rewrite — so a key the reader understood
comes back unchanged.

**Why.** The position of a preserved key is part of the "a rewrite is
byte-stable" contract in [`FORMAT.md`](FORMAT.md); pinning it stops a writer
change from silently reordering a document. (This is the one item on the task's
list where the reference's bytes agree for every key its parser can read; the
entry records the contract and its test, and the record is honest about that.)

**Test.** `TestMarshalPreservesUnknownKeys` in
`internal/frontmatter/render_test.go`.

### normalised title whitespace in the body heading

**Reference.** `create_doc` seeds the body from the template with the **raw**
title (`template.format(title=title)`) but stores `clean_scalar(title)` — the
whitespace-collapsed form — in the frontmatter. A title with surrounding or
repeated whitespace therefore produces a frontmatter `title` and a `# ` heading
that differ.

**specshelf.** The body heading uses the same normalised title as the
frontmatter: surrounding whitespace removed and inner runs collapsed to one
space. The id and the file name are still derived from the title **as given**, so
a whitespace-only difference changes neither. A document written by another tool
may still have the two differ, and the reader reports each string as it finds it.

**Why.** [`FORMAT.md`](FORMAT.md) and the vault-format spec's title-whitespace
scenario: the frontmatter title and the heading are the same string.

**Test.** `TestNormaliseTitle` and the `TestSeedBody` case "whitespace title
normalised for heading and frontmatter, raw for id and name" in
`internal/vault/body_test.go`.

### empty scalar values reading as empty rather than as `[]`

**Reference.** `parse_frontmatter` stores `title:` as an **empty list** and
`title: []` as the **string** `"[]"`. A document with no title is therefore
indexed as `"title": "[]"`, and the literal `[]` is read back as a title and
persisted by a rewrite.

**specshelf.** An empty value and an empty flow sequence both read as the empty
scalar. A document with no title has an empty title, not the text `[]`, and a
rewrite does not invent one.

**Why.** The vault-format spec: "Empty values mean the empty scalar." A title
that decays into `[]` on the first rewrite is data loss.

**Test.** `TestEmptyValuesAreEmpty` in
`internal/frontmatter/document_test.go`, and `assertEmptyTitleDivergence` with
the fixture-parity tests in `internal/vault/index_test.go`.

## Reading a document

The reference reader starts a document only when its bytes begin exactly
`---\n`, closes the block on the first `\n---\n`, and splits the block with
Python's `str.splitlines`. Those three choices make it mangle or discard
documents that are otherwise fine. The entries in this section are the reader
tolerance the vault-format spec requires; `internal/frontmatter/parse.go` is the
implementation and its doc comment lists the rules.

### a byte-order mark

**Reference.** `read_doc` requires the text to start with `---\n`, so a document
that opens with a UTF-8 BOM is read as a document **without frontmatter**: all
metadata is lost and the body becomes the whole file.

**specshelf.** A leading U+FEFF is dropped before the fence is looked for, so the
frontmatter is read and the mark does not appear in the body.

**Test.** `TestParseNormalisesInput` case "byte-order mark" in
`internal/frontmatter/document_test.go`, and `TestSplitFrontmatter` case "a
byte-order mark is ignored" in `internal/frontmatter/split_test.go`.

### an empty frontmatter block

**Reference.** For `---\n---\nbody`, the closing-fence search from offset 4 finds
no `\n---\n`, so the document is reported unterminated and `iter_docs` silently
skips it.

**specshelf.** The closing fence is searched from one byte before the opening
fence ends, so an immediately-following fence is an empty block: empty metadata
and the body everything after the fence.

**Test.** `TestSplitFrontmatter` case "empty block" in
`internal/frontmatter/split_test.go`.

### quoted values unquoted, with the escapes the writer emits decoded

**Reference.** `parse_frontmatter` never removes quotes, so it reads our
`title: "a: b"` as the seven-character text `"a: b"`, quotes included, and would
persist that.

**specshelf.** A well-formed single- or double-quoted scalar is unquoted. Inside
a double-quoted value it also resolves the escapes our writer emits — `\\`,
`\"`, `\n`, `\r`, `\t` and a four-digit `\uXXXX`; single quotes double as
`''`. Any other escape, and any malformed quoting, is left exactly as written, so
a stray quote cannot make the reader reinterpret text it does not understand.

**Test.** `TestParseBlock` cases "double quoted value", "escaped quote inside a
double quoted value", "escaped backslash inside a double quoted value", "single
quoted value", "doubled quote inside a single quoted value", "a unicode escape
is decoded" and "malformed double quoting is left alone" in
`internal/frontmatter/parse_test.go`.

### quoted keys unquoted

**Reference.** `line.split(":", 1)` splits a quoted key at the first colon
inside it, so `"a: b": value` becomes the key `"a` and the value `b": value`.

**specshelf.** A key the writer emitted quoted is read to its closing quote, so a
key containing `: ` or beginning with `- ` survives a round trip.

**Test.** `TestParseBlock` cases "a quoted key is unquoted", "a quoted key
containing a separator" and "a quoted dash key is not a list item" in
`internal/frontmatter/parse_test.go`.

### list items at any indentation

**Reference.** An item is recognised only when the line starts with exactly two
spaces and `- `. The writer indents two spaces, but a hand-edited or
differently-written file indented with a tab, four spaces or none loses every
item while keeping the key.

**specshelf.** An item is a line whose first non-space, non-tab characters are
`- `, at any indentation.

**Test.** `TestParseBlock` cases "four space indentation still reads", "tab
indentation still reads" and "no indentation still reads" in
`internal/frontmatter/parse_test.go`.

### an intervening non-item line does not close a list

**Reference.** Any line that is not an item clears `current_list`, so a comment
or a stray line between two items ends the list and the later items are dropped.

**specshelf.** A line carrying no key separator does not close the open list; a
line with a separator and a value does. A blank line never closes it.

**Test.** `TestParseBlock` cases "a line without a separator does not close the
list", "blank line inside a sequence" and "a key with a value closes the list" in
`internal/frontmatter/parse_test.go`.

### empty list items ignored

**Reference.** An item with no value is stored as an empty string, so
`tags:\n  - ` yields `[""]` and pollutes the tag list with a blank.

**specshelf.** An item with no text at all is ignored rather than stored; an
explicitly empty quoted item (`- ""`) is a value the writer emitted and is kept.

**Test.** `TestParseBlock` cases "an item with no value is ignored" and "an
explicitly empty quoted item survives" in
`internal/frontmatter/parse_test.go`.

### values containing exotic line separators

**Reference.** `str.splitlines()` breaks on more than `\n`: a vertical tab, a
form feed, U+001C–U+001E, NEL (U+0085), U+2028 and U+2029 all end the line early,
so `title: a\u2028b` is read as the title `a` and the rest of the value is lost.

**specshelf.** The block is split on `\n` only (after line-ending
normalisation), so a value holding one of those characters is read whole. None of
them is written raw: the writer escapes them (see "valid-YAML emission").

**Test.** `TestParseBlock` cases "an exotic separator does not truncate a value"
and "a vertical tab does not truncate a value" in
`internal/frontmatter/parse_test.go`; the same characters in a body are handled
by `TestSummarySplitsOnPythonLineBreaks` in `internal/vault/index_test.go`.

### line endings (CRLF and lone CR) normalised where the reference discards the document

**Reference.** A document saved with CRLF begins `---\r\n`, not `---\n`, so the
reference reads it as a document without frontmatter and every field is lost. A
lone-CR file is the same.

**specshelf.** CRLF and lone CR are normalised to LF before the fence is looked
for, and the writer emits LF only, so a document written back from a CRLF file
uses LF endings.

**Test.** `TestSplitFrontmatter` cases "windows line endings" and "lone carriage
returns" in `internal/frontmatter/split_test.go`, and
`TestRewriteCRLFDocumentUsesLF` in `internal/frontmatter/lineendings_test.go`.

### a `kind` derived from the directory where the reference assumes `spec`

**Reference.** `VaultDoc.kind` defaults to `"spec"` whenever the frontmatter has
no `kind` (`self.metadata.get("kind", "spec")`), so a document in `decisions/`
without the field is reported as a spec.

**specshelf.** A document with no `kind` derives it from its directory:
`specs/` is a spec and `decisions/` is a decision, and any other directory is a
spec. `kind` read from the frontmatter is unchanged.

**Test.** `TestParseWithoutFrontmatter` in
`internal/frontmatter/document_test.go`.

## The command surface

### stderr warnings instead of silently skipped documents

**Reference.** `iter_docs` catches the parse error and continues
(`except ValueError: continue`), so an unreadable document disappears from every
listing with no signal.

**specshelf.** A listing command skips the unreadable document, prints **one**
warning line to stderr naming the vault-relative path and the reason, and
continues with the remaining documents and a successful exit code. The strict
`Index` path still fails and names the file, which `doctor` builds on.

**Why.** The proposal lists this under corrections where upstream is wrong; a
document that vanishes silently is the opposite of the durability the vault is
for.

**Test.** `TestSkipUnreadableDocumentWarns`,
`TestUnreadableReasonExtraction` and `TestSkipWarnsOncePerUnreadableDocument` in
`internal/vault/list_test.go`.

### automatic index refresh

**Reference.** `index.jsonl` is written only by the `index` command; a document
created, edited or removed without running `specshelf index` leaves the index
stale.

**specshelf.** Every command that creates, modifies or removes a document
regenerates `index.jsonl` before it exits, atomically, so the index always matches
the files on disk. `index` still regenerates on demand and `--print` reports its
path.

**Why.** The proposal lists it under corrections: an index that trails the vault
is a cache with no invalidation. The conformance suite's `ignoreIndex` knob
carves this out of the file comparison.

**Test.** `TestWriteIndexFollowsMutation` in
`internal/vault/refresh_test.go` and `TestIndexConsistentAfterSessionDelete` in
`internal/tui/mutations_test.go`.

### new commands

**Reference.** Its command set is `init`, `new`, `decision`, `link-session`,
`list`, `search`, `show`, `index`, `prime` and `path`.

**specshelf.** All ten exist with the reference's flag names, aliases, defaults
and output shapes. Nine commands are added: `edit`, `rm`, `status`, `doctor`,
`export`, `stats`, `sync`, `upgrade` and `ui`. Superset **options** ride on
existing commands too: `--json` on every read command,
`--format tsv|table|csv|jsonl` on `list` and `search`, `--color`, and `--slug`
on `new` and `decision`. A hand-picked `--slug` is used verbatim, and one that
would escape the document directory — empty, containing a path separator, or
exactly `"."` or `".."` — is refused with a usage error before the vault is
created.

**Why.** Decisions D10 (upgrade), D11 (TUI) and D3 (`--slug` gives humans a
readable id); the superset is stated in the proposal and detailed in
`specs/cli/spec.md`.

**Test.** `TestSupersetCommandsAreAvailable` and
`TestCompletionMentionsEveryCommand` in `internal/cli/generate_test.go`;
`TestCreateUsesPickedSlugVerbatim`, `TestCreateRejectsInvalidPickedSlug` in
`internal/vault/command_test.go`, and `TestInvalidSlugIsUsageError` in
`internal/cli/exitcodes_test.go`.

### exit codes

**Reference.** `argparse` exits `2` on a bad command line, and anything else is
an uncaught Python exception: exit `1` with a traceback on stderr.

**specshelf.** Three codes are defined and pinned: `0` on success, `1` on a
runtime failure, `2` on a usage failure. Every failure prints exactly one line to
stderr prefixed `specshelf: `; no stack trace is ever printed. The classifier
maps the typed errors — `ConflictError`, `ProjectDisabledError`,
`UnknownVaultError` and `InvalidSlugError` to usage, `NoProjectVaultError`,
`UnknownReferenceError`, `AmbiguousReferenceError` and `OutsideVaultError` to
runtime — and recognises cobra's own flag, argument and unknown-command errors by
message.

**Why.** Data on stdout is a machine contract (`specs/cli/spec.md`); a traceback
on stdout or a nondeterministic code is not.

**Test.** `TestRunExitCodes` and `TestRunClassifiesTypedErrors` in
`internal/cli/exitcodes_test.go`; the recorded exit codes for the ten reference
invocations are in `testdata/python_vault_expected/exit_codes.json` and compared
by `internal/conformance`.

### colour rules

**Reference.** The reference has no colour at all.

**specshelf.** `--color auto|always|never`, default `auto`, which enables colour
only when stdout is a character device, `NO_COLOR` is unset and `TERM` is not
`dumb`. Colour never reaches `--json` output, any `--format` output other than
`table`, or the single-value stdout of `new`, `decision`, `link-session`, `path`
and `index --print`.

**Why.** Decision D5: colour from the standard library, gated so machine output
cannot be corrupted.

**Test.** `TestPaintEnabled`, `TestMachineFormatsNeverCarryEscapes`,
`TestColorFromConfig` and `TestTableFormatColourGating` in
`internal/cli/color_test.go` and `internal/cli/format_test.go`.

### index paths separated by `/` on every platform

**Reference.** `index_record` writes `str(doc.path.relative_to(home))`, which on
Windows uses `\`.

**specshelf.** An index path is vault-relative and slash-separated on every
platform, so the index bytes are identical across Linux, macOS and Windows and
match the reference's Unix-recorded fixtures. The path is also accepted back with
either separator: the read paths join a record path under the root with
`filepath.Join`, which accepts `/` on Windows as well.

**Test.** `TestIndexPathUsesForwardSlashes` in `internal/vault/index_test.go`,
and the byte-for-byte fixture comparison `TestIndexReferenceVaultParity`. On
Unix the platform join and a literal `/` produce the same bytes, so the assertion
distinguishes the two only on Windows; the parity tests are what bite there.

### an empty `list` printing nothing

**Reference.** `print_list` prints `No specs or decisions found.` on **stdout**
when there are no documents, so a caller reading one row per line sees a row.

**specshelf.** An empty listing prints nothing on stdout. The `tsv` and `jsonl`
list formats share this; `csv` and `table` are formats the reference does not
have, and print their header row as usual.

**Why.** `specs/cli/spec.md`: stdout carries the payload alone, and every tsv
line has the same nine columns. Prose belongs on stderr — and the reference's
sentence is prose.

**Test.** `TestEmptyListPrintsNothing` in `internal/cli/format_test.go`.

### a document reference outside the vault is refused

**Reference.** `find_doc` returns any existing path the user names, so
`link-session <path>` rewrites a file outside the vault. The reference's deletion
and edit paths are the same shape.

**specshelf.** Reference resolution still tries an existing path first, so that
part is compatible, but `rm`, `edit`, `status` and `link-session` refuse a
resolved document whose path is not inside the resolved vault root, with an
`OutsideVaultError`, and leave the file byte-identical. This is the same
confinement 4.12 applies to a hand-picked `--slug`.

**Why.** Data loss: a reference that resolves to an arbitrary path would let a
stray argument delete or overwrite a file the vault does not own. `rm`, `edit`,
`status` and `link-session` are the four mutation paths; the read commands do not
need it.

**Test.** `TestRemoveConfinement`, `TestEditRefusesOutsideVault`,
`TestSetStatusRefusesOutsideVault` and `TestLinkSessionsRefusesOutsideVault` in
`internal/vault/command_test.go`.
