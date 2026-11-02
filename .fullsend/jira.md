# Jira epic-to-code automation

Fullsend polls Red Hat Jira every 30 minutes for this candidate set:

```jql
project = RHAI
  AND issuetype = Epic
  AND labels = "epic-creator-auto-created"
  AND labels in ("epic-creator-ai-impl-high", "epic-creator-ai-impl-medium")
  AND labels = "ready-to-code"
  AND environment ~ "\"https://github.com/opendatahub-io/mlflow-go\""
  AND statusCategory != Done
ORDER BY updated DESC
```

These are epics that Epic Creator generated for AI implementation, that someone
has marked ready, and whose Environment field names this repository.
Environment is free text, so the match is a phrase match, not equality. The
selectors are provisional until the epic tooling settles on how it marks the
target repository and readiness.

The target is `opendatahub-io/mlflow-go`. There is no fixed issue key. Epics are
the coding work items; their children supply context, not separate dispatches.

This configuration needs Fullsend v0.54.0 or later, the first expected release
with [fullsend#8892](https://github.com/fullsend-ai/fullsend/pull/8892). Earlier
releases resolve every Jira event through RHAI's project roles, which requires
the Administer Projects permission for the polling account.

## Triggering implementation

- **Add `ready-to-code`** to a matching epic to start coding. Anyone who can
  edit the epic can do this: the `jira` authorization entry in `config.yaml`
  keeps Fullsend's default `labels: jira_editors` rule, which relies on Jira's
  Edit Issues permission.
- **Comment `/fs-code`** on a labeled epic to retry or to request a new run,
  optionally with instructions. The commenter must be in a group listed under
  `write` in the `jira` authorization entry. This repository lists
  `jira-software-users-redhat`, which includes every Red Hat Jira user. A team
  that wants a narrower audience lists a smaller group.

Because the candidate query requires `ready-to-code`, `/fs-code` on an epic
without the label does nothing. Removing the label stops discovery but does
not cancel a run that has already started.

The code agent itself has no trigger in this repository: `harness/code.yaml`
inherits the default code harness trigger (a `ready-to-code` label added, or a
`/fs-code` comment from a human). The file exists so that the poller registers
the `code` agent.

Some actions do not start coding:

- **Creating an epic that already has the label.** The poller reports a new
  epic as an `opened` event with its current labels and the reporter, which
  cannot show who added readiness. Post `/fs-code`, or remove and re-add the
  label.
- **Labels or comments from accounts Fullsend treats as bots.** The poller drops
  these events before authorization, with no log line. That covers the account
  that owns `JIRA_TOKEN`, accounts Jira reports as apps, and accounts whose
  display name ends in `bot` or `automation` or starts with `Automation for`
  (such as Jira's own automation rules). A person must add the label or post
  the comment.
- Ordinary comments, other label changes, and label removals.

The code agent reads the epic and its direct children through the inherited
read-only Jira provider, implements the SDK scope, runs checks, and commits
locally. Fullsend's post-script pushes a branch, creates a GitHub PR against
`main` with the Jira link, and reports the result on the epic. Human review and
CI still govern merging. An epic that needs clarification or spans other
repositories can produce an explanation instead of a code change; the agent must
not invent a complete implementation. The `rhai-epics` skill adds these rules
and has the agent recheck the live epic before editing code.

## Activation

1. Complete the GitHub App, token mint, and OpenAI setup in [README.md](README.md).
   This workflow forwards the same inference secrets and uses the existing
   repository-scoped OpenAI identity variables. The code agent remains Pi with
   `openai/gpt-6.1-sol`.
2. Set `JIRA_TOKEN` and `JIRA_USER_EMAIL` as Actions secrets on
   `opendatahub-io/mlflow-go` for a dedicated Jira bot account, not a personal
   account. The workflow fixes the tenant to `https://redhat.atlassian.net`; no
   Jira URL variable is required.
3. Confirm the account has these RHAI permissions: Browse Projects, Edit Issues
   (polling writes lock and checkpoint issue properties), Add Comments, and Edit
   Own Comments; and the global Browse users and groups permission, which the
   group list needs. It does not need Administer Projects.
4. Update the `TODO` pins in the workflows and `harness/code.yaml` to the
   v0.54.0 release commit SHAs and digest, then merge to `main` with maintainer
   approval. Both workflows reject manual runs on other branches and fork
   repositories. Polling checks out the run's exact `github.sha`.
5. Add `ready-to-code` to a small, well-scoped matching RHAI epic, then run
   **Actions, Fullsend RHAI epic coding, Run workflow**, or wait for the
   scheduled poll. Verify the poll dispatches `code`, the sandbox reads Jira, a
   linked GitHub PR is created with the expected scope and passing CI, and the
   result appears on the epic. This live smoke test has not been run as part of
   this change.

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
Each dispatch record launches its own **Fullsend RHAI epic execution** workflow
run, with exactly one `code` matrix entry. This avoids the pinned action's
shared `fullsend-code` artifact name colliding across epics
([fullsend#8961](https://github.com/fullsend-ai/fullsend/issues/8961)); remove
the fan-out once that is fixed. The poll job needs
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
manually run **Fullsend RHAI epic execution** on `main` with the original poll
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

The CLI installer and the execution workflow's dispatcher pin Fullsend v0.54.0,
and the code harness pins `fullsend-ai/agents` v0.54.0. Until those releases
exist, the pins are placeholders marked `TODO`. The GitHub workflow
(`fullsend.yaml`) keeps its own Fullsend pin, but GitHub code runs also use
`harness/code.yaml`, so they move to the same agents release. The Jira overlay
and the `rhai-epics` skill apply only to Jira source events.

Validate configuration with `fullsend agent list --fullsend-dir .fullsend`,
workflow syntax with `actionlint .github/workflows/fullsend*.yaml`, and
CEL routing with `fullsend dispatch --input-driver json --input-file <fixture>
--config-dir .fullsend`. Synthetic dispatch tests establish routing, not live
Jira connectivity, sandbox credential injection, or end-to-end PR creation.

Run the routing regression cases with a Fullsend CLI that includes
fullsend#8892 on your path:

```bash
python3 .fullsend/tests/test_jira_dispatch.py
python3 .fullsend/tests/test_jira_workflows.py
```

Set `FULLSEND_BIN` to use an explicit binary path. The cases cover activation,
retry, ordinary comments, label removal, label and `/fs-code` authorization,
plus two-epic fan-out, partial launch failure, provenance, and revision
mismatch rejection. The CLI may fetch the pinned public harness into
`.fullsend-cache/`; do not commit that generated cache.

References:

- [Reference poller](https://github.com/ralphbean/jira-code-test/blob/main/.github/workflows/fullsend-poll-jira.yaml)
- [Default code harness](https://github.com/fullsend-ai/agents/blob/main/harness/code.yaml)
- [Fullsend Jira integration guide](https://github.com/fullsend-ai/fullsend/blob/main/docs/guides/user/jira-integration.md)
- [fullsend#8892: Jira authorization without Administer Projects](https://github.com/fullsend-ai/fullsend/pull/8892)
