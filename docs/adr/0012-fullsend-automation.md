# ADR-0012: Fullsend Automation with OpenAI Models

**Status**: Proposed

**Date**: 2026-10-23

**Authors**: @ederign

## Context

The MLflow Go SDK needs a consistent way to triage GitHub issues, implement
scoped changes, review pull requests, and address review findings. These agents
need repository context for public API compatibility, authentication, workspace
isolation, artifact storage, and generated protobuf code.

ODH Dashboard already uses Fullsend's per-repository GitHub Actions dispatcher
and explicit OpenAI model selection for reviews. RFE Creator uses the same
platform with a mix of Claude, Gemini, and OpenAI models. For mlflow-go, the
chosen scope is GitHub issue and PR automation with OpenAI for both coding and
reviews, including triage and fixes.

## Decision

Adopt Fullsend in per-repository mode. Store agent configuration in
`.fullsend/config.yaml`, development guidance in `AGENTS.md`, and the event
dispatcher in `.github/workflows/fullsend.yaml`.

Enable the triage, coder, and review roles. The coder role serves both
issue-to-PR coding and PR fixes. Use the Pi runtime for every agent:

| Agent | Model |
|-------|-------|
| Code, review, fix | `openai/gpt-6.1-sol` |
| Triage | `openai/gpt-6-luna` |
| Review subagents | Explicit Sol or Luna selection per persona |

Configure review persona models explicitly rather than remapping Claude's
`opus`, `sonnet`, or `haiku` aliases to another provider. Correctness, security,
challenger, intent coherence, and cross-repository contract reviews use Sol.
Documentation, style, risk assessment, and security triage use Luna. This makes
the intended provider visible in the configuration.

Pin the reusable workflow to commit
`af8cd4756172ec105c3aa523515c9c7d51684e90`, following the dashboard reference.
Forward issue events and supported issue/PR commands to Fullsend's dispatcher,
which applies its authorization and eligibility checks. Restrict the local
workflow to `opendatahub-io/mlflow-go` and preserve `/fs-fix-stop`. The shim does
not check out or execute PR code; agent execution belongs to the reusable
workflow. Changes to the pin require review of that workflow's contract.

Keep GitHub and inference authentication separate. Grant the Fullsend triage,
coder, and review Apps access to the repository and use the hosted token mint.
Supply OpenAI inference access through the `FULLSEND_OPENAI_API_KEY` Actions
secret or a repository-scoped OpenAI workload identity mapping. The selected
models do not require Vertex or Anthropic inference credentials. Credentials
remain outside version control.

Use Fullsend's built-in agents and review skills. Dashboard-specific Jira,
CodeRabbit, and custom review integrations are outside this change. GitHub issue
support does not imply Jira issue support. Automatic merging is not configured.

## Alternatives Considered

### Alternative 1: Copy the dashboard configuration in full

The dashboard's custom review harness includes project-specific review context
and Jira/CodeRabbit adapters, while fixes retain Claude. Copying it would add
integration and credential requirements beyond the selected OpenAI-only scope.
Reuse the dispatch pattern and explicit model selection without those adapters.

### Alternative 2: Use RFE Creator's mixed-provider configuration

This would require maintaining Vertex access alongside OpenAI access and would
bring agent settings intended for a different repository. Use one inference
provider for this installation; model/provider changes can be evaluated later.

### Alternative 3: Limit Fullsend to PR reviews

Review-only automation would leave issue triage, implementation, and review
fixes outside the workflow. Enable those roles to support the requested
issue-to-PR flow, subject to Fullsend's authorization checks and repository CI.

## Consequences

### Positive

- Issue coding and reviews share one configuration and inference provider.
- Explicit model assignments make provider dependencies reviewable.
- Existing Fullsend routing and authorization avoid maintaining a custom agent
  dispatcher in this repository.
- Repository guidance records SDK constraints that apply across agent tasks.

### Negative

- Agent runs consume paid inference and GitHub Actions resources.
- Activation depends on App installation, mint access, model availability, and
  inference credentials; committing the files alone is insufficient.
- Pinned Fullsend workflows and built-in agent behavior require maintenance.
- Agent-generated changes still require review and CI validation; static
  configuration checks do not establish the quality of their output.

### Neutral

- SDK runtime behavior and public APIs are unchanged.
- Workload identity and API-key authentication are supported deployment choices;
  the PR does not provision either credential path.
- Before considering activation complete, run issue triage, issue coding, PR
  review, and PR fix smoke tests as described in the setup guide. These live
  tests have not been completed as part of this proposal.

## Follow-up: RHAI epic coding

Extend issue-driven coding to non-Done RHAI epics labeled `fs-integration` using
a separate scheduled Jira poller. The label is the readiness signal; authorized
label additions and `/fs-code` comments dispatch the code agent. Synthetic
creation events are excluded because the pinned poller reports current labels
and the reporter rather than the original readiness actor. Each record starts
a separate workflow run to avoid shared artifact names; execution requires the
exact polling SHA and a validated main-branch poll artifact. Pin the Jira-capable
code harness, retain Pi/OpenAI and GitHub dispatch,
and restrict generated PRs to `main`. Epic children provide implementation
context rather than independent dispatches. Jira project-role authorization and
checkpoint writes are additional activation requirements. See the
[Jira setup and operations guide](../../.fullsend/jira.md) for pins, credentials,
retry behavior, and the required live smoke test.

## References

- [Repository setup and activation guide](../../.fullsend/README.md)
- [Implementation PR #41](https://github.com/opendatahub-io/mlflow-go/pull/41)
- [ODH Dashboard configuration](https://github.com/opendatahub-io/odh-dashboard/blob/main/.fullsend/config.yaml)
- [RFE Creator configuration](https://github.com/fullsend-ai/rfe-creator/blob/main/.fullsend/config.yaml)
- [Pinned Fullsend dispatcher](https://github.com/fullsend-ai/fullsend/blob/af8cd4756172ec105c3aa523515c9c7d51684e90/.github/workflows/reusable-dispatch.yml)

## Follow-up: move Jira epic coding to AISDLC

Target AISDLC instead of RHAI. The pinned Jira poller requires project-role
lookup permissions; the integration account can administer AISDLC but cannot
administer RHAI. Update JQL, normalized-event project and URL guards, and the
epic skill's live validation and child query together. Keep the opt-in label,
human authorization gates, pinned harness, and half-hour schedule unchanged.

The token owner cannot trigger its own coding events: Fullsend classifies that
account as a bot. Use a separate authorized human for label additions or
`/fs-code` comments, including after moving an already-labeled epic.
