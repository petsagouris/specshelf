# cli Specification

## Purpose
Defines the command surface and its machine-readable contract: which commands
exist, how their flags behave, what is written to stdout and stderr, and what
exit codes callers and agents can rely on.

## Requirements

### Requirement: Reference command parity

The tool SHALL provide `init`, `new`, `decision`, `link-session`, `list`,
`search`, `show`, `index`, `prime` and `path`, accepting the flag names,
aliases, defaults and output shapes of the Python reference implementation, so
that existing vaults, scripts and agent instructions keep working.

#### Scenario: Creating a document with the reference flags

- **WHEN** `specshelf new "Contract" --repos a,b --tags x --codex-session s1` runs
- **THEN** the document is created with those repos, tags and session ids and its absolute path is printed to stdout

#### Scenario: Reference filter flags on listing

- **WHEN** `specshelf list --kind decision --tag tooling` runs
- **THEN** only decisions carrying that tag are printed

### Requirement: Superset commands

In addition to the reference commands, the tool SHALL provide `edit`, `rm`,
`status`, `doctor`, `export`, `stats`, `sync`, `upgrade` and `ui`, each
documented in `--help` and each listed by the root help output.

#### Scenario: Root help lists every command

- **WHEN** `specshelf --help` runs
- **THEN** all nineteen commands appear

### Requirement: Multi-value flag parsing

Flags that accept multiple values — `--repo`/`--repos`, `--tag`/`--tags`,
`--codex-session` — SHALL accept both repeated flags and comma-separated values,
SHALL trim surrounding whitespace, SHALL drop empty parts, and SHALL de-duplicate
while preserving first-seen order.
The current-session flag SHALL read `CODEX_THREAD_ID`, falling back to
`CODEX_SESSION_ID`, and SHALL fail with an error when neither is set.

#### Scenario: Mixed repetition and commas

- **WHEN** `--tags a,b --tags b --tags ' c '` is passed
- **THEN** the resulting tags are `a`, `b`, `c` in that order

#### Scenario: Current session unavailable

- **WHEN** `--current-codex-session` is used with neither environment variable set
- **THEN** the command fails with an error naming the missing variable and exits non-zero

### Requirement: stdout carries data only

Every command SHALL write only its payload to stdout: `list` the tab-separated
columns, `search` the tab-separated match rows, `show` the raw document, `new`
and `decision` and `link-session` the absolute path, `path` the vault root,
`index --print` the index path. Human-oriented prose, warnings and errors SHALL
go to stderr, so stdout can be parsed without filtering.

#### Scenario: Listing is parseable

- **WHEN** `specshelf list` output is piped to another process
- **THEN** every stdout line has the same nine tab-separated columns and contains no escape sequences

#### Scenario: Warnings do not pollute stdout

- **WHEN** a command skips an unreadable document
- **THEN** the warning appears on stderr and stdout contains only the payload

### Requirement: Structured output

Read commands SHALL support `--json`, emitting one JSON object per line for
collections and a single JSON object for single-document commands, using the
same field names as the index. `list` and `search` SHALL support
`--format tsv|table|csv|jsonl` with `tsv` as the default.

#### Scenario: JSON list

- **WHEN** `specshelf list --json` runs
- **THEN** each line parses as a JSON object carrying at least the index's field set

#### Scenario: Switching format

- **WHEN** `specshelf list --format csv` runs
- **THEN** the output is CSV with a header row and the same field order as the TSV output

### Requirement: Colour never breaks machine output

Colour SHALL be controlled by `--color auto|always|never`, defaulting to `auto`,
which enables colour only when stdout is a terminal, `NO_COLOR` is unset and
`TERM` is not `dumb`. Colour SHALL be absent from `--json` output, from
`--format` output other than `table`, and from the single-value stdout of `new`,
`decision`, `link-session`, `path` and `index --print`.

#### Scenario: Piped output has no escapes

- **WHEN** any display command's stdout is redirected to a file with default settings
- **THEN** the file contains no ANSI escape sequences

#### Scenario: Colour can be forced and disabled

- **WHEN** `--color always` and `--color never` are used with the same command
- **THEN** the first emits escape sequences and the second does not, regardless of terminal state

### Requirement: Exit codes

