# Fullsend for mlflow-go

The configuration enables issue triage, issue-driven coding, PR review, and fixes
for `opendatahub-io/mlflow-go`. All agents use Pi. Coding, review, and fixes use
`openai/gpt-6.1-sol`; triage uses `openai/gpt-6-luna`. Review personas explicitly
select one of these OpenAI models, including the security triage and risk
assessment personas. There are no cross-provider Claude model aliases.
The repository's `AGENTS.md` supplies SDK development context.
The scope, model selection, and tradeoffs are recorded in
[ADR-0012](../docs/adr/0012-fullsend-automation.md).

The workflow uses the same Fullsend commit as the odh-dashboard reference:
`af8cd4756172ec105c3aa523515c9c7d51684e90`. It is based on the upstream scaffold
at that commit, with issue routing, the dashboard's review/fix event filter, and an explicit
repository guard. Fork workflows do not dispatch agents. The shim checks out no
PR code. Fullsend's reusable dispatcher performs command authorization.

## Activation

Repository files alone do not activate a working installation. A repository or
organization administrator must complete these steps before the first run:

1. Grant the `fullsend-ai-triage`, `fullsend-ai-review`, and `fullsend-ai-coder` GitHub Apps access to
   `opendatahub-io/mlflow-go`. The coder App supplies the fix role.
2. Configure the organization's Fullsend token mint for this repository. Use
   `fullsend github setup opendatahub-io/mlflow-go --agents triage,review,coder --runtime pi`
   with the organization's mint settings, reviewing its generated changes against
   this configuration. Setup defaults can differ from the organization's setup.
   Confirm `FULLSEND_MINT_URL` and `FULLSEND_GCP_REGION` as applicable to that
   installation. `FULLSEND_PROJECT_NUMBER` is a GitHub Projects V2 number for
   prioritization, so it can remain unset for review/fix. A restricted mint also
   requires repository enrollment by its operator.
3. Configure inference for this repository. For OpenAI workload identity, obtain
   a repository-scoped mapping and configure `FULLSEND_OPENAI_AUDIENCE`,
   `FULLSEND_OPENAI_IDENTITY_PROVIDER_ID`, and
   `FULLSEND_OPENAI_SERVICE_ACCOUNT_ID` using Fullsend's inference setup.
   Alternatively, provide the `FULLSEND_OPENAI_API_KEY` Actions secret.
   Confirm access to both configured models. The dashboard's GCP WIF provider
   is scoped to `odh-dashboard` and cannot authenticate `mlflow-go`.
4. Allow the pinned public reusable workflow under the organization's Actions
   policy. Review the configuration PR, then merge it to the default branch.
5. On a small PR, have an authorized maintainer post `/fs-review`. Confirm the
   routing job, App token mint, inference authentication, and posted review.
   Test `/fs-fix` on a same-repository branch and verify its code changes and CI.
   `/fs-fix-stop` adds `fullsend-no-fix`; it requires write access or PR authorship.
6. On a small GitHub issue, have an authorized maintainer post `/fs-triage`, then
   `/fs-code`. Verify the triage result, generated PR, and CI checks.

Review runs respond to non-draft PR opens, pushes, readiness events,
`ready-for-review` labels, and `/fs-review`. Fix runs route requested-change
reviews and `/fs-fix` through Fullsend's own authorization and eligibility checks.
There are no CodeRabbit host adapters in this configuration.
Issue opened, edited, and labeled events reach the reusable dispatcher, which
applies its triage/coding eligibility and authorization rules. Issue comments
support `/fs-triage` and `/fs-code`.

The separate [RHAI epic-to-code workflow](jira.md) polls non-Done RHAI epics
that Epic Creator generated for this repository and someone labeled
`ready-to-code`. Adding the label, or an authorized `/fs-code` comment, starts
coding. An epic created with the label still needs one of those explicit
actions. It needs Fullsend v0.54.0 or later and a Jira bot account; see
[jira.md](jira.md). GitHub issue coding remains enabled.

## Validation and maintenance

Run `fullsend agent list --fullsend-dir .fullsend` to load and inspect the
configuration, and `actionlint .github/workflows/fullsend.yaml` to validate the
workflow. These checks do not prove App installation or inference access.
When updating the Fullsend pin, compare the upstream scaffold and reusable
workflow contract and repeat a live review/fix smoke test.

References:

- [Dashboard configuration](https://github.com/opendatahub-io/odh-dashboard/blob/main/.fullsend/config.yaml)
- [Dashboard workflow](https://github.com/opendatahub-io/odh-dashboard/blob/main/.github/workflows/fullsend.yaml)
- [Fullsend GitHub setup](https://fullsend.sh/docs/guides/getting-started/configuring-github)
- [Fullsend inference setup](https://fullsend.sh/docs/guides/infrastructure/openai-workload-identity)
