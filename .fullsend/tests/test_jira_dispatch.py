"""Exercise Jira routing with the pinned Fullsend CLI; no Jira API calls."""

import copy
import json
import os
import pathlib
import subprocess
import tempfile
import runpy

os.chdir(pathlib.Path(__file__).resolve().parents[2])
binary = os.environ.get("FULLSEND_BIN", "fullsend")
select_record = runpy.run_path(".fullsend/scripts/jira_dispatch.py")["select_record"]
base = {
    "repo": "opendatahub-io/mlflow-go",
    "entity": {
        "kind": "work_item",
        "id": 12345,
        "key": "AISDLC-123",
        "url": "https://redhat.atlassian.net/browse/AISDLC-123",
    },
    "transition": {
        "kind": "label_changed",
        "label": {"name": "fs-integration", "action": "added"},
    },
    "actor": {
        "id": "human-test",
        "kind": "human",
        "role": "write",
        "is_entity_author": False,
    },
    "state": {"labels": ["fs-integration"]},
    "source": {"system": "jira", "raw_type": "changelog", "raw_action": "updated"},
}
cases = []


def add(name, expected, **changes):
    event = copy.deepcopy(base)
    for key, val in changes.items():
        event[key] = val
    cases.append((name, event, expected))


add("label_added", 1)
add("admin_actor", 1, actor=base["actor"] | {"role": "admin"})
# A synthetic opened event has current labels and the authorized reporter,
# even when another user added readiness later. It must never authorize code.
add("created_with_label", 0, transition={"kind": "opened"})
add(
    "reporter_cannot_authorize_later_label",
    0,
    transition={"kind": "opened"},
    actor=base["actor"] | {"is_entity_author": True},
)
add(
    "explicit_retry",
    1,
    transition={
        "kind": "comment_added",
        "comment": {"body": "/fs-code", "command": "/fs-code"},
    },
)
add(
    "ordinary_comment",
    0,
    transition={"kind": "comment_added", "comment": {"body": "hello"}},
)
add(
    "label_removed",
    0,
    transition={
        "kind": "label_changed",
        "label": {"name": "fs-integration", "action": "removed"},
    },
)
add(
    "other_label",
    0,
    transition={
        "kind": "label_changed",
        "label": {"name": "ready-to-code", "action": "added"},
    },
)
add("missing_opt_in", 0, state={"labels": []})
add("read_only_actor", 0, actor=base["actor"] | {"role": "read"})
add("bot", 0, actor=base["actor"] | {"kind": "bot"})
add("other_repo", 0, repo="other/repo")
add("github_event", 0, source=base["source"] | {"system": "github"})
add(
    "previous_rhai_project",
    0,
    entity=base["entity"] | {
        "key": "RHAI-123",
        "url": "https://redhat.atlassian.net/browse/RHAI-123",
    },
)
add(
    "previous_rhai_url",
    0,
    entity=base["entity"] | {"url": "https://redhat.atlassian.net/browse/RHAI-123"},
)
add("other_project", 0, entity=base["entity"] | {"key": "OTHER-123"})
add(
    "other_tenant",
    0,
    entity=base["entity"] | {"url": "https://other.atlassian.net/browse/AISDLC-123"},
)
with tempfile.NamedTemporaryFile(mode="w+", suffix=".json") as f:
    for name, event, expected in cases:
        f.seek(0)
        f.truncate()
        json.dump(event, f)
        f.flush()
        p = subprocess.run(
            [
                binary,
                "dispatch",
                "--input-driver",
                "json",
                "--input-file",
                f.name,
                "--config-dir",
                ".fullsend",
                "--output-driver",
                "json",
            ],
            text=True,
            capture_output=True,
        )
        if p.returncode:
            raise RuntimeError(name + ": " + p.stderr)
        result = json.loads(p.stdout)
        assert len(result or []) == expected, (name, result, p.stderr)
        if expected:
            assert result[0]["agent"] == "code", result
            assert select_record(result, 0) == {"include": [result[0]]}
        print(name + ": passed", flush=True)
