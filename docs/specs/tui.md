# tui Specification

## Purpose
Provides an interactive full-screen way to browse, search, read and change a
vault, with exactly the same storage guarantees and on-disk results as the
command-line interface.

## Requirements

### Requirement: Interactive session preconditions

`specshelf ui` SHALL start an interactive full-screen session only when stdout is
a terminal. When it is not, the command SHALL fail with a message directing the
user to the non-interactive commands, exit non-zero, and not modify the vault.

#### Scenario: Redirected output

- **WHEN** `specshelf ui` runs with stdout redirected to a file
- **THEN** it exits non-zero without starting a session

### Requirement: Browsing views

The session SHALL provide a list view showing every matching document with its
kind, title, status, update date and tags, a detail view showing one document's
frontmatter and body, and a creation prompt that captures a title and a kind.
Moving the selection, opening the detail view, returning to the list and
creating a document SHALL be possible from the keyboard, and a help view SHALL
show the available keys.

#### Scenario: Opening one document

- **WHEN** the user selects a document in the list view and confirms
- **THEN** the detail view shows that document's frontmatter values and body

#### Scenario: Creating from the session

- **WHEN** the user submits a title in the creation prompt
- **THEN** a document is created with that title and appears in the list view

### Requirement: Filtering without disk scans

The list view SHALL filter the loaded documents in memory as the user types, and
SHALL offer filtering by kind, status, repo and tag. Typing in the filter SHALL
NOT re-read the vault from disk, and an explicit reload SHALL be the only
operation that does.

#### Scenario: Filtering is immediate

- **WHEN** the user types a query in the filter
- **THEN** the visible list narrows to matching titles, paths and metadata without a filesystem scan

#### Scenario: External change appears after reload

- **WHEN** a document is created by another process and the user reloads
- **THEN** the new document appears in the list view

### Requirement: Every change goes through the vault operations

The interactive session SHALL create, edit, change status and delete documents
only by calling the same storage operations the command-line interface uses. It
SHALL NOT read or write vault files, frontmatter or the index directly.

#### Scenario: Session output matches the command line

- **WHEN** a document is created in the session and an identical one is created with `specshelf new`
- **THEN** the two files have identical frontmatter field order and body content, differing only in id and date as those would differ for any two creations

#### Scenario: Index stays consistent

- **WHEN** a document is deleted in the session
- **THEN** `index.jsonl` no longer contains it

### Requirement: Editing uses the user's editor

Editing a document in the session SHALL hand the file to the editor named by
`$VISUAL`, then `$EDITOR`, then the configured editor, and SHALL return to the
session when the editor exits. After returning, the document SHALL be re-read
and the index refreshed, and `updated` SHALL be changed only if the content
changed.

#### Scenario: Editor makes no change

- **WHEN** editing is started and the editor exits without saving
- **THEN** the document's bytes and `updated` field are unchanged and the detail view shows the unchanged content

#### Scenario: Editor changes the body

- **WHEN** the editor saves new content
- **THEN** the detail view shows the new content and the index reflects the updated document

### Requirement: Destructive actions are confirmed

Deleting a document SHALL require an explicit confirmation inside the session and
SHALL be cancellable. The session SHALL never delete a document as a side effect
of another action.

#### Scenario: Cancelled deletion

- **WHEN** the user requests deletion and declines the confirmation
- **THEN** the document still exists

### Requirement: The terminal is restored

The session SHALL leave the terminal in its original state when it exits for any
reason, including on quit, on error, and when the session ends because the
terminal window was resized or the input stream closed.

#### Scenario: Quitting restores the terminal

- **WHEN** the session exits after the user quits
- **THEN** the alternate screen is left, the cursor is visible, and normal line editing works in the invoking shell
