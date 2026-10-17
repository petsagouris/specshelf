# Design

## Context

The reference implementation is 600 lines of Python (`specv/cli.py`,
`specv/vault.py`) that we have read in full, plus one test module. Its observable
behaviour is the compatibility target: see `specs/vault-format` and
`specs/vault-store`. Three of its properties shape this design more than
anything else:

1. Its frontmatter is **not YAML**. A hand-rolled 20-line parser reads
   `key: value` and `  - item` lines; the writer emits titles containing `: ` or
   `#` unquoted, which a real YAML parser rejects.
2. Its identity and file naming depend on Python's
   `unicodedata.normalize("NFKD", …)` plus an ASCII filter, with a SHA-1
   fallback for titles that filter away to nothing.
3. Writes are non-atomic and creation is non-exclusive, so concurrent agents can
   lose data — the failure mode this port exists to fix.

Constraints: Go 1.26+, pure one-shot CLI, `go install`-only distribution,
Linux/macOS/Windows parity, minimal runtime dependencies (see
`openspec/config.yaml`).

## Goals / Non-Goals

**Goals**

- Byte-compatible documents and identical ids, so one vault is usable by both
  tools in both directions.
- A storage layer with no data-loss failure modes: atomic writes, exclusive
  creation, a readable report instead of silent skips.
- The same storage layer used by every surface (CLI now, TUI in this change, MCP
  later), so behaviour cannot drift between surfaces.

**Non-Goals**

- Reimplementing the Python frontmatter dialect *as YAML*. We accept more than
  upstream on read and emit less than upstream on write; the difference is
  enumerated in the divergence list.
- Any cross-vault merging, indexing or search infrastructure beyond a linear
  scan. A vault is hundreds of small files; an inverted index would be a cache
  to invalidate for no measurable gain.
- Vendoring `git`, re-implementing git object handling, or shipping release
  artifacts.

## Decisions

### D1 — Seven packages, one reason each

`cmd/specshelf` (argv→exit code only) · `internal/frontmatter` (one document's
bytes) · `internal/vault` (the collection: resolution, discovery, CRUD, atomic
IO, index, slug, validation) · `internal/config` · `internal/gitx` ·
`internal/cli` · `internal/tui` · `internal/docs` (embedded templates).

*Alternatives*: one flat package (the format code becomes untestable in
isolation and the TUI can reach the filesystem); a package per command (theatre
for 19 commands that are each a thin argument parser). Rejected both. The
`index` and `slug` concerns deliberately live inside `vault`: they are derived
data and one function respectively, not modules.

### D2 — Hand-rolled frontmatter codec, YAML library only in tests

A ~150-line codec reads the upstream dialect tolerantly and writes a fixed field
order with scalars quoted only when required. `go.yaml.in/yaml/v3` is a
**test-only** dependency used to assert that everything we emit parses as real
YAML.

*Alternatives*: `yaml.Node` round-tripping (preserves key order and comments but
not quoting style, and cannot read `title: Foo: bar`, so it fails on real
upstream vaults); `goccy/go-yaml` (same read failure, different API); a
frontmatter library (`adrg/frontmatter`, `goldmark-meta` — both split on `---`
and hand back a map, losing order). Rejected: the read side must accept what the
Python tool writes, which is the definition of "not YAML".

### D3 — Slug and id via the ported algorithm

15 lines: `norm.NFKD` → keep ASCII → lowercase → `[^a-z0-9]+`→`-` → trim →
72-char cap → `sha1(title)[:10]` fallback. Measured 12/12 exact parity with
Python across ASCII, accented, ß, Ø, CJK, Cyrillic, emoji and punctuation-only
titles. The uppercase-then-lowercase step is what keeps the byte-slice truncation
correct: after the ASCII filter the string is pure ASCII, so Go's byte slicing
matches Python's character slicing.

*Alternatives*: `gosimple/slug` (MPL-2.0, last release 2024, transliterates
CJK/Cyrillic to pinyin, so it mints different ids for exactly the titles this
project sees; also returns an empty string for punctuation-only titles and does
not truncate); `rgglez/go-slugify` (untagged, zero importers, six weeks old);
`mozillazg/go-slugify` (2016, MIT, same transliteration). All rejected on id
parity. `new --slug` gives humans a way to pick a readable id.

