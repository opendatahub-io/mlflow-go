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
| [0004](0004-prompt-type-abstraction.md) | Prompt Type Abstraction | Accepted | 2026-01-15 |
| [0005](0005-flat-package-structure.md) | Multi-Package Structure | Accepted | 2026-01-15 |
| [0006](0006-protobuf-strategy.md) | Protobuf Strategy | Accepted | 2026-01-16 |
| [0007](0007-python-sdk-naming-alignment.md) | Python SDK Naming Alignment | Accepted | 2026-01-23 |
| [0008](0008-oss-only-target-platform.md) | OSS-Only Target Platform | Accepted | 2026-01-14 |
| [0009](0009-experiment-tracking.md) | Experiment Tracking Client | Accepted | 2026-02-25 |

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
