# specshelf

A single-binary Go implementation of
[agent-spec-vault](https://github.com/xixifast/agent-spec-vault): a global,
Markdown-native vault of durable specs, cross-repo contracts and decisions for
AI coding agents.

Vaults written by that tool keep working. `go install` produces one binary for
Linux, macOS and Windows, with no Python toolchain required.

## Install

Requires Go 1.26 or newer.

```bash
go install github.com/petsagouris/specshelf/cmd/specshelf@latest
```

From a checkout, install the working copy instead:

```bash
go install ./cmd/specshelf
```

## Quickstart

The vault lives at `~/.agents/spec-vault` unless `--home`, `SPECSHELF_HOME` or
the configuration file names another. Create a vault, add a document, and find it
again:

```bash
# Create the vault layout in its default location.
specshelf init

# Add a spec, then list and search what the vault holds.
specshelf new Retries --repos billing --tags reliability
specshelf list
specshelf search retries

# Summarise the vault for an agent to read.
specshelf prime
```

The rest of the loop has the same shape. `show <ref>` prints one document,
`edit <ref>` hands it to your editor, `status <ref> <status>` moves its status,
`link-session <ref> --codex-session <id>` records which agent session produced
it, and `sync` commits and pushes the vault with your own `git`. They are
described here rather than run because each needs something the block above does
not have — a document reference, an editor, or a remote.

`specshelf --help` lists every command, and `specshelf <command> --help`
documents its flags.

## Interactive session

`specshelf ui` opens a full-screen browser over the vault: a list of every
document with its kind, title, status, update date and tags, a detail view
showing one document's frontmatter and body, a plain-text filter that narrows
the loaded list as you type, and a help view. Press `?` for the keys. It needs an
interactive terminal; when stdout is redirected it exits non-zero and points you
at the non-interactive commands instead.

## Documentation

Architecture, the vault format and the configuration file are described in
[`docs/`](docs/), and the behaviour each capability must deliver is specified in
[`docs/specs/`](docs/specs/).

Contributions are in [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Licence

MIT, retaining the upstream copyright notice — see `LICENSE` and `NOTICE`.
