# Spec Delta

## Purpose

Makes format compatibility with the Python implementation a tested property
rather than an intention: committed fixtures, an optional live differential
comparison, and an explicit inventory of every place where the two tools
deliberately differ.

## ADDED Requirements

### Requirement: Fixtures run without Python

The repository SHALL contain fixture vaults and expected outputs generated from
the Python implementation, committed to the repository, and the default test
suite SHALL compare against them without requiring Python or network access.

#### Scenario: Test suite without Python

- **WHEN** the test suite runs on a machine with no Python installed
- **THEN** the compatibility tests run and pass using the committed fixtures

### Requirement: Live differential comparison

A test suite gated behind an explicit build tag SHALL, when the Python
implementation is available, run both tools against equivalent vaults and
compare the resulting files, stdout and exit codes. It SHALL skip rather than
fail when Python or the Python package is absent.

#### Scenario: Both tools available

- **WHEN** the tagged suite runs with the Python tool installed
- **THEN** each comparable command's stdout, exit code and resulting vault files are compared and differences are reported

#### Scenario: Python absent

- **WHEN** the tagged suite runs without the Python tool
- **THEN** it reports the comparison as skipped and does not fail the build

### Requirement: Bidirectional interchange

A vault created and maintained by the Python implementation SHALL be fully
readable by specshelf, and a vault created and maintained by specshelf SHALL be
fully readable by the Python implementation, including documents added by either
tool after the other last ran.

#### Scenario: Python-created vault read by specshelf

- **WHEN** specshelf lists, searches and shows documents in a vault written only by the Python tool
- **THEN** every document is found with its metadata intact and no warnings are emitted

#### Scenario: Specshelf-created vault read by Python

- **WHEN** the Python tool lists, searches and shows documents in a vault written only by specshelf
- **THEN** every document is found with its metadata intact

#### Scenario: Mixed authorship

- **WHEN** both tools add documents to the same vault
- **THEN** each tool sees all documents and neither produces unreadable files for the other

### Requirement: Document identity parity

For a committed corpus of titles covering ASCII, accented Latin, German sharp S,
Nordic letters, CJK, Cyrillic, emoji, punctuation-only and over-length inputs,
the ids and file names produced by specshelf SHALL equal those produced by the
Python implementation, except where a documented divergence applies.

#### Scenario: Corpus parity

- **WHEN** the corpus is processed by specshelf
- **THEN** every produced id matches the recorded expected id for that title

### Requirement: Enumerated divergences

Every intentional difference from the Python implementation's observable
behaviour SHALL be recorded in a committed list, each entry stating the
behaviour, the reason, and the test that covers it. A behaviour change that is
not on the list and not byte-compatible SHALL be treated as a defect.

#### Scenario: Divergence list is complete

- **WHEN** the compatibility suite finds a difference in a compared command
- **THEN** the difference is either listed as an intentional divergence with its governing test, or the suite fails

#### Scenario: Documented divergence is enforced

- **WHEN** the test for a listed divergence runs
- **THEN** it asserts the documented behaviour exactly, so an accidental return to the upstream behaviour also fails

### Requirement: Cross-platform verification

The unit tests, the fixture-based compatibility tests and the command-level
tests SHALL run on Linux, macOS and Windows. The repository SHALL keep a
workflow that runs them on all three operating systems, enabled whenever the
repository is public.

#### Scenario: Windows runs the suite

- **WHEN** the test suite runs on Windows
- **THEN** it exercises path handling, file replacement and line endings

#### Scenario: Local verification while the workflow is disabled

- **WHEN** the suite is run locally while the workflow is disabled
- **THEN** the Linux run passes, and the packages compile and vet for macOS and Windows
