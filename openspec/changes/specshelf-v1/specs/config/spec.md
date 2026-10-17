# Spec Delta

## Purpose

Lets a user set defaults once — which vault to use, how output is coloured,
which editor to open, how syncing behaves, and named vaults — without repeating
flags, while keeping every setting overridable per invocation.

## ADDED Requirements

### Requirement: Config file location

The tool SHALL read its configuration from
`<user config directory>/specshelf/config.toml`, where the user config directory
is the platform's conventional location, and SHALL use the file named by
`SPECSHELF_CONFIG` instead when that variable is set. A missing configuration
file SHALL NOT be an error and SHALL leave every setting at its default.

#### Scenario: No configuration file

- **WHEN** any command runs with no config file present
- **THEN** it behaves exactly as with an empty configuration file

#### Scenario: Override through the environment

- **WHEN** `SPECSHELF_CONFIG` points at an existing file
- **THEN** the settings from that file apply and the default location is not read

### Requirement: Setting precedence

For every setting the tool SHALL apply, in order: the command-line flag, then the
environment variable, then the configuration file, then the built-in default.
The effective value SHALL be reportable so a user can see where a setting came
from.

#### Scenario: Flag beats config

- **WHEN** the config file sets a default vault and `--home` names another
- **THEN** the flag's vault is used

#### Scenario: Config applies when nothing overrides it

- **WHEN** only the config file sets a colour preference and no flag or environment variable is present
- **THEN** that preference takes effect

### Requirement: Configurable settings

The configuration SHALL cover: the default vault path, named vaults, colour
mode, the editor command, whether the index is refreshed automatically after
mutations, git sync behaviour (remote and whether to push), and whether project
vault discovery is enabled. Unknown keys SHALL be reported as a warning and
ignored rather than treated as fatal.

#### Scenario: Disabling automatic index refresh

- **WHEN** the configuration disables index auto-refresh and a document is created
- **THEN** `index.jsonl` is left unchanged until `specshelf index` runs

#### Scenario: Unknown key

- **WHEN** the configuration contains a key the tool does not know
- **THEN** a warning naming the key goes to stderr and the command completes normally

### Requirement: Named vaults

The configuration SHALL allow declaring named vaults with `[vaults.<name>]`
sections carrying at least a path. The `--vault <name>` selector SHALL use the
named vault's path, and an unknown name SHALL fail with an error that lists the
declared names.

#### Scenario: Selecting a named vault

- **WHEN** `[vaults.work]` declares a path and `--vault work` is used
- **THEN** commands operate on that path

#### Scenario: Unknown vault name

- **WHEN** `--vault personal` is used with no such section
- **THEN** the command fails, names the unknown vault, lists the declared names, and exits non-zero

### Requirement: Malformed configuration is reported

A configuration file that is not valid TOML, or whose values have the wrong
type, SHALL fail the command with an error naming the file and the location of
the problem inside it, and SHALL NOT fall back silently to defaults.

#### Scenario: Syntax error in the config file

- **WHEN** the config file contains an unclosed section header
- **THEN** the command exits non-zero with a message naming the file and the offending line, and no vault is modified
