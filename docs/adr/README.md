# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the MLflow Go SDK.

## What is an ADR?

An ADR is a document that captures an important architectural decision made along with its context and consequences.

## ADR Index

| ID | Title | Status | Date |
|----|-------|--------|------|
| [0001](0001-authentication-pattern.md) | Authentication Pattern | Accepted | 2026-01-14 |
| [0002](0002-error-type-design.md) | Error Type Design | Accepted | 2026-01-14 |
| [0003](0003-resilience-strategy.md) | Resilience Strategy | Accepted | 2026-01-14 |

## Creating a New ADR

1. Copy `_template.md` to `NNNN-title.md` (next sequential number)
2. Fill in all sections
3. Submit PR with the ADR and any related code changes
4. Update the index above

## ADR Statuses

- **Proposed**: Under discussion
- **Accepted**: Decision made and in effect
- **Deprecated**: No longer applies (superseded or context changed)
- **Superseded**: Replaced by another ADR (link to replacement)
