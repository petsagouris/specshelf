# Contributing

`specshelf` is a small tool with one maintainer, so the easiest change to accept
is a small, focused one. Bug fixes and documentation corrections need no
ceremony; anything larger is worth an issue first.

## Reporting an issue

Open one at <https://github.com/petsagouris/specshelf/issues>. Include the
command you ran, what you expected, and what happened — the raw output is worth
more than a description of it.

Never paste the contents of a real vault. `~/.agents/spec-vault` is your own
private data; reproduce with a temporary directory and `--home`.

## Proposing a change

Say what changes, why, and what it would break. If the change touches what is
written on disk, say what happens to vaults that already exist — that is the part
that has to be right.

## Building and verifying

Go 1.26 or newer is required.

```bash
go build ./... && go vet ./... && gofmt -l . && go test ./...
```

All four have to be clean before a pull request is ready. State the verification
for the work you are doing, and paste its raw output: a summary of the output is
not evidence. Fuzz runs (`go test -fuzz FuzzX -fuzztime 30s`) count as raw output
too. Tests use temporary directories; never point a command at the real vault.

The vault format is specified in [`docs/specs/vault-format.md`](docs/specs/vault-format.md)
and described in [`docs/FORMAT.md`](docs/FORMAT.md). A change to it is a
deliberate edit to that spec, and it covers documents already on disk.

## Where the rest is written down

- The behaviour each capability must deliver: [`docs/specs/`](docs/specs/).
- The package boundaries, and the one rule — `internal/vault` is the only package
  that touches vault files: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
- Delegated work: what an agent must be given, and the worktree rules —
  [`AGENTS.md`](AGENTS.md).
- How v0.1.0 was specified and verified: [`docs/HISTORY-0.1.0.md`](docs/HISTORY-0.1.0.md).
