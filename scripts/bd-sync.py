#!/usr/bin/env python3
"""Mirror the OpenSpec task list into beads.

`openspec/changes/specshelf-v1/tasks.md` is authoritative: it decides what is in
scope and what is complete. This script derives a beads mirror from it so that
execution state — claims, blockers, work discovered mid-task — has somewhere to
live that survives a context reset.

    scripts/bd-sync.py [--dry-run]

The mirror is keyed on the external reference `tasks.md#<task id>`, so a re-run
only creates what is missing and closes or reopens what changed. Each milestone
becomes an epic with its tasks as children, and an epic closes only once all of
its tasks are closed.

Titles are labels, truncated to fit bd's limit; the description carries the task
text verbatim. Task-to-task dependencies are deliberately NOT derived: tasks.md
does not record them, and guessing them from prose would put invented structure
in the board. Record a real blocker with `bd dep add` when you find one.
"""

import json
import pathlib
import re
import subprocess
import sys

TASKS_MD = (
    pathlib.Path(__file__).resolve().parent.parent
    / "openspec/changes/specshelf-v1/tasks.md"
)
MILESTONE = re.compile(r"^## (\d+)\.\s+(.*)$")
TASK = re.compile(r"^- \[( |x)\] (\d+\.\d+)\s+(.*)$")
TITLE_LIMIT = 200

failures = []


def bd(args, dry_run=False):
    """Run bd, returning parsed JSON when the output is JSON.

    Under --dry-run writes are only printed. Read commands always run, because
    the decision to create or update depends on the current board.
    """
    if dry_run:
        print("would run: bd " + " ".join(args))
        return None
    result = subprocess.run(["bd", *args], capture_output=True, text=True)
    if result.returncode != 0:
        failures.append(f"bd {' '.join(args[:4])}…: {result.stdout.strip()}{result.stderr.strip()}")
        return None
    out = result.stdout.strip()
    if out.startswith("{") or out.startswith("["):
        return json.loads(out)
    return out


def label(prefix, text):
    """Build a title that fits bd's title limit, keeping the text in the body."""
    text = " ".join(text.split())
    if len(prefix) + len(text) > TITLE_LIMIT:
        cut = text[: TITLE_LIMIT - len(prefix)].rsplit(" ", 1)[0]
        text = cut + "…"
    return f"{prefix} {text}"


def parse_tasks():
    milestones, tasks = [], []
    for line in TASKS_MD.read_text().splitlines():
        if m := MILESTONE.match(line):
            milestones.append((m.group(1), m.group(2).strip()))
            continue
        if t := TASK.match(line):
            tasks.append(
                {
                    "done": t.group(1) == "x",
                    "id": t.group(2),
                    "text": t.group(3).strip(),
                    "milestone": milestones[-1][0] if milestones else "0",
                }
            )
    return milestones, tasks


def main():
    dry_run = "--dry-run" in sys.argv
    milestones, tasks = parse_tasks()
    existing = {
        b["external_ref"]: b
        for b in (bd(["list", "--json", "--all"]) or [])
        if b.get("external_ref")
    }

    created = closed = reopened = 0
    epics = {}

    for number, title in milestones:
        ref = f"tasks.md#milestone-{number}"
        if bead := existing.get(ref):
            epics[number] = (bead["id"], bead["status"])
            continue
        bead = bd(
            [
                "create",
                title,
                "-t",
                "epic",
                "--external-ref",
                ref,
                "--description",
                f"Milestone {number} of the specshelf-v1 change",
                "--json",
            ],
            dry_run,
        )
        epics[number] = (bead["id"] if bead else "<new>", "open")
        created += 1

    open_by_milestone = {}
    for task in tasks:
        ref = f"tasks.md#{task['id']}"
        open_by_milestone[task["milestone"]] = open_by_milestone.get(
            task["milestone"], 0
        ) + (0 if task["done"] else 1)

        bead = existing.get(ref)
        if not bead:
            bd(
                [
                    "create",
                    label(f"[{task['id']}]", task["text"]),
                    "-t",
                    "task",
                    "--external-ref",
                    ref,
                    "--description",
                    task["text"],
                    "--deps",
                    f"parent-child:{epics[task['milestone']][0]}",
                    "--json",
                ],
                dry_run,
            )
            created += 1
            continue

        want = "closed" if task["done"] else "open"
        if bead["status"] == want:
            continue
        if task["done"]:
            print(f"closing {bead['id']} ({task['id']})")
            bd(["close", bead["id"]], dry_run)
            closed += 1
        else:
            print(f"reopening {bead['id']} ({task['id']})")
            bd(["update", bead["id"], "-s", "open"], dry_run)
            reopened += 1

    # An epic is done only when every task under it is done, and bd refuses to
    # close one with open children, so only close when it will succeed.
    for number, (epic_id, status) in epics.items():
        if status == "closed" or open_by_milestone.get(number, 0) > 0:
            continue
        print(f"closing epic {epic_id} (milestone {number})")
        bd(["close", epic_id], dry_run)
        closed += 1

    print(
        f"{len(tasks)} tasks and {len(milestones)} milestones mirrored: "
        f"{created} created, {closed} closed, {reopened} reopened"
        + (" (dry run)" if dry_run else "")
    )
    if failures:
        print(f"\n{len(failures)} failure(s):", file=sys.stderr)
        for failure in failures:
            print(f"  {failure}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
