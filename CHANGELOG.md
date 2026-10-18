# Changelog

All notable changes to `specshelf` are documented in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2026-10-18

`specshelf` is a Go port and superset of
[`agent-spec-vault`](https://github.com/xixifast/agent-spec-vault) v0.1.2: the
same vault, the same documents and the same ids, in a single binary. One name
differs from the Python CLI — there is no `specv` alias and `SPECV_HOME` is not
read; the equivalent is `SPECSHELF_HOME`.

### Parity surface

- The ten reference commands, with the reference's flag names and defaults:
  `init`, `new`, `decision`, `link-session`, `list`, `search`, `show`, `index`,
  `prime`, `path`.
- The vault format: the directory layout, the frontmatter fields, the
  `spec-`/`decision-` document id with its `YYYYMMDD` date and `-N` collision
  suffix, the `YYYY-MM-DD-<slug>.md` file name, and the `index.jsonl` record.
  See [`docs/FORMAT.md`](docs/FORMAT.md).
- Compatibility with the Python tool, checked in both directions: a vault
  written by either is readable by the other. The default suite compares
  specshelf's output and files against committed fixtures from the reference;
  the `//go:build conformance` suite runs both tools against equivalent vaults
  and fails on a difference no entry in
  [`docs/DIVERGENCES.md`](docs/DIVERGENCES.md) explains.

### Corrected behaviours

Each item is an intentional difference from the reference, recorded with its own
test in [`docs/DIVERGENCES.md`](docs/DIVERGENCES.md).

- Frontmatter is always written as valid YAML, so a third-party parser can read
  it; the reference's writer emits a private line dialect.
- Every write is atomic — the complete bytes to a temporary file, flushed to
  disk, then renamed over the target — and creation is exclusive, retrying the
  next `-N` name. A crash, a full disk or two concurrent writers can no longer
  truncate a document or overwrite a just-created one.
- An empty value reads as the empty scalar rather than the text `[]`, and an
  empty list field is written `[]` rather than a bare key a YAML reader takes
  for null.
- The reader keeps documents the reference mangles or discards: a byte-order
  mark, an empty frontmatter block, CRLF and lone-CR endings, quoted keys and
  quoted values, list items at any indentation, and values holding exotic line
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

- Nine commands beyond the reference set: `edit`, `rm` (with `--dry-run`),
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
  guarantees. Its key bindings are in [`README.md`](README.md).
- `completion` for bash, zsh, fish and PowerShell, and `man` for the root
  command and every subcommand.
- Documentation: [`docs/FORMAT.md`](docs/FORMAT.md),
  [`docs/DIVERGENCES.md`](docs/DIVERGENCES.md),
  [`docs/CONFIG.md`](docs/CONFIG.md) and
  [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

### Known limitations

- Distribution is `go install
  github.com/petsagouris/specshelf/cmd/specshelf@v0.1.0` only: no release
  binaries, no signing and no packaging manifests. Go 1.26 or newer is required.
  The suite runs in CI on Linux, macOS and Windows.
- `go install` and `upgrade` both need the Go module proxy, so there is no
  offline release artefact to fall back on.
