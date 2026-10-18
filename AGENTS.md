# AGENTS.md

Context packet for anyone — human or agent — working in this repository. Read it
before writing code, and give it to every delegated agent.

## What this is

`specshelf` is a Go port and superset of
[agent-spec-vault](https://github.com/xixifast/agent-spec-vault) (Python, MIT,
v0.1.2). It keeps durable specs, cross-repo contracts and ADR-style decisions as
Markdown under a user-level vault (`~/.agents/spec-vault` by default). What it
must do is specified in `docs/specs/`, not in this file.

## Read these before writing code

| What | Where |
| --- | --- |
| The behaviour contract to satisfy | `docs/specs/<capability>.md` |
| The package map, and the one rule | `docs/ARCHITECTURE.md` |
| The vault format on disk | `docs/FORMAT.md` |
| Configuration keys and precedence | `docs/CONFIG.md` |
| How v0.1.0 was built, and how each piece was verified | `docs/HISTORY-0.1.0.md` |

Capabilities: `vault-format`, `vault-store`, `cli`, `config`, `git-sync`,
`upgrade`, `tui`, `conformance`.

## Hard constraints

- Pure one-shot CLI. No daemon, no watcher, no network listener.
- Distribution is `go install github.com/petsagouris/specshelf/cmd/specshelf@version`
  only: no release artifacts, no signing, no packaging manifests.
- Linux, macOS and Windows; CI runs the test suite on all three.
- Runtime dependencies are exactly `spf13/cobra`, `golang.org/x/text`,
  `pelletier/go-toml/v2`, `charm.land/bubbletea/v2` (+ `charm.land/bubbles/v2`).
  Anything else needs an explicit decision first. Cobra's own document
  generation, which `man` uses, compiles in `cpuguy83/go-md2man/v2` and
  `russross/blackfriday/v2` as indirect modules rather than us hand-rolling
  roff.
- Charm imports are `charm.land/...`. The `github.com/charmbracelet/...` v2 paths
  do not compile.
- Nothing may depend on release assets, checksums or signatures: distribution is
  `go install` only, so the module is the whole of what ships.
- MIT, retaining the upstream copyright notice: this is a derived work — see
  `LICENSE` and `NOTICE`.
- Vault format, document bytes and document ids stay compatible with the Python
  tool. Every intentional divergence is listed in `docs/DIVERGENCES.md` and has a
  test that asserts it exactly.
- Go 1.26 minimum.

## Package map and the one rule

The package map — what each package owns and which may import which — is in
`docs/ARCHITECTURE.md`. Its package list is the single source of truth: a test in
`cmd/specshelf` compares it against the directories on disk, so a package added
or removed without updating the document fails the build.

**The one rule: `internal/vault` is the only package that touches vault files.**
Every surface — CLI now, TUI in this change, MCP later — calls it. A second write
path is a defect, not a shortcut.

## Verifying

```
go build ./... && go vet ./... && gofmt -l . && go test ./...
```

Cross-platform expectations are part of that verification. A test that compares a filesystem path must build it with `filepath`, never with a literal containing `/`, because the tool prints the platform's separator; the index record's `path` field is the single exception and stays `/` everywhere. Normalising separators must be lossless — `filepath.ToSlash` rewrites every backslash, including JSON's `\"` escape — so apply it to both sides of a comparison, or only where a path is really embedded.

Work you take on states its own verification before it starts, and the report
carries the raw output; a summary of the output is not evidence. Fuzz runs
(`go test -fuzz FuzzX -fuzztime 30s`) count as raw output too. Where a change is
specified as a numbered task, its verification is recorded in
`docs/HISTORY-0.1.0.md`.

Never point a command at the real vault. Tests use temporary directories.
`testdata/` holds generated fixtures from the Python reference implementation and
must not be hand-edited.

## Parallel work

Independent tasks run in **git worktrees**, one per task, so no agent ever edits
the same checkout as another:

```
git worktree add -b wt/<task-id> /tmp/opencode/worktrees/<task-id>
```

Rules:

- One agent owns one worktree, and one worktree carries one task.
- Commit before creating a worktree. A worktree branches from a commit, so
  branching from a dirty main checkout gives the agent a tree missing the work it
  depends on.
- The agent commits on its branch and reports the hash; the main agent reviews
  the diff before merging anything. Nothing is merged unreviewed.
- Never edit `docs/ARCHITECTURE.md` or `docs/specs/` in a worktree. Those are
  single-writer files in the main checkout, and an edit made on a branch silently
  diverges from the tree it merges into.
- A worktree is for tasks whose files are disjoint from work in flight. Two
  agents editing one package is not parallelism, it is a merge conflict with
  extra steps; combine those tasks into one worktree and one agent instead.
- Read-only work — research, verification passes, an independent review of
  something already written — needs no worktree. Say so in the brief.

## Subagent brief

Every delegation must contain all six of these, or it is not ready to send:

1. **Read first** — this file and the capability spec(s) in `docs/specs/` that
   cover the work, named explicitly.
2. **Task** — the exact work, quoted: what changes, where, and the observable
   result that means it is done.
3. **Scope** — the exact files or directories it may create or modify; everything
   else is read-only. It may not edit `docs/ARCHITECTURE.md` or `docs/specs/`;
   the main agent is their single writer.
4. **Verification** — the command to run, and the requirement to paste raw output.
5. **Environment** — Go 1.27.2 is on the PATH; scratch space is `/tmp/opencode`;
   no `sudo`, no system-wide installs, no network access beyond the Go module
   proxy and the sources named in the task.
6. **Report** — files changed, raw verification output, and anything surprising or
   ambiguous in the spec.

Delegations are command-sized, not requirement-sized: an agent that must re-read
five specs to write one test has cost more than it saved.

### Precision checklist

A brief that names a file but leaves the work to interpretation is not
precise enough. Add all of these:

- **Exact paths** to create or modify, and the exact paths that are read-only.
- **Exact commands** to run, copied literally, including flags and arguments.
- **Exact pass condition** — the observable result or transcript fragment that
  means the task is done, not "the test passes".
- **Stop-and-report** on anything not decided: an ambiguous spec, a needed
  decision, a file outside the scope, a command that fails twice. The delegate
  reports back with the raw failure instead of choosing an interpretation.
- **No improvisation**: no renaming, no reformatting, no "while I was here"
  cleanups, no substituting a different tool or layout.

Precision beats context when they conflict: a delegate told to copy an exact
command will not wander, whereas one told to "generate the fixtures" will invent
a layout. Give both where you have both; when in doubt, give the command.

The agent that writes code never verifies it. Verification of delegated work is
done by the main agent or by a separate reviewer agent.
