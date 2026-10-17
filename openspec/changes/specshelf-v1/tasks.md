# Tasks

## 1. M0 — Module and guardrails

- [x] 1.1 Initialise the module `github.com/petsagouris/specshelf` with Go 1.26, add `spf13/cobra`, and create `cmd/specshelf/main.go` whose root command prints help and reports a development version; verify with `go build ./...` and `go run ./cmd/specshelf --version`
- [x] 1.2 Add the package skeleton (`internal/frontmatter`, `internal/vault`, `internal/config`, `internal/gitx`, `internal/cli`, `internal/tui`, `internal/docs`) with one-line package docs; verify `go vet ./...` is clean
- [x] 1.3 Add `LICENSE` (MIT, retaining the upstream copyright line), `NOTICE`, a README skeleton stating the project is a Go port of `agent-spec-vault`, and `.gitignore`; verify the upstream notice appears in both LICENSE and README
- [x] 1.4 Add the CI workflow: build, `vet`, `gofmt -l`, and `go test ./...` on ubuntu, macos and windows, plus golangci-lint on ubuntu; verify it is accepted by GitHub, that the referenced actions exist, and that the same command sequence passes locally (the workflow is disabled on GitHub while the repository is private)
- [x] 1.5 Add the install smoke test that builds into a temporary `GOBIN` with `go install` and asserts `--version` prints the development marker; verify the test passes locally and in CI

## 2. M1 — Vault format

- [x] 2.1 Implement document splitting in the reader: a pure `splitFrontmatter(name, data)` that separates the frontmatter block from the body, normalises CRLF and lone CR to LF, strips a leading byte-order mark, treats a document with no frontmatter as body-only with empty metadata, accepts an empty block, and reports an unterminated block as an error naming the file (the exported `Parse` entry point arrives with the typed view in 2.3); verify with a table test covering each case
- [x] 2.2 Implement the line dialect: unquoted scalars, block sequences, blank lines ignored, lines without a key separator ignored, list items with no preceding key ignored, values keeping colons and hashes, and quoted values unquoted; verify with a table test covering each case
- [x] 2.3 Implement the typed view over the parsed fields: scalar fields (`id`, `kind`, `title`, `status`, `created`, `updated`) and list fields (`repos`, `tags`, `codex_sessions`), an empty value meaning the empty string for scalars and the empty list for lists, `title: []` meaning empty, and accessors that never return a list where a scalar is expected; verify with a table test including the empty-value cases
- [x] 2.4 Verify the reader against the reference fixtures: every document in `testdata/python_vault` and `testdata/python_vault_rewritten` parses with its expected title, kind and lists, including the invalid-YAML titles, and none reports an error; verify the fixture test passes
- [ ] 2.5 Implement the writer with the fixed field order, quoting rules and preserved unknown keys, and assert idempotence (writing twice is byte-identical); verify with round-trip tests
- [ ] 2.6 Add `go.yaml.in/yaml/v3` as a test-only dependency and a test that every frontmatter block emitted by the writer parses as valid YAML; verify the test fails when the quoting rules are weakened
- [ ] 2.7 Add a fuzz test for the reader asserting no panic and that any successfully parsed document re-emits parseable frontmatter; verify `go test -run Fuzz -fuzz FuzzReadFrontmatter -fuzztime 30s` finds no failure
- [x] 2.8 Implement `slugify` and id generation using `golang.org/x/text/unicode/norm`, and commit the title corpus (ASCII, accented, sharp S, Nordic letters, CJK, Cyrillic, emoji, punctuation-only, over-length) with expected values generated from the Python implementation; verify all corpus cases match
- [x] 2.9 Implement document file naming and the document id: the id is `spec-` or `decision-` followed by the local date as `YYYYMMDD` and the slug, with a `-N` suffix on collision, and the file name is the local date as `YYYY-MM-DD` followed by the slug and `.md`; verify tests cover same-title and same-day collisions and that no existing file is overwritten
- [x] 2.10 Implement vault layout initialisation with the built-in README and templates, creating only what is missing; verify tests cover a fresh directory and a vault with a user-edited template
- [ ] 2.11 Implement body seeding from the template for a document's kind, including recreating a deleted template, and normalise the title used for both the frontmatter and the heading (surrounding whitespace removed, inner runs collapsed) while the id and file name stay derived from the title as given; verify tests cover both kinds, the deleted-template case and a title carrying stray whitespace
- [ ] 2.12 Implement `index.jsonl` encoding with the documented field order, path relativisation, summary extraction, directory ordering and literal UTF-8; verify tests cover a mixed vault and a non-ASCII title
- [ ] 2.13 Write `docs/FORMAT.md` describing the vault layout, frontmatter fields, id derivation and index record, and add a test that parses the document sample from that file with the real reader; verify the test fails if the sample and the code disagree

