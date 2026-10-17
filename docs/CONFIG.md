# Configuration

`specshelf` reads user preferences from one TOML file. Every setting is
optional, a missing file is not an error, and any setting can be overridden per
invocation.

## Location

The file is read from:

1. the path named by the `SPECSHELF_CONFIG` environment variable, when it is set
   and non-empty; otherwise
2. `config.toml` inside a `specshelf` directory under the platform's user
   configuration directory (`os.UserConfigDir`), which is
   `$XDG_CONFIG_HOME/specshelf/config.toml` (usually `~/.config/...`) on Linux,
   `~/Library/Application Support/specshelf/config.toml` on macOS and
   `%AppData%\specshelf\config.toml` on Windows.

A file that does not exist is treated exactly like an empty file: every setting
keeps its built-in default and no warning is printed.

## Precedence

For every setting the value is taken from the first of:

1. the command-line flag,
2. the environment variable,
3. the configuration file,
4. the built-in default.

`specshelf path --json` reports which level selected the vault, so a surprising
value can be traced to its source.

## Settings

| Key | Type | Built-in default | Meaning |
| --- | --- | --- | --- |
| `default_vault` | string | `~/.agents/spec-vault` | Vault used when no flag, environment variable or selector names one. |
| `color` | `auto` \| `always` \| `never` | `auto` | When to emit ANSI colour. `auto` colours only a terminal with `NO_COLOR` unset and `TERM` not `dumb`. |
| `editor` | string | unset | Editor command for `edit` and the session. `$VISUAL` and `$EDITOR` take precedence when set. |
| `auto_index` | boolean | `true` | Regenerate `index.jsonl` after a command mutates the vault. |
| `project` | boolean | `true` | Allow `--project` to discover a `.specshelf` vault by walking up from the working directory. |
| `[git].remote` | string | unset | Remote `sync` pushes to. Empty uses the branch's configured remote. |
| `[git].push` | boolean | `true` | Push after committing. `sync --push` / `--no-push` override it. |
| `[vaults.<name>].path` | string | — | Path of a named vault, selectable with `--vault <name>`. |

Named vaults come from the file only; `--vault <name>` selects one. Using a name
that is not declared fails with an error listing the declared names.

## Sample file

```toml
# The vault to use when no flag, environment variable or selector names one.
default_vault = "~/.agents/spec-vault"

# Colour output: auto, always or never.
color = "auto"

# Editor command; $VISUAL and $EDITOR take precedence when set.
editor = "vim"

# Refresh index.jsonl after a command changes the vault.
auto_index = true

# Allow --project to discover a .specshelf vault by walking up from here.
project = true

[git]
# Remote to push to; empty uses the branch's configured remote.
remote = "origin"
# Push after committing. Override per run with --push or --no-push.
push = true

# Named vaults, selected with --vault <name>.
[vaults.work]
path = "~/vaults/work"

[vaults.archive]
path = "/srv/vaults/archive"
```

## Unknown keys

Keys the tool does not recognise are reported as one warning each
(`unknown configuration key "colour"`) and ignored; the command still runs.

## Malformed files

A file that is not valid TOML, or a value with the wrong type, fails the command
with an error naming the file and the line and column of the problem, for
example:

```
specshelf: /home/you/.config/specshelf/config.toml:2:13: expected ']' to close table name
```

The tool never silently falls back to defaults when the file it was told to read
is malformed.
