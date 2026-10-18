# vault-store Specification

## Purpose
Establishes which vault a command operates on and makes every document
operation safe: resolving a vault from flags, environment and project markers,
creating and mutating documents without losing data, and keeping the index
consistent with the files on disk.

## Requirements

### Requirement: Vault resolution

A command SHALL resolve exactly one vault, using the first of: the `--home`
flag, the `SPECSHELF_HOME` environment variable, the `--vault <name>` selector
resolved through the config file, the project vault found by walking up from the
working directory, the config file's default vault, and finally
`~/.agents/spec-vault`. The resolution source SHALL be reportable, and
`specshelf path` SHALL print the resolved vault root to stdout.

#### Scenario: Default vault

- **WHEN** no flag, environment variable or config entry selects a vault
- **THEN** the resolved vault is `~/.agents/spec-vault`

#### Scenario: Environment beats config

- **WHEN** `SPECSHELF_HOME` is set and the config file also names a default vault
- **THEN** the environment variable wins

#### Scenario: Resolution source is discoverable

- **WHEN** `specshelf path --json` runs
- **THEN** the output names the vault root and which selector produced it

### Requirement: Conflicting vault selectors are rejected

Combining two explicit selectors — `--home` with `--vault`, or either with
`--project` — SHALL fail as a usage error with a non-zero exit code and a
message naming both conflicting options, rather than silently preferring one.

#### Scenario: Two explicit selectors

- **WHEN** a command is invoked with both `--home /tmp/a` and `--vault work`
- **THEN** the command prints a usage error to stderr, does not touch either vault, and exits with the usage exit code

### Requirement: Project vault discovery

With `--project`, the tool SHALL search from the working directory upward for a
directory named `.specshelf` and use it as the vault root. The search SHALL stop
at the user's home directory or the filesystem root, whichever comes first. When
no project vault exists the command SHALL fail with an error rather than falling
back to another vault.

#### Scenario: Nested working directory

- **WHEN** `.specshelf` exists at `~/work/project` and a command with `--project` runs in `~/work/project/internal/vault`
- **THEN** the vault in use is `~/work/project/.specshelf`

#### Scenario: No project vault

- **WHEN** `--project` is used where no ancestor holds `.specshelf`
- **THEN** the command fails with an error naming the searched starting directory and exits non-zero

#### Scenario: The search does not escape the home directory

- **WHEN** `.specshelf` exists only above the user's home directory
- **THEN** the search does not find it and reports that no project vault exists

### Requirement: Atomic document writes

Every write that modifies or creates a document SHALL write the complete new
content to a temporary file in the destination directory, flush it to durable
storage, and rename it over the target. A failure or interruption at any point
SHALL leave either the previous content or the complete new content, never a
truncated or partially written file.

#### Scenario: Interrupted write leaves the original intact

- **WHEN** writing a document fails between creating the temporary file and renaming it
- **THEN** the target file still contains its previous bytes and no temporary file is left behind

#### Scenario: Existing file is replaced in one step

- **WHEN** a document is modified by any command
- **THEN** after the command returns, the file content is the complete new content

### Requirement: Exclusive document creation

Creating a document SHALL use an exclusive create so that an existing file is
never truncated. When a concurrent create has already taken the intended name,
the tool SHALL retry with the next free `-2`, `-3` suffixed name and report the
path it actually created.

#### Scenario: Concurrent creation of the same title

- **WHEN** two `specshelf new` processes create the same title at the same time
- **THEN** both succeed, each printing its own path, and the vault contains two distinct documents

### Requirement: Unreadable documents are reported, not hidden

When a command reads the vault and a document cannot be parsed, the command
SHALL skip that document, write exactly one warning line to stderr naming the
path and the reason, and continue with the remaining documents and a successful
exit code. Collection commands SHALL include the failing file in their report
and exit non-zero.

#### Scenario: Unterminated frontmatter during listing

- **WHEN** `specshelf list` runs on a vault containing a file with an unterminated frontmatter block
- **THEN** the readable documents are listed on stdout, one warning naming the bad file appears on stderr, and the exit code is 0

#### Scenario: The same file under doctor

- **WHEN** `specshelf doctor` runs on that vault
- **THEN** the report identifies the file as unreadable and the exit code is non-zero

### Requirement: Index refresh on mutation

Every command that creates, modifies or removes a document SHALL regenerate
`index.jsonl` before it exits, and the regeneration SHALL itself be atomic.
`specshelf index` SHALL regenerate the index on demand and `--print` SHALL
additionally print its path.

#### Scenario: Index follows a write

- **WHEN** a document is created without running `specshelf index`
- **THEN** `index.jsonl` already contains the new document

#### Scenario: Index follows a removal

- **WHEN** a document is removed
- **THEN** its line is gone from `index.jsonl` before the command exits

### Requirement: Concurrent writers do not corrupt the vault

The tool SHALL NOT lock the vault. Concurrent commands against the same vault
SHALL each complete with a well-formed vault: no partially written document, no
corrupted index, and at most one writer's version of a simultaneously edited
document surviving.

#### Scenario: Last write wins on the same document

- **WHEN** two processes modify the same document at the same time
- **THEN** the resulting file parses correctly and contains the changes of one of the two writers in full

### Requirement: Document reference resolution

A reference SHALL resolve, in order of precedence, to an existing path, then to
a document id, then to a file name, then to a file stem. An ambiguous or unknown
reference SHALL fail with an error that names the reference, and an unknown
reference SHALL list nothing rather than guess.

#### Scenario: Reference by id, stem and path

- **WHEN** the same document is referenced by its id, by its file stem, and by its full path
- **THEN** all three resolve to that same document

#### Scenario: Unknown reference

- **WHEN** a reference matching no document is used with `show`
- **THEN** the command prints an error naming the reference to stderr and exits non-zero
