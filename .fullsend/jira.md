# Jira epic-to-code automation

Fullsend polls Red Hat Jira every 30 minutes for this candidate set:

```jql
project = AISDLC AND issuetype = Epic AND labels = "fs-integration"
  AND statusCategory != Done ORDER BY updated DESC
```

AISDLC replaces RHAI because the integration account needs access to Jira's
project-role APIs, which require project administration. The initial smoke-test
epic was moved to AISDLC. Existing RHAI epics are no longer candidates.

The target is `opendatahub-io/mlflow-go`. There is no fixed issue key. Epics are
the coding work items; their children supply context, not separate dispatches.

## Triggering implementation

An AISDLC user with a project role named `Developers` or `Administrators` can:

- Add `fs-integration` to an existing epic to start coding.
- Comment `/fs-code` on a labeled epic to retry or request a new run.

The label is this repository's readiness signal; `ready-to-code` is not also
required. Creating an already-labeled epic does not start coding: post `/fs-code`
or remove and re-add the label with an authorized account. The pinned poller
synthesizes creation events from current labels and the reporter, which cannot
establish who added the readiness label. Ordinary comments and other label
changes do not trigger coding.
The human triggering work must be different from the account owning
`JIRA_TOKEN`: the poller classifies its own account as a bot even if it is a
personal account. Moving an already-labeled epic is not itself a coding trigger;
use a fresh authorized `/fs-code` comment or label addition after activation.
The poller excludes its own service account, and the local trigger requires a
human with `write` or `admin` authorization. Done epics and unlabeled epics are
excluded. The agent rechecks the live epic before editing code.

The code agent reads the epic and its direct children through the inherited
read-only Jira provider, implements the SDK scope, runs checks, and commits
locally. Fullsend's post-script pushes a branch and creates a GitHub PR against
`main` with the Jira link. Human review and CI still govern merging. An epic
that needs clarification or spans other repositories can produce an explanation
instead of a code change; the agent must not invent a complete implementation.

## Activation

1. Complete the GitHub App, token mint, and OpenAI setup in [README.md](README.md).
   This workflow forwards the same inference secrets and uses the existing
   repository-scoped OpenAI identity variables. The code agent remains Pi with
   `openai/gpt-6.1-sol`.
2. Set `JIRA_TOKEN` and `JIRA_USER_EMAIL` as Actions secrets on
   `opendatahub-io/mlflow-go`. Use a dedicated Jira service account. The workflow
   fixes the tenant to `https://redhat.atlassian.net`; no Jira URL variable is
   required.
3. Confirm the account can browse the AISDLC epics, comments, and child issues,
   including issue-security restrictions, and that the organization permits
   API-token access. Unlike a read-only connection check, polling writes Jira
   issue properties for locks and checkpoints. The pinned poller documents
   Developers and Administrators project roles for its service account so it
   can coordinate polls and inspect command authors' role membership.
4. Confirm AISDLC's role names and membership with its administrators. The poller
   maps `Developers` to `write` and `Administrators` to `admin`; custom role
   names do not imply write access. This authorization uses Jira membership,
   not membership in the GitHub repository.
