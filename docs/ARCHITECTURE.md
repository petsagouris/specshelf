# Architecture

`specshelf` is a pure one-shot CLI: it starts, does one thing, prints a result
and exits. There is no daemon, no watcher and no network listener, so the
packages below are the whole of the runtime structure.

The split is one package per reason to change, rather than one flat package or a
package per command: a boundary that exists for a stated reason is a boundary a
test can hold, and it keeps each surface's imports pointing one way. This file
records the boundaries that exist; the format of what is on disk belongs to
[`FORMAT.md`](FORMAT.md), the configuration file to [`CONFIG.md`](CONFIG.md), and
every intentional difference from the Python reference tool to
[`DIVERGENCES.md`](DIVERGENCES.md).

## The shape

```
argv → cmd/specshelf → internal/cli ─┬→ internal/vault ─┬→ internal/frontmatter
                                     │                  ├→ internal/config
                                     │                  └→ internal/docs
                                     └→ internal/config

internal/tui ──(its Store interface)──→ internal/vault
internal/gitx          leaf
internal/conformance   test-only: builds cmd/specshelf and drives the binary
```

Dependencies point one way. `internal/frontmatter`, `internal/config`,
`internal/docs`, `internal/gitx` and `internal/conformance` import nothing else
in this project. `internal/vault` builds on the first three. `internal/cli` and
`internal/tui` call `internal/vault` and never go around it. `cmd/specshelf`
imports `internal/cli` and nothing else.

## The single-writer rule

**`internal/vault` is the only package that touches vault files**, on read or on
write. Every surface — the CLI, the TUI, MCP later — calls the
store. A second write path is a defect, not a shortcut.

It exists for two reasons. First, the storage guarantees are the point of this
port: every write is a temp-file-plus-rename and every creation exclusive, so
concurrent agents cannot lose data. A second writer would need its own copy of
that logic, and the guarantee would hold for one surface and not the other.
Second, everything derived from a title — the slug, the document id, the file
name, the seeded body — is derived in one place inside `internal/vault`; a second
writer would reintroduce the collision and normalisation bugs the format work
fixed. The same aim applies from the TUI's side: behaviour cannot drift between
surfaces.

What the rule forbids:

- No package other than `internal/vault` may read or write a file under a vault.
  A surface holds a vault root path and calls the store; it does not open the
  document itself.
- `internal/tui` must not import `internal/frontmatter` or the filesystem
  packages. `TestImportBoundary` in `internal/tui/boundary_test.go` fails the
  build if it gains one.
- The CLI's `edit` command does not rewrite a document either. It passes a
  `vault.Editor` implementation into `vault.Edit`, which performs the write and
  the index refresh; the TUI uses the same seam.
- `internal/conformance` drives the built binary instead of calling the store, so
  it never becomes a second code path.

What the rule does not forbid: `internal/docs` returns template bytes in memory
rather than writing files, and `internal/config` reads the configuration file,
which is not a vault file. Test packages may copy fixture vaults around; no
production package other than `internal/vault` writes a vault document.

## Packages

Each row is one package directory, and this table is the package list that the
test in `cmd/specshelf` compares against the directories on disk.

| Package | Owns |
| --- | --- |
| `cmd/specshelf` | The process entry point: argv in, exit code out. It calls `cli.Main` and holds no logic. |
| `internal/cli` | The command surface: the cobra command tree, flags, aliases, output formats, colour, the stdout/stderr split and the exit codes. It turns arguments into calls on `internal/vault` and `internal/config` and writes no vault file itself. |
| `internal/vault` | The store, and the only package that touches vault files: resolution, project discovery, document create/read/edit/remove, atomic IO and exclusive creation, template seeding, index build, encode and refresh, slug and id derivation, reference resolution, list, search, query, validation and statistics. |
| `internal/frontmatter` | One document's bytes: the tolerant reader for the upstream line dialect and the writer with its fixed field order and quoting rules. It knows nothing about the vault. |
| `internal/config` | The TOML configuration file: where it is found, how it is parsed, and the per-setting precedence flag → environment → file → default. |
| `internal/gitx` | Committing and pushing a vault by running the user's own git binary. It is a leaf with no vault knowledge; the `sync` command in `internal/cli` calls it. |
| `internal/tui` | The interactive full-screen browser: model, views, key bindings, in-memory filtering and the editor handoff. It reaches the vault only through its own `Store` interface, one method per operation. |
| `internal/docs` | The embedded vault templates and README bytes a new vault is seeded with. |
| `internal/conformance` | The test-only compatibility package: the default suite builds the real binary and compares it against the committed Python fixtures; the `//go:build conformance` differential suite runs both tools against equivalent vaults. |

## Where the rest is written down

- The behaviour each capability must deliver: [`specs/`](specs/).
- Vault layout, frontmatter fields, document id, file name and index record:
  [`FORMAT.md`](FORMAT.md).
- Every intentional divergence from the reference tool, each with its own test:
  [`DIVERGENCES.md`](DIVERGENCES.md).
- Configuration keys, file location and precedence: [`CONFIG.md`](CONFIG.md).
- How v0.1.0 was specified and verified: [`HISTORY-0.1.0.md`](HISTORY-0.1.0.md).

The boundaries above are deliberate, and each has a reason that outlived the
change that introduced it: one package per reason to change, atomic writes with
no lock to leave behind, the TUI sharing the store instead of reaching the
filesystem, and a later MCP server arriving behind that same seam.