Commands SHALL exit `0` on success, `1` on a runtime error with a message on
stderr prefixed `specshelf: `, and `2` on a usage error such as an unknown flag,
an unknown command, or conflicting selectors.

#### Scenario: Runtime failure

- **WHEN** `specshelf show` is given an unknown reference
- **THEN** the exit code is 1 and stderr carries one message starting with `specshelf: `

#### Scenario: Usage failure

- **WHEN** an unknown flag is passed
- **THEN** the exit code is 2

### Requirement: Filtering and ordering

`list`, `search` and `prime` SHALL accept `--kind`, `--repo`, `--tag`,
`--codex-session` and `--status` filters, ANDed together, and SHALL order
matches by `updated` descending, then by id descending. `list` SHALL default to
50 results and `prime` to 12, both overridable with `--limit`.

#### Scenario: Combined filters

- **WHEN** `--kind spec --status active` is used
- **THEN** only active specs are listed

#### Scenario: Default ordering

- **WHEN** several documents share an `updated` date
- **THEN** they are ordered by id descending within that date

### Requirement: Agent prime output

`prime` SHALL print a Markdown summary of the matching documents, headed by the
vault path, with each entry showing id, kind, status, title, relative path,
repos, tags and linked session ids, filtered by the same flags as `list` and
available as `--json` for machine consumers.

#### Scenario: Prime for one repository

- **WHEN** `specshelf prime --repo billing` runs
- **THEN** the output starts with a heading naming the vault, and contains only documents tagged with that repo

### Requirement: Editing, removal and status changes

`edit` SHALL open the document in the editor named by `$VISUAL`, then
`$EDITOR`, then the configured editor, and SHALL update `updated` only when the
file content actually changed. `rm` SHALL delete a document and SHALL support
`--dry-run`, which reports the path that would be removed without removing it.
`status <ref> <status>` SHALL set the frontmatter status and update `updated`.
All three SHALL refresh the index.

#### Scenario: Unchanged edit

- **WHEN** `edit` opens a document and the editor exits without modifying it
- **THEN** the file's bytes and its `updated` field are unchanged

#### Scenario: Dry-run removal

- **WHEN** `rm <ref> --dry-run` runs
- **THEN** the document still exists and its path is reported

#### Scenario: Status change

- **WHEN** `status <ref> accepted` runs on an active spec
- **THEN** the frontmatter `status` is `accepted` and `updated` is today's date

### Requirement: Vault diagnostics and export

`doctor` SHALL verify the vault layout, report every document that fails to
parse, report documents whose `kind` disagrees with their directory, report
documents whose id or file name do not match their title, rebuild the index when
asked, and exit non-zero when it reports at least one problem.
`export` SHALL write every document to stdout as JSON or JSONL. `stats` SHALL
report counts grouped by kind, status, repo and tag.

#### Scenario: Healthy vault

- **WHEN** `doctor` runs on a vault created and maintained by specshelf
- **THEN** it reports no problems and exits 0

#### Scenario: Kind mismatch is reported

- **WHEN** a document with `kind: decision` lives in `specs/`
- **THEN** `doctor` reports the mismatch and exits non-zero

### Requirement: Shell completions and manual pages

The tool SHALL generate shell completion scripts for bash, zsh, fish and
PowerShell, and man pages for the root command and every subcommand, from the
command definitions themselves, with no hand-maintained copies.

#### Scenario: Completion script is generated

- **WHEN** `specshelf completion bash` runs
- **THEN** the output is a valid bash completion script mentioning every command

#### Scenario: Man page is generated

- **WHEN** `specshelf man` runs
- **THEN** roff man pages covering the root command and its subcommands are produced

### Requirement: Version reporting

`specshelf --version` SHALL report the module version recorded in the binary at
build time, so that a binary installed with `go install ...@version` reports
that version, and SHALL fall back to a development marker when no version is
recorded.

#### Scenario: Installed binary reports its version

- **WHEN** a binary built by `go install` at tag `v0.1.0` runs `--version`
- **THEN** the output contains `v0.1.0`

#### Scenario: Local build

- **WHEN** a binary built from an untagged working copy runs `--version`
- **THEN** the output identifies itself as a development build rather than an empty or wrong version