## 3. M2 — Store and CLI parity (v0.1.0)

- [ ] 3.1 Implement vault resolution (`--home`, `SPECSHELF_HOME`, `--vault`, `--project`, config default, built-in default) with source reporting and rejection of conflicting selectors; verify tests cover each precedence step, both conflict cases and the usage exit code
- [ ] 3.2 Implement project discovery: walk up for `.specshelf`, stop at `$HOME` or the filesystem root, error when absent; verify tests cover a nested working directory, a missing vault and a marker above the home directory
- [ ] 3.3 Implement atomic writes (temp file in the destination directory, flush, rename) and exclusive creation with `-N` retry; verify tests simulate an interrupted rename and two concurrent creations of the same title
- [ ] 3.4 Implement unreadable-document handling: skip, one warning line on stderr naming path and reason, successful exit; verify tests assert stdout purity and the exit code, and that the same file makes `doctor` fail later (3.10)
- [ ] 3.5 Implement reference resolution (existing path, then id, then file name, then stem) with a non-zero error for unknown references; verify tests cover all four forms and the unknown case
- [ ] 3.6 Implement index refresh after every mutating command plus `index` and `index --print`; verify tests assert the index updates after create, edit and remove without an explicit `index` call
- [ ] 3.7 Implement the ten reference commands (`init`, `new`, `decision`, `link-session`, `list`, `search`, `show`, `index`, `prime`, `path`) with reference flags, aliases, defaults, limits and ordering; verify with `testscript` command tests, one script per command
- [ ] 3.8 Implement multi-value flag parsing and the session flags including `CODEX_THREAD_ID`/`CODEX_SESSION_ID` fallback and the error when neither is set; verify tests cover repeated flags, comma lists, whitespace, de-duplication and the missing-variable error
- [ ] 3.9 Implement the stdout/stderr contract and exit codes (0/1/2 with the `specshelf: ` prefix); verify `testscript` assertions cover payload-only stdout and each exit code
- [ ] 3.10 Implement `doctor` and `stats` over the store's validation API; verify tests cover a healthy vault, an unreadable file, a kind/directory mismatch, exit codes and grouped counts
- [ ] 3.11 Implement colour with the three modes and the gating rules; verify tests use a non-terminal writer and `NO_COLOR`, and assert that `--json`, `--format tsv` and single-value stdout never carry escapes
- [ ] 3.12 Generate the compatibility fixtures from the Python tool once, commit them under `testdata/`, and add the default-suite test that compares specshelf's output and files against them; verify the suite passes with Python unavailable
- [ ] 3.13 Write `docs/DIVERGENCES.md` listing every intentional divergence (valid-YAML emission, atomic writes, exclusive creation, stderr warnings, automatic index refresh, preserved-key ordering, new commands, exit codes, colour rules, empty scalar values reading as empty rather than as `[]`, normalised title whitespace in the body heading, and reader tolerance of documents the reference parser mangles or discards: a byte-order mark, an empty frontmatter block, quoted values unquoted, list items at any indentation, an intervening non-item line, empty list items, and values containing exotic line separators) and add one test per entry asserting exactly the documented behaviour; verify each test fails if the behaviour is reverted
- [ ] 3.14 Add the `//go:build conformance` differential suite that runs both tools against equivalent vaults and compares stdout, exit codes and resulting files, skipping when Python or the package is absent; verify it skips without Python and reports differences when a divergence is unlisted
- [ ] 3.15 Add the bidirectional interchange tests: a Python-written fixture vault is fully readable by specshelf with no warnings, and the reverse vault is readable by the Python tool inside the conformance suite; verify the default-suite direction passes without Python
- [ ] 3.16 Document install and quickstart in the README and add a test that runs the documented command sequence against a temporary vault; verify the test fails when a documented command or flag changes

## 4. M3 — Superset commands

