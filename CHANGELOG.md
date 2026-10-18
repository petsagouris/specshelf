# Changelog

All notable changes to `specshelf` are documented in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.1] - 2026-10-18

The command surface and the vault format are unchanged from `0.1.0`: this release
exists because the module is what `go install` downloads, and it no longer
carries the spec-development tree or the reference fixtures. The distribution
limitations under `0.1.0` still apply.

### Removed

- The `openspec/` tree: the eight capability specs are `docs/specs/`, the
  proposal and task list are [`docs/HISTORY-0.1.0.md`](docs/HISTORY-0.1.0.md),
  and the beads execution mirror and its scripts are gone.
- The Python compatibility apparatus: `internal/conformance`, the
  `//go:build conformance` differential suite, `docs/DIVERGENCES.md`, the
  `conformance` capability spec, and the whole of `testdata/` — 72 files of
  recorded output and fixtures that every `go install` was downloading.
- The 17-row key table in the README, and the test that kept it in step with the
  help view. Press `?` in the session for the keys.

### Changed

- The vault format is specified by this project, in
  [`docs/specs/vault-format.md`](docs/specs/vault-format.md) and
  [`docs/FORMAT.md`](docs/FORMAT.md). Every code comment, spec and document that
  justified a rule as "the reference tool does X" now states the rule.
- The README is a hub — what it is, install, one quickstart loop, the session and
  where the documentation lives — and contribution guidance moved to
  [`CONTRIBUTING.md`](CONTRIBUTING.md).
- The `0.1.0` entry below was rewritten after its tag, so the text here differs
  from the copy inside the published `v0.1.0` module, which is frozen.

## [0.1.0] - 2026-10-18

`specshelf` is a Go implementation of the Markdown vault and the command surface
[`agent-spec-vault`](https://github.com/xixifast/agent-spec-vault) v0.1.2 began:
the same vault directory, the same documents and the same ids, in a single
binary, so a vault written by the Python tool keeps working. There is no `specv`
alias and `SPECV_HOME` is not read; the equivalent is `SPECSHELF_HOME`.

From this release the vault format is specified by this project, in
[`docs/specs/vault-format.md`](docs/specs/vault-format.md) and
[`docs/FORMAT.md`](docs/FORMAT.md), rather than by the tool the port started
from.

### The vault and the base commands

- The ten base commands, with stable flag names and defaults: `init`, `new`,
  `decision`, `link-session`, `list`, `search`, `show`, `index`, `prime`, `path`.
- The vault format: the directory layout, the frontmatter fields, the
  `spec-`/`decision-` document id with its `YYYYMMDD` date and `-N` collision
  suffix, the `YYYY-MM-DD-<slug>.md` file name, and the `index.jsonl` record.
  See [`docs/FORMAT.md`](docs/FORMAT.md).
- A vault that already exists keeps working: the reader normalises what it finds
  and the writer never emits a document it cannot read back. The slug rules, the
  id form and the index encoding are pinned by tests over the values that are
  hardest to get right — accented, CJK and emoji titles, punctuation-only and
  empty titles, and the awkward frontmatter shapes such documents carry.

### Behaviour decisions

Each item is a deliberate choice, here and in
[`docs/specs/`](docs/specs/), with a test that asserts it.

- Frontmatter is always written as valid YAML, so a third-party parser can read
  it.
- Every write is atomic — the complete bytes to a temporary file, flushed to
  disk, then renamed over the target — and creation is exclusive, retrying the
  next `-N` name. A crash, a full disk or two concurrent writers can no longer
  truncate a document or overwrite a just-created one.
- An empty value reads as the empty scalar rather than the text `[]`, and an
  empty list field is written `[]` rather than a bare key a YAML reader takes
  for null.
- The reader keeps documents a looser writer produces: a byte-order mark, an
  empty frontmatter block, CRLF and lone-CR endings, quoted keys and quoted
  values, list items at any indentation, and values holding exotic line
  separators. A rewrite emits LF.
- An unreadable document is skipped with one warning line on stderr naming its
  path and reason and a successful exit, instead of disappearing from listings
  silently; `doctor` still reports the same file as a problem.
- `index.jsonl` is regenerated, atomically, after every command that changes the
  vault, rather than only when `index` is run.
- Exit codes are defined and stable: `0` on success, `1` on a runtime failure,
  `2` on a usage failure, with one `specshelf: ` line on stderr and no stack
  trace.
- `rm`, `edit`, `status` and `link-session` refuse a reference that resolves to
  a path outside the vault, and a hand-picked `--slug` that would escape the
  document directory, leaving the target file untouched.

### New capabilities

- Nine commands beyond the base set: `edit`, `rm` (with `--dry-run`),
  `status`, `doctor`, `export` (JSON and JSONL), `stats`, `sync`, `upgrade` and
  `ui`.
- Machine-readable output: `--json` on the read commands, `--format
  tsv|table|csv|jsonl` on `list` and `search`, and `--color auto|always|never`.
  Colour never reaches `--json` output, a `--format` other than `table`, or the
  single-value output of `new`, `decision`, `link-session`, `path` and `index
  --print`.
- Configuration and named vaults: a TOML file at the platform configuration
  directory, a `SPECSHELF_CONFIG` override, per-setting precedence flag →
  environment → file → default, `--vault <name>` over a `[vaults.<name>]` table,
  and `--project` discovery of a `.specshelf/` vault by walking up from the
  working directory. See [`docs/CONFIG.md`](docs/CONFIG.md).
- `sync` commits the vault and pushes it with `--push`/`--no-push`, through the
  user's own `git` installation.
- `upgrade` reports and installs a newer version under the `go install`-only
  distribution, with `--check` to report without changing anything.
- `ui`: a full-screen browser over the vault with in-memory filtering, the
  editor handoff, a status change and a confirmed delete, reusing the store's
  guarantees. Press `?` in the session for the key bindings.
- `completion` for bash, zsh, fish and PowerShell, and `man` for the root
  command and every subcommand.
- Documentation: [`docs/FORMAT.md`](docs/FORMAT.md),
  [`docs/CONFIG.md`](docs/CONFIG.md),
  [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md),
  [`docs/specs/`](docs/specs/) and
  [`docs/HISTORY-0.1.0.md`](docs/HISTORY-0.1.0.md).

### Known limitations

- Distribution is `go install
  github.com/petsagouris/specshelf/cmd/specshelf@v0.1.0` only: no release
  binaries, no signing and no packaging manifests. Go 1.26 or newer is required.
  The suite runs in CI on Linux, macOS and Windows.
- `go install` and `upgrade` both need the Go module proxy, so there is no
  offline release artefact to fall back on.
