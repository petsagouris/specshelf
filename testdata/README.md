# `testdata/` — generated reference fixtures

Everything in this directory is **generated output of the Python reference
implementation** of [agent-spec-vault](https://github.com/xixifast/agent-spec-vault).
It is not hand-written and **must not be hand-edited**. If a fixture looks wrong,
fix the generator (see [How to regenerate](#how-to-regenerate)) or record the
difference as an intentional divergence; never edit the bytes here.

## Upstream reference

| Item | Value |
| --- | --- |
| Project | `agent-spec-vault` (Python, MIT) |
| Version | `0.1.2` |
| Upstream URL | <https://github.com/xixifast/agent-spec-vault> |
| Source distribution | <https://files.pythonhosted.org/packages/f5/bd/a70a91de4b587108ac61d4d7efeb345ade4723de481d9f8fd03175c015f9/agent_spec_vault-0.1.2.tar.gz> |
| `python3 --version` | `Python 3.14.8` (`3.14.8 (main, Oct  2 2026, 10:23:31) [GCC 16.2.1 20260810]`) |
| Tool version output | `python3 -m specv --version` → `specv 0.1.2` |
| Install route used | **Checkout, no install.** The sdist was unpacked under `/tmp/opencode/pyspecv/agent_spec_vault-0.1.2` and the tool run as `python3 -m specv` with that directory as the working directory (upstream documents running from a checkout without installing). The `--home <dir>` flag was passed on every invocation; the isolated-install fallback was **not** needed. |
| Date of generation | 2026-10-17 (local date; the tool stamps this into ids and file names) |

All invocations were run as:

```text
cd /tmp/opencode/pyspecv/agent_spec_vault-0.1.2
python3 -m specv --home <vault-dir> <args...>
```

The real `~/.agents/spec-vault` was never touched.

## Directory contents

| Path | What it is |
| --- | --- |
| `slug_corpus.json` | One `new` document per title, each in its own fresh vault directory. Records the frontmatter `id`, the slug (id minus the leading `spec-YYYYMMDD-`), and the file name produced for 15 titles that exercise Unicode, punctuation-only, whitespace-only, over-long and empty titles. |
| `python_vault/` | A single vault created by the Python tool: `README.md`, `templates/{spec,decision}.md`, `index.jsonl`, `specs/` and `decisions/`. Titles here deliberately stress the frontmatter dialect. Copied verbatim from the generator's scratch vault. |
| `python_vault_expected/` | Recorded `stdout`, `stderr` and exit code of ten CLI invocations run against a scratch copy of `python_vault/`. One `<name>.stdout` / `<name>.stderr` pair per command plus `exit_codes.json`. |
| `README.md` | This file. |

### `python_vault/specs/` (9 documents)

```text
2026-10-17-1-priority.md                          (#1 priority)
2026-10-17-992681441c.md                          (质量分析展示契约; slug is a sha1 prefix)
2026-10-17-bold-start.md                          (*bold* start)
2026-10-17-da39a3ee5e.md                          (empty title; sha1("")[:10])
2026-10-17-dash-start.md                          (- dash start)
2026-10-17-fix-42-the-thing.md                    (Fix #42: the thing)
2026-10-17-he-said-hi.md                          (He said "hi")
2026-10-17-quality-analysis-display-contract.md   (Quality analysis display contract)
2026-10-17-trailing-space.md                      (Trailing space , trailing space stripped in frontmatter)
```

### `python_vault/decisions/` (1 document)

```text
2026-10-17-use-markdown-vault-instead-of-beads.md (Use Markdown vault instead of Beads)
```

## Exact generation commands

Driver scripts live in `/tmp/opencode/pyspecv/` (scratch, not committed):
`driver_slug_corpus.py`, `driver_python_vault.py`, `driver_expected.py`. All three
call the CLI with argument **lists** via `subprocess` (no shell strings), so
spaces, quotes and leading punctuation in titles are passed literally.

### `slug_corpus.json`

For each title, in order, a fresh empty vault directory `/tmp/opencode/pyspecv/run_slug_corpus/NN/`:

```text
python3 -m specv --home /tmp/opencode/pyspecv/run_slug_corpus/NN new "<title>"
```

Then the created file's frontmatter `id` and the file's base name are read off
disk. Titles, exactly as passed:

1. `Quality analysis display contract`
2. `质量分析展示契约`
3. `不用 Beads 管历史 spec`
4. `Über Straße`
5. `Fix #42: the thing`
6. `Café résumé naïve`
7. `Ångström Øresund`
8. `Привет мир`
9. `日本語 mixed Latin`
10. `emoji 🚀 rocket`
11. `!!!???`
12. `Trailing   spaces   and---dashes`
13. `a very long title that goes on and on and on and on and on and on and on and on and on`
14. `` (the empty string)
15. `Ünïcödé mixed with trailing punctuation!!`

### `python_vault/`

One vault directory `/tmp/opencode/pyspecv/vault_gen/`, commands in this order
(never `init` twice), then a verbatim `shutil.copytree` to `testdata/python_vault/`:

```text
python3 -m specv --home <v> init
python3 -m specv --home <v> new "Quality analysis display contract" --repos lippi-smart-customer,aics-web-repos --tags quality-analysis,contract --codex-session session-a,session-b
python3 -m specv --home <v> decision "Use Markdown vault instead of Beads" --tags tooling,specs --status accepted
python3 -m specv --home <v> new "质量分析展示契约"
python3 -m specv --home <v> new "Fix #42: the thing"
python3 -m specv --home <v> new "#1 priority"
python3 -m specv --home <v> new "- dash start"
python3 -m specv --home <v> new "*bold* start"
python3 -m specv --home <v> new 'He said "hi"'
python3 -m specv --home <v> new "Trailing space "
python3 -m specv --home <v> new ""
python3 -m specv --home <v> link-session spec-20261017-quality-analysis-display-contract --codex-session session-c
python3 -m specv --home <v> index
```

The id passed to `link-session` is the one produced for the second document,
resolved by reading that document's frontmatter rather than being hard-coded.

### `python_vault_expected/`

`testdata/python_vault/` was copied to the scratch directory
`/tmp/opencode/pyspecv/expected_vault/` (and a fresh empty directory
`/tmp/opencode/pyspecv/expected_init/` created for `init`), then each command
below was run with the scratch directory as `--home` and its `stdout`, `stderr`
and exit code recorded byte-for-byte. Reads never touched the committed vault.

| Name (`<name>.stdout` / `.stderr`) | Command |
| --- | --- |
| `init` | `init` (empty directory `/tmp/opencode/pyspecv/expected_init`) |
| `list` | `list` |
| `list_json` | `list --json` |
| `list_kind_decision_tag_tooling` | `list --kind decision --tag tooling` |
| `search_beads` | `search Beads` |
| `search_quality` | `search quality` |
| `show` | `show spec-20261017-quality-analysis-display-contract` |
| `prime_repo` | `prime --repo lippi-smart-customer` |
| `index_print` | `index --print` |
| `path` | `path` |

`exit_codes.json` maps each name to its exit code; all ten exit `0` and all ten
`stderr` files are empty.

> Note: `init`, `index_print` and `path` print absolute paths, so those three
> `.stdout` files embed the scratch directories used at generation time
> (`/tmp/opencode/pyspecv/expected_init` and `/tmp/opencode/pyspecv/expected_vault`).
> A consumer comparing them must normalise the vault path. The other outputs use
> vault-relative paths and are stable.

## How to regenerate

1. Download the sdist named above and unpack it under a scratch directory, e.g.
   `/tmp/opencode/pyspecv/`. It contains a `specv/` package directory.
2. From inside the unpacked directory, confirm `python3 -m specv --version`
   prints `specv 0.1.2`. If that fails, fall back to an isolated install:
   `python3 -m pip install --target /tmp/opencode/pyspecv-lib agent-spec-vault==0.1.2`
   and run with `PYTHONPATH=/tmp/opencode/pyspecv-lib python3 -m specv`.
3. Re-run the three driver scripts in `/tmp/opencode/pyspecv/`. They rewrite
   `slug_corpus.json`, `python_vault/` and `python_vault_expected/` in place.
   Reproduce the commands above by hand only if a driver is unavailable — keep
   the invocation order exact.
4. Generation is date-dependent: the tool stamps the current local date into
   every id (`spec-YYYYMMDD-...`) and file name. Regenerating on a different day
   changes those, and therefore the fixtures. Update this README's date if you
   intend to commit a regenerated set.

## python_vault_rewritten/

The "after one rewrite" state of `python_vault/`. Our writer must reproduce
upstream's output for documents it *rewrites* (unknown keys preserved, field
order stable), not only for freshly created ones; tasks 2.2 and 3.12 need a
reference for that.

Every one of the ten documents in `python_vault/` was rewritten once by the
provenance-carrying `link-session` command, adding the same session id
`roundtrip-1`. Nothing else about the documents was touched by hand.

- `python_vault_rewritten/` — the rewritten vault, copied verbatim. Same ten
  documents, same ids and same file names as `python_vault/`; `link-session`
  never renames a file or changes an id. Each document now carries
  `codex_sessions` containing `roundtrip-1` (the spec that already had
  `session-a`/`session-b`/`session-c` gained `roundtrip-1` after them), and
  `updated` is still `2026-10-17` because the rewrite happened the same day it
  was created. The empty-title document's `title:` line, which upstream reads
  back as an empty list, is rendered as `title: []` on rewrite.
- `python_vault_rewritten_expected/` — recorded `stdout`, `stderr` and exit code
  of each `link-session` invocation, named `link_session_<file stem>.stdout` /
  `.stderr`, plus `exit_codes.json` mapping that name to its exit code. Layout
  matches `python_vault_expected/`. All ten exit `0` and all ten `.stderr` files
  are empty.

### Exact generation commands

Step 1, the vault to rewrite (a copy, so the committed vault is never written):

```text
rm -rf /tmp/opencode/rt2 && cp -r testdata/python_vault /tmp/opencode/rt2
```

Step 2, from `/tmp/opencode/pyspecv/agent_spec_vault-0.1.2`, once per document.
The ids were read out of the copied files, never retyped, and the loop below is
conceptual — the driver called the CLI with argument lists via `subprocess`, not
a shell string:

```text
for id in <ids read from /tmp/opencode/rt2/specs/*.md and /tmp/opencode/rt2/decisions/*.md>; do
  python3 -m specv --home /tmp/opencode/rt2 link-session "$id" --codex-session roundtrip-1
done
```

Step 3, regenerate the index:

```text
python3 -m specv --home /tmp/opencode/rt2 index
```

Step 4, copy verbatim to `testdata/python_vault_rewritten/` (a
`shutil.copytree`, no edits).

The driver is `/tmp/opencode/pyspecv/driver_rewritten.py` (scratch, not
committed).

> Note: each `link_session_*.stdout` file is the absolute path of the rewritten
> document, so it embeds the generation scratch directory `/tmp/opencode/rt2`.
> A consumer comparing them must normalise the vault path. The `.stderr` files
> are empty.