### D4 — CLI on cobra alone

Measured: cobra-only is 8 modules / 2.4 MB stripped; adding `fang` for prettier
help is 42 modules / 4.4 MB. The full "batteries" stack (fang + lipgloss +
go-pretty + koanf + charm log) does not even compile without manually bumping
two `charmbracelet/x/*` modules. Cobra already generates completions and man
pages, which is the only reason we want a framework at all.

*Alternatives*: `fang` (a drop-in wrap of cobra, so deferring costs nothing —
this is the escape hatch if help output needs polish later); `urfave/cli/v3`;
stdlib `flag` (no subcommand dispatch, no GNU-style long flags); kingpin and
friends (unmaintained). Rejected.

### D5 — Colour from the standard library

~40 lines: `--color auto|always|never`, `auto` = stdout is a character device
AND `NO_COLOR` unset AND `TERM != dumb`. Applies to display commands only, never
to `--json`, and never to the single-value stdout of `new`, `path` and
`index --print`, which are parsed by callers.

*Alternatives*: `lipgloss` (+34 modules for styling), `fatih/color` (a
dependency for `\x1b[31m`). Rejected.

### D6 — Atomic writes, exclusive creation, no locking

Every write is temp-file-in-the-destination-directory + flush + rename. Creation
uses an exclusive create and retries the next free `-N` name, which is what makes
concurrent `new` calls safe. There is no lock file: the vault is single-user
local storage, an advisory lock needs platform-specific code
(`flock`/`LockFileEx`) and would still not protect against an editor writing the
file directly. Concurrent edits of one document are last-write-wins, documented.

*Alternatives*: `flock` via `x/sys` (cost without benefit at this scale);
optimistic version checks (would need per-document versions, a format change).

### D7 — TOML configuration, no framework

`os.UserConfigDir()/specshelf/config.toml`, overridable with `SPECSHELF_CONFIG`,
decoded by `pelletier/go-toml/v2` (+1 module, +0.5 MB), with precedence
flag > env > config > default resolved per setting.

*Alternatives*: YAML config (reintroduces the implicit-typing footguns the
frontmatter work exists to avoid, and adds a dependency we deliberately don't
carry); JSON (hostile to hand editing); `koanf` or `viper` (a framework for ~10
keys); `BurntSushi/toml` (equally good — kept as the swap if the pelletier
LICENSE, which GitHub reports as NOASSERTION, turns out not to be permissive).
Requirement: verify that LICENSE file before depending on it.

### D8 — Vault resolution order and project discovery

`--home` → `SPECSHELF_HOME` → `--vault <name>` → `--project` → config default →
`~/.agents/spec-vault`. Two explicit selectors are a usage error. `--project`
walks up for a `.specshelf` directory, stopping at the filesystem root or
`$HOME`, and errors rather than falling back when nothing is found. `path --json`
reports which selector won.

*Alternatives*: global-only (the reference behaviour, kept as the default);
project-local by default (breaks the cross-repo purpose and drops files into
users' repositories); merging global and project vaults (breaks `show <ref>`
uniqueness, ordering and limits — deferred, not forgotten); `.agents/` as the
marker (an unratified proposal, not a convention).

### D9 — git by subprocess

`sync` runs `git -C <vault> …` and nothing else. The user's credential helper,
SSH agent, `~/.ssh/config`, hooks and signing all keep working, at zero modules
and zero bytes.

*Alternatives*: `go-git/v5` — measured +48 modules and +6.8 MB for commit-and-push,
no credential-helper support, no partial clones, and it shells out to `git` for
`file://` remotes anyway; `go-git/v6` is still beta. Rejected.

### D10 — Upgrade against the module proxy

`upgrade --check` issues one HTTPS GET to `proxy.golang.org/.../@latest` and
compares versions. `upgrade` shells out to `go install …@latest` **only** when
`os.Executable()` lives in the directory `go install` writes to; otherwise it
prints the command and exits 0. On Windows it never attempts to replace the
running executable: the Go toolchain itself fails there (it removes, then
renames to `~`, then gives up with "Access is denied"), so we pre-empt it with a
clear instruction. On Unix the toolchain's remove-then-create sequence succeeds
over a running binary, unlike a hand-rolled `open(O_TRUNC)` updater, which hits
`ETXTBSY`.

