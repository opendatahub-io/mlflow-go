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
        "key": "RHAI-123",
        "url": "https://redhat.atlassian.net/browse/RHAI-123",
    },
    "transition": {
        "kind": "label_changed",
        "label": {"name": "ready-to-code", "action": "added"},
    },
    "actor": {
        "id": "human-test",
        "kind": "human",
        "role": "write",
        "is_entity_author": False,
    },
    "state": {"labels": ["ready-to-code"]},
    "source": {"system": "jira", "raw_type": "changelog", "raw_action": "updated"},
}
cases = []


def add(name, expected, **changes):
    event = copy.deepcopy(base)
    for key, val in changes.items():
        event[key] = val
    cases.append((name, event, expected))


# These cases exercise the default code harness trigger with this repository's
# configuration, using a Fullsend CLI that includes fullsend#8892. Project,
# tenant and repository are not checked here: the poll's --jira-project,
# --jira-url, JQL and --target-repo fix them. Bot events never reach dispatch:
# the poller drops them first, so a label added by automation that Fullsend
# classifies as a bot does not start coding.
add("label_added", 1)
add("admin_actor", 1, actor=base["actor"] | {"role": "admin"})
# A synthetic opened event has current labels and the reporter, even when
# another user added readiness later. It must never start coding.
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
    "bot_fs_code",
    0,
    transition={
        "kind": "comment_added",
        "comment": {"body": "/fs-code", "command": "/fs-code"},
    },
    actor=base["actor"] | {"kind": "bot"},
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
        "label": {"name": "ready-to-code", "action": "removed"},
    },
)
add(
    "other_label",
    0,
    transition={
        "kind": "label_changed",
        "label": {"name": "fs-integration", "action": "added"},
    },
)
# With the jira provider's default labels: jira_editors (fullsend#8892), a
# label addition is authorized by Jira's Edit Issues permission, so the
# actor's resolved role does not matter. /fs-code still needs write.
add("triage_actor_adds_label", 1, actor=base["actor"] | {"role": "triage"})
add("read_only_actor_adds_label", 1, actor=base["actor"] | {"role": "read"})
add(
    "read_only_fs_code",
    0,
    transition={
        "kind": "comment_added",
        "comment": {"body": "/fs-code", "command": "/fs-code"},
    },
    actor=base["actor"] | {"role": "read"},
)
add(
    "triage_fs_code",
    0,
    transition={
        "kind": "comment_added",
        "comment": {"body": "/fs-code", "command": "/fs-code"},
    },
    actor=base["actor"] | {"role": "triage"},
)
add("github_event", 0, source=base["source"] | {"system": "github"})
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
