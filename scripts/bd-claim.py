#!/usr/bin/env python3
"""Claim a mirrored task before starting it, in one step.

`openspec/changes/specshelf-v1/tasks.md` is authoritative and beads is its mirror
(see AGENTS.md). A board that does not know what is being worked on stops being
useful for exactly the thing it exists for — surviving a context reset — so the
claim is the first action of a task, not the last:

    scripts/bd-claim.py 2.7 --worktree /tmp/opencode/worktrees/2.7
    scripts/bd-claim.py 4.5 --worktree /tmp/opencode/worktrees/4.5
    scripts/bd-claim.py 2.12                      # worked in this checkout

Claim for a delegated task *before* the subagent starts: a worktree does not carry
`.beads/` (it is ignored), so a subagent cannot claim its own bead and must not run
`bd` at all.

The script resolves the bead by its external reference, so a typo cannot fall
back to "the last touched issue" and mutate something unrelated. It refuses a
closed bead (the work is ticked in tasks.md; that is not an undone task) and
refuses to overwrite another actor's live claim.

    scripts/bd-claim.py <task-id> [--worktree PATH] [--dry-run]

The worktree is recorded in the bead's notes rather than its description, because
bd-sync.py rewrites descriptions from tasks.md and would wipe anything appended
there.
"""

import argparse
import json
import subprocess
import sys

failures = []


def bd(args, dry_run=False):
    """Run bd, returning parsed JSON when the output is JSON.

    Under --dry-run writes are only printed; reads always run, because the
    decision depends on the current board.
    """
    if dry_run:
        print("would run: bd " + " ".join(args))
        return None
    result = subprocess.run(["bd", *args], capture_output=True, text=True)
    if result.returncode != 0:
        failures.append(
            f"bd {' '.join(args[:3])}…: {result.stdout.strip()}{result.stderr.strip()}"
        )
        return None
    out = result.stdout.strip()
    if out.startswith("{") or out.startswith("["):
        return json.loads(out)
    return out


def find_bead(task_id):
    """Return the bead mirroring this task id, by external reference.

    This is a read, so it runs even under --dry-run: the decision depends on the
    current board, and a dry run that cannot see the board verifies nothing.
    """
    want = f"tasks.md#{task_id}"
    rows = bd(["list", "--json", "--all"]) or []
    for row in rows:
        if row.get("external_ref") == want:
            return row
    known = sorted(
        (r.get("external_ref") or "").removeprefix("tasks.md#")
        for r in rows
        if r.get("external_ref")
    )
    sys.exit(f"error: no bead mirrors task {task_id}.\nknown tasks: {', '.join(known)}")


def main():
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("task_id", help="task id from tasks.md, e.g. 2.12")
    parser.add_argument("--worktree", help="worktree path this task is worked in")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    bead = find_bead(args.task_id)
    bead_id, status = bead["id"], bead["status"]

    if status == "closed":
        sys.exit(
            f"error: {args.task_id} is already closed ({bead_id}). Tick it in "
            "tasks.md instead of claiming it; a closed bead mirrors a ticked task."
        )

    # bd --claim is idempotent for the same actor and refuses to overwrite
    # another actor's live claim, so the guard belongs to bd, not to this script.
    bd(["update", bead_id, "--claim"], args.dry_run)

    if args.worktree:
        note = f"worktree {args.worktree} on branch wt/{args.task_id}"
        recorded = (bd(["show", bead_id, "--json"]) or [{}])[0].get("notes") or ""
        if note in recorded:
            print(f"note already on {bead_id}: {note}")
        else:
            bd(["update", bead_id, "--append-notes", note], args.dry_run)

    if failures:
        sys.exit("\n".join(failures))

    if not args.dry_run:
        # Keep the exported board the web UI reads in step with the claim.
        bd(["export", "-o", ".beads/issues.jsonl"])

    after = (bd(["show", bead_id, "--json"]) or [{}])[0]
    print(
        f"claimed {bead_id} ({args.task_id}): status={after.get('status', status)} "
        f"assignee={after.get('assignee', '')}"
        + (f" worktree={args.worktree}" if args.worktree else "")
    )


if __name__ == "__main__":
    main()