*Alternatives*: a self-update library (`creativeprojects/go-selfupdate`) — it
downloads release assets we deliberately don't publish; publishing artifacts
with goreleaser and signing (reverses the distribution decision).

### D11 — TUI reuses the store and hands editing to the editor

`charm.land/bubbletea/v2` (v1 and the `github.com/charmbracelet/...` v2 paths
both fail to compile — the v2 modules declare `charm.land/...`). One model, three
views (list, detail, create), in-memory filtering over the loaded documents, and
no filesystem or frontmatter access at all: mutations call the same `vault`
functions the CLI calls, and editing is a `$EDITOR` handoff through a one-method
`Editor` interface so tests can inject a fake. An import-boundary test enforces
this. Measured cost: +19 modules / +1.4 MB for the core, before `bubbles`.

*Alternatives*: `rivo/tview` (credible, but pre-1.0 since forever and heavier to
theme); no TUI. Rejected for the stated scope; the TUI is also the designated
cut line if milestones run long.

### D12 — MCP deferred behind the same seam

Not in this change. The reason it can be deferred safely is D1: because `vault`
is the only writer and exposes typed operations, a future `internal/mcp` is a
thin adapter. Notes recorded for that day: the official
`modelcontextprotocol/go-sdk` speaks the current protocol revision (a +13
module / +5.6 MB cost); tool annotations are untrusted by clients and client-side
confirmation is the only human-in-the-loop mechanism; roots/sampling/logging were
deprecated in the 2026-07-28 revision, so there is no in-protocol sandbox to
design around and a `--allow-write` switch is the only real control.

### D13 — Testing: fixtures now, Python optionally

Default suite: unit tests, `testscript` command-level tests against real
binaries, and committed fixture vaults with expected outputs generated once from
the Python tool. `//go:build conformance` adds a live differential run that skips
when Python is absent. `go-cmp` for diffs. CI runs the default suite on Linux,
macOS and Windows, and the conformance suite on Linux with Python installed.

*Alternatives*: requiring Python in CI for every run (slow, brittle, and it makes
compatibility untestable for contributors); snapshot-only testing (cannot detect
a divergence from upstream).

## Risks / Trade-offs

- **Windows-specific behaviour is unverifiable on this machine** (console
  detection for colour, `os.Rename` over an open file, `go install` over a
  running `.exe`) → explicit tests for rename and path handling, plus
  cross-compilation and `go vet` for windows and darwin on every local
  verification run. The three-OS CI matrix stays in the repository but is
  disabled while it is private, so runtime behaviour on those platforms is
  unverified until it is enabled; the upgrade path pre-emptively refuses to
  self-replace on Windows.
- **We emit different bytes than upstream for titles needing quotes** → it is an
  enumerated divergence with its own test, and the Python reader accepts our
  output, so interchange holds in both directions.
- **Slug parity depends on x/text's Unicode tables matching Python's
  `unicodedata` version** → the committed corpus test catches a regression; if a
  future x/text changes a decomposition, that is a new enumerated divergence
  rather than a silent drift.
- **`go install` excludes anyone without a Go toolchain** → accepted, documented;
  goreleaser remains a later, purely additive change.
- **The TUI doubles the surface area and is the least testable part** → it is
  forbidden from owning any storage logic, its editor is injectable, and it is
  the declared cut line.
- **A last-write-wins collision can lose one of two simultaneous edits** →
  documented in the README; `sync` plus the vault's Markdown nature makes
  recovery a `git diff` away, which is the intended safety net.
- **`pelletier/go-toml/v2`'s LICENSE reads as NOASSERTION on GitHub** → read the
  file before depending on it; swap to `BurntSushi/toml` if it is not permissive.

## Migration Plan

No migration: this is a new module and the vault format is unchanged. Data safety
for a vault shared with the Python tool is handled by M2's interchange tests
before the first write path ships.

## Open Questions

None that would change the specs, the approach or the task breakdown.
