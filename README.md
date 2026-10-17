# specshelf

A single-binary Go implementation of
[agent-spec-vault](https://github.com/xixifast/agent-spec-vault): a global,
Markdown-native vault of durable specs, cross-repo contracts and decisions for
AI coding agents.

Same vault, same documents, same ids — installable with `go install` on Linux,
macOS and Windows, with no Python toolchain required.

**Status: under construction.** The behaviour contract lives in `openspec/`;
`AGENTS.md` explains how the work is organised and where each requirement is
written down.

## Install

Not released yet. Until the first tag the only way to build is from a checkout:

```bash
go install ./cmd/specshelf
```

## Interactive session

`specshelf ui` opens a full-screen browser over the vault: a list of every
document with its kind, title, status, update date and tags, a detail view
showing one document's frontmatter and body, a plain-text filter that narrows
the loaded list as you type, and a help view. It needs an interactive terminal;
when stdout is redirected it exits non-zero and points you at the non-interactive
commands instead.

| Key | Action |
| --- | --- |
| `j`, `down` | move the selection down |
| `k`, `up` | move the selection up |
| `enter` | open the selected document |
| `esc` | go back or cancel |
| `/` | filter the list as you type |
| `K` | cycle the kind filter |
| `S` | cycle the status filter |
| `R` | cycle the repo filter |
| `T` | cycle the tag filter |
| `r` | reload the vault from disk |
| `c` | create a document |
| `tab` | switch kind in the create prompt |
| `?` | show this help |
| `q`, `ctrl+c` | quit |

## Development

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./...
```

## Licence

MIT, retaining the upstream copyright notice — see `LICENSE` and `NOTICE`.
