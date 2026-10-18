# git-sync Specification

## Purpose
Keeps a vault under version control by using the git installation and
credentials the user already has, so a shared or backed-up vault can be
committed and pushed without the tool managing keys, remotes or history itself.

## Requirements

### Requirement: Syncing uses the user's git installation

`specshelf sync` SHALL perform its work by invoking the `git` executable with
the vault as its working directory, so that the user's own configuration,
credential helpers, SSH agent and remote settings apply. The tool SHALL NOT
implement its own git object handling.

#### Scenario: Commit and push with an existing remote

- **WHEN** `sync` runs in a vault that is a git repository with a configured remote
- **THEN** the pending changes are committed and the branch is pushed to that remote

#### Scenario: First sync in a plain directory

- **WHEN** `sync` runs in a vault that is not yet a git repository
- **THEN** a repository is initialised in the vault root and the current documents are committed

### Requirement: Nothing-to-commit is a success

When the vault has no pending changes, `sync` SHALL report that there is nothing
to commit and exit successfully, rather than surfacing the underlying command's
non-zero status as an error.

#### Scenario: Repeated sync

- **WHEN** `sync` runs twice in a row with no intervening changes
- **THEN** the second run reports nothing to commit and exits 0

### Requirement: Push is optional and never forced

Pushing SHALL be controlled by `--push` and `--no-push` and by the configured
default. The tool SHALL NOT force-push, SHALL NOT rewrite history, and SHALL
report a push rejection as an error without altering local or remote history.

#### Scenario: Local-only sync

- **WHEN** `sync --no-push` runs in a vault with a remote
- **THEN** a commit is created locally and no network operation is attempted

#### Scenario: Rejected push

- **WHEN** the remote rejects the push because it has diverged
- **THEN** the command exits non-zero, reports the rejection, and leaves local history unchanged

### Requirement: Sync is never implicit

No command other than `sync` SHALL commit, stage or push anything. Creating,
editing or removing documents SHALL NOT produce git activity on its own.

#### Scenario: Creating a document does not commit

- **WHEN** `specshelf new` runs inside a git-tracked vault
- **THEN** the working tree has uncommitted changes and no new commit exists

### Requirement: Missing git is reported clearly

When the `git` executable is not available, `sync` SHALL fail with a message
explaining that git is required for this command only, and no other command
SHALL require git to be installed.

#### Scenario: git not installed

- **WHEN** `sync` runs on a machine without `git` on the PATH
- **THEN** the command exits non-zero with a message naming the missing executable

#### Scenario: Other commands are unaffected

- **WHEN** any non-sync command runs on a machine without `git`
- **THEN** it behaves normally
