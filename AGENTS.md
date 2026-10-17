# AGENTS.md

Context packet for anyone — human or agent — working in this repository. Read it
before writing code, and give it to every delegated agent.

## What this is

`specshelf` is a Go port and superset of
[agent-spec-vault](https://github.com/xixifast/agent-spec-vault) (Python, MIT,
v0.1.2). It keeps durable specs, cross-repo contracts and ADR-style decisions as
Markdown under a user-level vault (`~/.agents/spec-vault` by default). What we
are building is specified in OpenSpec, not in this file.

## Read these before writing code

| What | Where |
| --- | --- |
| Why, what changes, breaking changes, out of scope | `openspec/changes/specshelf-v1/proposal.md` |
| The behaviour contract to satisfy | `openspec/changes/specshelf-v1/specs/<capability>/spec.md` |
| Design decisions D1–D13, each with its rejected alternatives | `openspec/changes/specshelf-v1/design.md` |
| The work, with each task's own verification | `openspec/changes/specshelf-v1/tasks.md` |
| Non-negotiable project constraints | `openspec/config.yaml` |

Capabilities: `vault-format`, `vault-store`, `cli`, `config`, `git-sync`,
`upgrade`, `tui`, `conformance`.

## Hard constraints

- Pure one-shot CLI. No daemon, no watcher, no network listener.
- Distribution is `go install github.com/petsagouris/specshelf/cmd/specshelf@version`
  only: no release artifacts, no signing, no packaging manifests.
- Linux, macOS and Windows; CI runs the test suite on all three.
- Runtime dependencies are exactly `spf13/cobra`, `golang.org/x/text`,
  `pelletier/go-toml/v2`, `charm.land/bubbletea/v2` (+ `charm.land/bubbles/v2`).
  Anything else needs an explicit decision first.
- Charm imports are `charm.land/...`. The `github.com/charmbracelet/...` v2 paths
  do not compile.
- Vault format, document bytes and document ids stay compatible with the Python
  tool. Every intentional divergence is listed in `docs/DIVERGENCES.md` and has a
  test that asserts it exactly.
- Go 1.26 minimum.

## Package map and the one rule

`cmd/specshelf` argv → exit code · `internal/frontmatter` one document's bytes ·
`internal/vault` resolution, discovery, CRUD, atomic IO, index, slug, validation ·
`internal/config` · `internal/gitx` · `internal/cli` · `internal/tui` ·
`internal/docs` embedded templates.

**The one rule: `internal/vault` is the only package that touches vault files.**
Every surface — CLI now, TUI in this change, MCP later — calls it. A second write
path is a defect, not a shortcut.

## Verifying

```
go build ./... && go vet ./... && gofmt -l . && go test ./...
```

Each task in `tasks.md` states its own verification. Run it and paste the raw
output; a summary of the output is not evidence. Fuzz runs
(`go test -fuzz FuzzX -fuzztime 30s`) count as raw output too.

Never point a command at the real vault. Tests use temporary directories.
`testdata/` holds generated fixtures from the Python reference implementation and
must not be hand-edited.

## Execution mirror (beads)

`openspec/changes/specshelf-v1/tasks.md` is **authoritative** for scope and
completion. A beads board mirrors it so execution state — claims, blockers, work
found mid-task — survives a context reset. The board is project-local
(`.beads/`), derived, and never committed.

```
scripts/bd-sync.py                 # after editing tasks.md; idempotent, --dry-run supported
bd ready                           # unblocked work
bd update <bead-id> --claim        # take a task before starting it
bd unclaim <bead-id>               # release a claim whose agent died
bd --help                          # command reference (safe, unlike `bd prime`)
```

Keep beads' own *rule text* out of an agent's context. `bd setup opencode` appends
a section to this file and `bd prime` injects one at runtime; both say "use beads
for ALL task tracking" and "you MUST `bd close` before saying done", which
contradict the precedence above, and neither can be configured to stop — only
`no-git-ops` and `agent.profile` change the git wording. So: do not run
`bd setup`, do not install the editor integrations, prefer `bd --help` over
`bd prime`. (`bd hooks install` is unrelated — it installs git hooks, not context
injection.) Beads is built for repositories where it *is* the tracker; this one is
the inverse, so its session-close protocol points the wrong way here.

- A bead exists only because a task exists. Work discovered mid-task is recorded
  in beads *and* added to `tasks.md`; the reverse never happens.
- Completion is ticked in `tasks.md` first and then mirrored. Never close a bead
  to mean "done" — that is exactly the drift this rule prevents.
- Titles are labels truncated to fit; the bead description holds the task text
  verbatim.
- The script does not derive task-to-task dependencies, because tasks.md does not
  record them. Add a real blocker with `bd dep add` when you find one.

Per-project settings this relies on, stored in `.beads/config.yaml` and therefore
**lost on a fresh clone** — reapply them after `bd init`:

```
bd config set events-journal true   # the board refreshes in seconds without re-reading everything
bd config set export.auto true      # keeps .beads/issues.jsonl fresh after writes
bd export -o .beads/issues.jsonl    # seed it once: `bd list --json` omits dependency
                                    # links, so the board builds its epic→task tree
                                    # from this file, and nothing creates it by default
```

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
- Never edit `tasks.md` or anything under `openspec/` in a worktree. Those are
  single-writer files in the main checkout, and a tick made on a branch is lost.
- A worktree is for tasks whose files are disjoint from work in flight. Two
  agents editing one package is not parallelism, it is a merge conflict with
  extra steps; combine those tasks into one worktree and one agent instead.
- Read-only work — research, verification passes, an independent review of
  something already written — needs no worktree. Say so in the brief.

## Subagent brief

Every delegation must contain all six of these, or it is not ready to send:

1. **Read first** — this file, the capability spec(s) covering the work, and the
   design decisions it depends on, named explicitly (e.g. "D2 and D6").
2. **Task** — the task id and its exact text from `tasks.md`.
3. **Scope** — the exact files or directories it may create or modify; everything
   else is read-only. It may not edit `tasks.md` (the main agent is its single
   writer) and may not edit anything under `openspec/`.
4. **Verification** — the command to run, and the requirement to paste raw output.
5. **Environment** — Go 1.27.2 is on the PATH; scratch space is `/tmp/opencode`;
   no `sudo`, no system-wide installs, no network access beyond the Go module
   proxy and the sources named in the task.
6. **Report** — files changed, raw verification output, and anything surprising or
   ambiguous in the spec.

Delegations are command-sized, not requirement-sized: an agent that must re-read
five specs to write one test has cost more than it saved.

### Precision checklist

A brief that names the task id but leaves the work to interpretation is not
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
