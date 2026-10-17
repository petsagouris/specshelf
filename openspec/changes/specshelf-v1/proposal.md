# Proposal

## Why

Coding agents need durable, cross-repository context — specs, contracts, design
decisions — that survives chat resets. `agent-spec-vault` keeps exactly that in
plain Markdown, but it is a 600-line Python package: it needs a Python
toolchain, ships no binaries, silently drops documents whose frontmatter it
cannot parse, writes frontmatter that is not valid YAML, and has no write
atomicity, so two agents writing at once can lose data. `specshelf` is a Go port
and superset: one static binary, installable with `go install` on Linux, macOS
and Windows, reading and writing the same vault.

## What Changes

- New Go implementation, `github.com/petsagouris/specshelf`, binary `specshelf`,
  module requiring Go 1.26 or newer.
- **Format compatibility is preserved**: same vault directory, same document
  bytes, same document ids. A vault written by either tool is fully usable by
  the other. Verified against the ported slug algorithm on a 12-title corpus,
  including CJK, Cyrillic and emoji titles.
- Corrections where upstream is wrong: frontmatter is always written as valid
  YAML; every write is atomic (temp file plus rename) and creation is exclusive;
  unreadable documents produce a warning on stderr instead of disappearing from
  listing; `index.jsonl` is refreshed after every mutating command.
- Superset command surface: `edit`, `rm`, `status`, `doctor`, `export`, `stats`,
  `sync`, `upgrade`, `ui`, plus `--json` on every read command,
  `--format tsv|table|csv|jsonl`, `--color`, shell completions and man pages.
- New capabilities not present upstream: a TOML config file with named vaults,
  project-local `.specshelf/` vault discovery, git-backed syncing through the
  user's own `git`, a `go install`-aware upgrade check, and a full-screen
  browser.
- **BREAKING** (against the Python CLI, for existing scripts and agent
  instructions): there is no `specv` binary alias, and `SPECV_HOME` is not read;
  the equivalent is `SPECSHELF_HOME`.
- Out of scope: an MCP server, any daemon, watcher or network listener, release
  artifacts and code signing, merged search across multiple vaults at once, and
  Windows-specific vault relocation.

## Capabilities

### New Capabilities

- `vault-format`: the on-disk contract — vault layout, document frontmatter and
  body, document identity and slugs, file naming, and the JSONL index.
- `vault-store`: choosing which vault is in use and applying every document
  operation safely — resolution and project discovery, create/read/update/remove,
  atomic writes, handling of unreadable documents, index refresh.
- `cli`: the command surface and its machine-readable contract — commands,
  flags, output formats, colour, exit codes, filtering and sorting.
- `config`: user preferences and named vaults from a TOML file, including
  precedence against flags and environment variables.
- `git-sync`: committing and optionally pushing a vault with the user's own git
  installation.
- `upgrade`: reporting and applying updates under a `go install`-only
  distribution.
- `tui`: interactive browsing, searching, viewing and editing that reuses the
  CLI's storage guarantees.
- `conformance`: the compatibility contract with the Python implementation —
  fixtures, bidirectional interchange and the enumerated list of intentional
  divergences.

### Modified Capabilities

None — this is the first change in the project; no specs exist yet.

## Impact

- New Go module with no existing code to migrate; the reference implementation
  stays untouched and remains the source of truth for format behaviour.
- Runtime dependencies: `spf13/cobra`, `golang.org/x/text`,
  `pelletier/go-toml/v2`, `charm.land/bubbletea/v2` (+ `charm.land/bubbles/v2`).
- Reads and writes user-owned vaults under `~/.agents/spec-vault`, including
  vaults that already contain documents created by the Python tool. Data loss is
  the top risk of this change; that is why atomic writes and the conformance
  suite are in scope rather than deferred.
- Derived work: MIT `agent-spec-vault` by xixifast, copyright notice retained.
- Distribution: `go install github.com/petsagouris/specshelf/cmd/specshelf@version`
  only, so the project carries no release tooling and no signing infrastructure.