5. Obtain maintainer approval for the protected workflow changes described in
   [implementation issue #45](https://github.com/opendatahub-io/mlflow-go/issues/45),
   then merge to `main`. Both workflows reject manual runs on other branches and
   fork repositories. Polling checks out the run’s exact `github.sha`.
6. Add the label to a small, well-scoped AISDLC epic, then run **Actions → Fullsend
   AISDLC epic coding → Run workflow**, or wait for the scheduled poll. Verify the
   poll dispatches `code`, the sandbox reads Jira, and a linked GitHub PR is
   created with the expected scope and passing CI. This live smoke test has not
   been run as part of this change.

Credentials stay in Actions secrets and Fullsend's credential providers. Do not
put token values in YAML, Jira comments, workflow inputs, or PR descriptions.
The PR may contain implementation details derived from the epic, so only opt in
work appropriate for this public repository.

## Polling behavior and recovery

Fullsend detects events; it does not start a fresh run for every labeled epic
on every poll. Its per-repository Jira checkpoints suppress repeated processing.
The pinned poller only backfills 24 hours. An old labeled epic with no recent
qualifying activity will not automatically dispatch when this workflow is first
enabled: an authorized user should post `/fs-code` or remove and re-add the
label. The same applies to missed events after an outage longer than 24 hours.

Workflow-level concurrency serializes manual and scheduled discovery/dispatch.
Each dispatch record launches its own **Fullsend AISDLC epic execution** workflow
run, with exactly one `code` matrix entry. This avoids the pinned action's
shared `fullsend-code` artifact name colliding across epics. The poll job needs
`actions: write` to launch these runs using its repository `GITHUB_TOKEN`.
All records are retained for seven days in the poll's `jira-dispatches-<attempt>`
artifact. If one launch request fails, remaining records are still attempted and
the poll fails visibly. An empty dispatch list launches no runs; malformed poll
output fails the workflow.

Execution accepts only a poll run ID, attempt, and record index. Before any Jira
or inference secrets are passed to the code job, it verifies that the artifact
came from this repository's main-branch poll workflow, then selects one record.
It also requires the poll's exact SHA to equal the execution run's `github.sha`,
which the reusable dispatcher uses for configuration. If `main` moves between
polling and execution, the execution run fails before starting the agent. Inspect
the poll artifact and post a fresh authorized `/fs-code` comment to retry at the
new revision. Every child execution run must be monitored separately: a green
poll means launches succeeded, not that coding succeeded.

Polling checkpoints are not an exactly-once delivery guarantee. If
dispatch or coding fails after checkpointing, inspect the run and retry with a
fresh `/fs-code` comment. Check for an existing PR first. Rerunning the parent
poll reads the advanced Jira checkpoints; it does not replay its saved records.
For a failed launch while `main` is still at the polling SHA, a maintainer may
manually run **Fullsend AISDLC epic execution** on `main` with the original poll
run ID, attempt, and failed record's zero-based index. Replay only failed records:
an accepted launch can still return a network error, so check child runs before
retrying. Parent reruns change the attempt and invalidate older artifact replays;
a fresh `/fs-code` is the recovery path if the revision or attempt has changed. Removing the label
prevents future discovery but does not cancel an already running job; cancel
that Actions run explicitly if needed.

The default poller considers up to 50 recently updated candidates and processes
up to five per cycle. Monitor backlog and rate limits if the opted-in set grows.
GitHub can disable scheduled workflows after repository inactivity; check that
the workflow remains enabled. Jira authentication, project permissions, and
API-policy failures require operator attention.

## Pins and validation

The CLI installer and dispatcher retain Fullsend commit
`af8cd4756172ec105c3aa523515c9c7d51684e90`. The code harness explicitly pins
`fullsend-ai/agents` commit `f4acd0350dc9445112f4a0a6f781c02a31a64636` with a
SHA-256 digest. This harness includes the Jira pre-script, read-only provider,
and external-work-item-aware GitHub post-script. Older integration documentation
describing the missing Jira pre/post pipeline does not describe this pinned
agent implementation. The existing GitHub code stage also uses this code harness;
the Jira overlay and local epic skill apply only to Jira source events.

Validate configuration with `fullsend agent list --fullsend-dir .fullsend`,
workflow syntax with `actionlint .github/workflows/fullsend*.yaml`, and
CEL routing with `fullsend dispatch --input-driver json --input-file <fixture>
--config-dir .fullsend`. Synthetic dispatch tests establish routing, not live
Jira connectivity, sandbox credential injection, or end-to-end PR creation.

Run the routing regression cases with the pinned CLI on your path:

```bash
python3 .fullsend/tests/test_jira_dispatch.py
python3 .fullsend/tests/test_jira_workflows.py
```

Set `FULLSEND_BIN` to use an explicit binary path. The cases cover activation,
retry, ordinary comments, label removal, authorization, and repo/project/tenant
boundaries, plus two-epic fan-out, partial launch failure, provenance, and revision
mismatch rejection. The CLI may fetch the pinned public harness into
`.fullsend-cache/`; do not commit that generated cache.

References:

- [Reference poller](https://github.com/ralphbean/jira-code-test/blob/main/.github/workflows/fullsend-poll-jira.yaml)
- [Pinned Jira-capable code harness](https://github.com/fullsend-ai/agents/blob/f4acd0350dc9445112f4a0a6f781c02a31a64636/harness/code.yaml)
- [Pinned Jira poller guide and operational limits](https://github.com/fullsend-ai/fullsend/blob/af8cd4756172ec105c3aa523515c9c7d51684e90/docs/guides/user/jira-integration.md)
