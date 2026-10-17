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

## Development

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./...
```

## Licence

MIT, retaining the upstream copyright notice — see `LICENSE` and `NOTICE`.
