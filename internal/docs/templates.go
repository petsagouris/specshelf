package docs

// SpecTemplate is the default body of a new spec. Its bytes match the Python
// reference vault's templates/spec.md exactly.
const SpecTemplate = `# {title}

## Background

## Goals

## Non-Goals

## Repositories

## Contract

## Verification

## Decisions
`

// DecisionTemplate is the default body of a new decision. Its bytes match the
// Python reference vault's templates/decision.md exactly.
const DecisionTemplate = `# {title}

## Context

## Decision

## Consequences

## Related Specs
`

// VaultReadme is the default vault README. Its bytes match the Python reference
// vault's README.md exactly.
const VaultReadme = "# Agent Spec Vault\n" +
	"\n" +
	"Global Markdown specs and decisions for AI coding agents.\n" +
	"\n" +
	"- `specs/`: durable specifications and cross-repo contracts\n" +
	"- `decisions/`: ADR-style decisions and rationale\n" +
	"- `index.jsonl`: generated search/index surface\n"
