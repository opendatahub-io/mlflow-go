"""Fan out trusted poll records and validate their provenance before execution.

The fan-out works around https://github.com/fullsend-ai/fullsend/issues/8961
(same-agent records in one matrix collide on the fullsend-<agent> artifact).
Remove this script and fullsend-code-jira.yaml once that is fixed.
"""

import json
import os
from pathlib import Path
import subprocess
import sys

REPOSITORY = "opendatahub-io/mlflow-go"
POLL_WORKFLOW = ".github/workflows/fullsend-poll-jira.yaml"
CODE_WORKFLOW = "fullsend-code-jira.yaml"


def launch(records, run_id, attempt, api):
    """Attempt every checkpointed record, even if an earlier API call fails."""
    failures = []
    for index, _ in enumerate(records):
        payload = {
            "ref": "main",
            "inputs": {
                "poll_run_id": str(run_id),
                "poll_attempt": str(attempt),
                "record_index": str(index),
            },
        }
        try:
            api(
                f"repos/{REPOSITORY}/actions/workflows/{CODE_WORKFLOW}/dispatches",
                payload,
            )
        except subprocess.CalledProcessError:
            failures.append(index)
    if failures:
        raise RuntimeError(
            f"Failed to dispatch records {failures}; inspect poll artifact and retry with /fs-code"
        )


def validate_run(run, sha, attempt):
    """Only artifacts produced by the main-branch poller may reach credentials."""
    if (
        run["repository"]["full_name"] != REPOSITORY
        or run["head_repository"]["full_name"] != REPOSITORY
        or run["path"] != POLL_WORKFLOW
        or run["head_branch"] != "main"
        or run["event"] not in ("schedule", "workflow_dispatch")
        or run["head_sha"] != sha
        or run["run_attempt"] != int(attempt)
    ):
        raise ValueError(
            "Untrusted poll run or revision mismatch; inspect run and retry with /fs-code"
        )


def select_record(records, index):
    index = int(index)
    if index < 0 or index >= len(records):
        raise ValueError("Record index is outside the poll artifact")
    record = records[index]
    if record["agent"] != "code" or record["source_repo"] != REPOSITORY:
        raise ValueError("Unexpected agent or target repository")
    event = json.loads(record["event_payload"])["_normalized_event"]
    if event["source"]["system"] != "jira" or event["repo"] != REPOSITORY:
        raise ValueError("Unexpected event source")
    return {"include": [record]}


def api(endpoint, payload=None):
    args = ["gh", "api", endpoint]
    if payload is not None:
        args.extend(["--method", "POST", "--input", "-"])
    result = subprocess.run(
        args,
        input=json.dumps(payload) if payload is not None else None,
        text=True,
        capture_output=True,
        check=True,
    )
    return json.loads(result.stdout) if result.stdout.strip() else None


def main():
    mode = sys.argv[1]
    if mode == "validate":
        run_id = int(os.environ["POLL_RUN_ID"])
        attempt = int(os.environ["POLL_ATTEMPT"])
        run = api(f"repos/{REPOSITORY}/actions/runs/{run_id}")
        validate_run(run, os.environ["GITHUB_SHA"], attempt)
    else:
        records = json.loads(Path("dispatches.json").read_text())
        if not isinstance(records, list):
            raise ValueError("Poll output must be an array")
        if mode == "launch":
            launch(
                records,
                os.environ["GITHUB_RUN_ID"],
                os.environ["GITHUB_RUN_ATTEMPT"],
                api,
            )
        elif mode == "select":
            matrix = select_record(records, os.environ["RECORD_INDEX"])
            with open(os.environ["GITHUB_OUTPUT"], "a") as output:
                output.write(
                    "matrix=" + json.dumps(matrix, separators=(",", ":")) + "\n"
                )
        else:
            raise ValueError(f"Unknown mode: {mode}")


if __name__ == "__main__":
    main()