- [ ] 4.1 Implement `edit` (editor resolution, `updated` only on change), `rm` with `--dry-run`, and `status <ref> <status>`; verify tests cover an unchanged edit, a dry-run removal and a status change with the date bump
- [ ] 4.2 Implement `export` (JSON and JSONL) with tests asserting every document appears once with index-equivalent fields
- [ ] 4.3 Implement `--json` on every read command and `--format tsv|table|csv|jsonl` on `list` and `search`; verify tests assert field-name parity with the index record and the default format
- [ ] 4.4 Implement `completion` for bash, zsh, fish and PowerShell and `man` for the root command and subcommands; verify a CI step sources the bash and zsh completions and generation mentions every command
- [ ] 4.5 Implement config loading: TOML at the platform config directory, `SPECSHELF_CONFIG` override, per-setting precedence, warnings for unknown keys, and errors naming file and location for malformed input; verify tests cover each case and write `docs/CONFIG.md` with a sample file the loader test parses
- [ ] 4.6 Implement named vaults and `--vault`, including the unknown-name error that lists declared names; verify tests cover selection and the error message
- [ ] 4.7 Implement `sync` through the `git` executable with `--push`/`--no-push`, nothing-to-commit success, initial-repository creation, rejection handling and a clear error when git is missing; verify tests run against temporary repositories and skip when git is unavailable
- [ ] 4.8 Implement `upgrade --check` against the module proxy and `upgrade` with the install-directory check, the print-the-command path, the missing-toolchain path and the Windows no-replace path; verify tests use a stubbed proxy endpoint and a non-install-directory executable
- [ ] 4.9 Add `--slug` to document creation with a test asserting a hand-picked slug is used verbatim and the id stays unique

## 5. M4 — Interactive session

- [ ] 5.1 Add `charm.land/bubbletea/v2` and build the model with list, detail, create and help views plus key handling; verify snapshot tests of the rendered view at three terminal sizes
- [ ] 5.2 Implement in-memory filtering and the kind/status/repo/tag filters; verify tests assert filtering narrows results without filesystem access (via a counting fake store)
- [ ] 5.3 Implement the `Editor` interface and the `$EDITOR` handoff with reload and index refresh; verify tests with a fake editor cover the unchanged and changed cases, including the `updated` rule
- [ ] 5.4 Route create, status change and delete through the store, and add the byte-identical test (session-created document versus `specshelf new`) plus an index-consistency test after deletion; verify both tests pass
- [ ] 5.5 Implement delete confirmation and cancellation; verify the cancelled-deletion test leaves the document in place
- [ ] 5.6 Implement the non-terminal guard and terminal restoration, including exit on resized or closed input; verify tests cover the non-TTY error and assert restoration on the quit path
- [ ] 5.7 Add an import-boundary test asserting `internal/tui` imports neither the frontmatter codec nor the filesystem packages and depends on `internal/vault` for every mutation; verify the test fails when a direct file read is added
- [ ] 5.8 Document the session and its key bindings in the README and verify the documented keys match the help view rendered in a test

## 6. M5 — Hardening and first tag

- [ ] 6.1 Run the full suite locally, including cross-compilation and `vet` for macOS and Windows, together with the conformance suite with Python available; verify the local run is green and, once the repository is public, that the workflow passes on all three runners with one covering test per divergence entry
- [ ] 6.2 Write `docs/ARCHITECTURE.md` describing the package boundaries and the single-writer rule, and add a test asserting the documented package list matches the directories on disk
- [ ] 6.3 Write `CHANGELOG.md` for v0.1.0 covering the parity surface, the corrected behaviours and the new capabilities
- [ ] 6.4 Tag `v0.1.0`, then verify on Linux, macOS and Windows that `go install github.com/petsagouris/specshelf/cmd/specshelf@v0.1.0` produces a binary whose `--version` reports the tag
- [ ] 6.5 Integration check: run the documented quickstart end to end on each operating system against a fresh vault, and run `doctor` afterwards; verify the exit codes and that no conformance divergence appears

## Workflow follow-up

- After M5, review the change with the OpenSpec verify workflow.
- Archive the change so the eight capability specs are promoted into `openspec/specs/`.
- Deferred by decision, not to be picked up here: the MCP server, merged multi-vault search, release artifacts and signing.
- CI is disabled on GitHub while the repository is private, because private-repository Actions are blocked by the account's billing gate. Until it is enabled, Windows and macOS runtime behaviour stays unverified; cross-compilation and `vet` for both platforms on every local verification run are the substitute.
