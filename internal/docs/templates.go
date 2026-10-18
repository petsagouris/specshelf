package docs

// SpecTemplate is the default body of a new spec, and the bytes written to
// templates/spec.md in a new vault.
const SpecTemplate = `# {title}

## Background

## Goals

## Non-Goals

## Repositories

## Contract

## Verification

## Decisions
`

// DecisionTemplate is the default body of a new decision, and the bytes written
// to templates/decision.md in a new vault.
const DecisionTemplate = `# {title}

## Context

## Decision

## Consequences

## Related Specs
`

// VaultReadme is the default vault README, written to README.md in a new vault.
const VaultReadme = "# Agent Spec Vault\n" +
	"\n" +
	"Global Markdown specs and decisions for AI coding agents.\n" +
	"\n" +
	"- `specs/`: durable specifications and cross-repo contracts\n" +
	"- `decisions/`: ADR-style decisions and rationale\n" +
	"- `index.jsonl`: generated search/index surface\n"
