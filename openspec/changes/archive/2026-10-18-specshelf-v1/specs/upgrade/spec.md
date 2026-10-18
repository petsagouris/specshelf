# Spec Delta

## Purpose

Keeps a `go install`-only distribution current: it can tell a user whether a
newer version exists, and can apply the update when it is safe to do so, without
the project publishing release artifacts or a self-update client.

## ADDED Requirements

### Requirement: Version check without side effects

`specshelf upgrade --check` SHALL query the Go module proxy for the latest
version of the module and report the running version and the available version.
It SHALL NOT write to the vault, the repository or the filesystem, and SHALL
exit `0` when an update is available as well as when the binary is current.

#### Scenario: Update available

- **WHEN** `upgrade --check` runs and the proxy reports a version newer than the running binary
- **THEN** both versions are printed and the exit code is 0

#### Scenario: Already current

- **WHEN** the running version is the latest published version
- **THEN** the output says so and the exit code is 0

#### Scenario: Proxy unreachable

- **WHEN** the module proxy cannot be reached
- **THEN** the command exits non-zero with a message naming the network failure and no version claim

### Requirement: Upgrades only apply to the running installation

`specshelf upgrade` SHALL install a new version only when the running executable
is located in the current installation directory that `go install` writes to.
When it is not, the command SHALL print the exact `go install` command to run,
name the path of the running executable, and exit `0`, because a binary installed
by other means must not be silently duplicated or replaced.

#### Scenario: Binary installed by go install

- **WHEN** `upgrade` runs and the running executable lives in the install directory
- **THEN** the update is applied and the newly installed version is reported

#### Scenario: Binary copied elsewhere

- **WHEN** `upgrade` runs from a binary in `/usr/local/bin` that was copied there
- **THEN** nothing is installed, the recommended command is printed, and the exit code is 0

### Requirement: Upgrades never break the running process

The tool SHALL NOT attempt to overwrite a running executable on Windows; on that
platform it SHALL print the command to run in a fresh shell instead. On every
platform, a failed or interrupted upgrade SHALL leave a working binary in place.

#### Scenario: Windows upgrade attempt

- **WHEN** `upgrade` runs on Windows
- **THEN** no attempt is made to replace the running executable and the command to run is printed

#### Scenario: Install tooling unavailable

- **WHEN** the Go toolchain is not available on the PATH
- **THEN** the command reports that the toolchain is required, prints the install command, and does not modify anything

### Requirement: No other network access

`upgrade --check` SHALL be the only operation in the program that performs a
network request, with the exception of git operations performed by `sync`. No
other command SHALL contact any remote service.

#### Scenario: Offline document operations

- **WHEN** any command other than `upgrade` and `sync` runs without network access
- **THEN** it completes normally
