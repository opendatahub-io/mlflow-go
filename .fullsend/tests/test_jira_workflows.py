"""Check fan-out, provenance and the one-record execution boundary without APIs."""

import copy
import importlib.util
import json
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location(
    "jira_dispatch", ROOT / ".fullsend/scripts/jira_dispatch.py"
)
dispatch = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(dispatch)
SHA = "a" * 40


class JiraWorkflowTests(unittest.TestCase):
    def setUp(self):
        self.run = {
            "repository": {"full_name": dispatch.REPOSITORY},
            "head_repository": {"full_name": dispatch.REPOSITORY},
            "path": dispatch.POLL_WORKFLOW,
            "head_branch": "main",
            "event": "schedule",
            "head_sha": SHA,
            "run_attempt": 1,
        }
        self.records = [
            {
                "agent": "code",
                "source_repo": dispatch.REPOSITORY,
                "event_payload": json.dumps(
                    {
                        "_normalized_event": {
                            "source": {"system": "jira"},
                            "repo": dispatch.REPOSITORY,
                            "entity": {"key": f"AISDLC-{number}"},
                        }
                    }
                ),
            }
            for number in (123, 456)
        ]

    def test_two_epics_get_separate_runs_with_original_records(self):
        calls = []
        dispatch.launch(
            self.records,
            42,
            1,
            lambda endpoint, payload: calls.append((endpoint, payload)),
        )
        self.assertEqual(len(calls), 2)
        for index, (endpoint, payload) in enumerate(calls):
            self.assertTrue(endpoint.endswith("/fullsend-code-jira.yaml/dispatches"))
            self.assertEqual(payload["ref"], "main")
            self.assertEqual(
                payload["inputs"],
                {
                    "poll_run_id": "42",
                    "poll_attempt": "1",
                    "record_index": str(index),
                },
            )
            matrix = dispatch.select_record(self.records, index)
            self.assertEqual(matrix, {"include": [self.records[index]]})
            self.assertEqual(matrix["include"][0]["agent"], "code")

    def test_failed_dispatch_does_not_drop_later_checkpointed_records(self):
        attempted = []

        def api(endpoint, payload):
            index = payload["inputs"]["record_index"]
            attempted.append(index)
            if index == "0":
                raise subprocess.CalledProcessError(1, "gh")

        with self.assertRaisesRegex(RuntimeError, "records \\[0\\]"):
            dispatch.launch(self.records, 42, 1, api)
        self.assertEqual(attempted, ["0", "1"])

    def test_empty_poll_does_not_launch(self):
        dispatch.launch([], 42, 1, lambda *_: self.fail("Unexpected dispatch"))

    def test_schedule_and_main_manual_poll_are_valid(self):
        for event in ("schedule", "workflow_dispatch"):
            self.run["event"] = event
            dispatch.validate_run(self.run, SHA, 1)

    def test_untrusted_or_changed_provenance_is_rejected(self):
        changes = [
            {"repository": {"full_name": "someone/fork"}},
            {"head_repository": {"full_name": "someone/fork"}},
            {"path": ".github/workflows/other.yaml"},
            {"head_branch": "feature", "event": "workflow_dispatch"},
            {"event": "pull_request"},
            {"head_sha": "b" * 40},
            {"run_attempt": 2},
        ]
        for change in changes:
            with self.subTest(change=change), self.assertRaises(ValueError):
                dispatch.validate_run(self.run | change, SHA, 1)

    def test_invalid_record_is_rejected(self):
        for index in (-1, 2):
            with self.assertRaises(ValueError):
                dispatch.select_record(self.records, index)
        for change in ({"agent": "review"}, {"source_repo": "someone/fork"}):
            records = copy.deepcopy(self.records)
            records[0].update(change)
            with self.assertRaises(ValueError):
                dispatch.select_record(records, 0)

    def test_workflows_pin_config_and_limit_permissions(self):
        # These contracts protect the reusable workflow's github.sha checkout.
        for name in ("fullsend-poll-jira.yaml", "fullsend-code-jira.yaml"):
            source = (ROOT / ".github/workflows" / name).read_text()
            self.assertIn("github.ref == 'refs/heads/main'", source)
            self.assertIn("ref: ${{ github.sha }}", source)
            self.assertIn("permissions: {}", source)
        source = (ROOT / ".github/workflows/fullsend-code-jira.yaml").read_text()
        self.assertIn("needs: validate", source)
        self.assertIn("matrix: ${{ needs.validate.outputs.matrix }}", source)


if __name__ == "__main__":
    unittest.main()
