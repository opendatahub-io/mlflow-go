#!/usr/bin/env python3
"""Strip unused APIs from MLflow service.proto for the Go SDK.

Each group below is stripped because its RPCs reference messages defined in
proto files that tools/proto/fetch-protos.sh does not vendor (issues.proto,
prompt_optimization.proto, label_schemas.proto, review_queues.proto). Leaving
them in makes protoc fail with "is not defined", so keeping an API means
vendoring its proto and generating a Go package for it.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

# Imports of proto files the Go SDK does not vendor. Stripped along with every
# RPC that references them.
UNVENDORED_IMPORTS = [
    "issues.proto",
    "prompt_optimization.proto",
    "label_schemas.proto",
    "review_queues.proto",
]

RPC_NAMES = (
    "createIssue|updateIssue|getIssue|searchIssues|"
    "createPromptOptimizationJob|getPromptOptimizationJob|"
    "searchPromptOptimizationJobs|cancelPromptOptimizationJob|"
    "deletePromptOptimizationJob|"
    # label_schemas.proto (MLflow 3.13+)
    "createLabelSchema|getLabelSchema|getLabelSchemaByName|"
    "listLabelSchemas|updateLabelSchema|deleteLabelSchema|"
    # review_queues.proto (MLflow 3.13+)
    "createReviewQueue|getOrCreateUserQueue|getReviewQueue|"
    "getReviewQueueByName|listReviewQueues|updateReviewQueue|"
    "deleteReviewQueue|addItemsToReviewQueue|removeItemsFromReviewQueue|"
    "listReviewQueueItems|setReviewQueueItemStatus"
)

MESSAGE_NAMES = [
    "CreatePromptOptimizationJob",
    "GetPromptOptimizationJob",
    "SearchPromptOptimizationJobs",
    "CancelPromptOptimizationJob",
    "DeletePromptOptimizationJob",
    "PromptOptimizationJob",
    "PromptOptimizationJobConfig",
    "PromptOptimizationJobTag",
]

FORBIDDEN = [
    *(f'import "{name}"' for name in UNVENDORED_IMPORTS),
    "PromptOptimizationJob",
    "CreateIssue",
    "createPromptOptimizationJob",
    "Issue RPCs",
    "Prompt Optimization API Messages",
    "prompt optimization job",
    "mlflow.label_schemas.",
    "mlflow.review_queues.",
    "Label Schema RPCs",
    "Review Queue RPCs",
]


def default_proto_dir() -> Path:
    """Return internal/gen/mlflowpb relative to this repository."""
    # tools/proto/strip-unused-service-apis.py → repo root → internal/gen/mlflowpb
    return Path(__file__).resolve().parent.parent.parent / "internal" / "gen" / "mlflowpb"


def validate_proto_path(path_arg: str, allowed_dir: Path | None = None) -> Path:
    """Resolve and validate a CLI proto path before open/write.

    Requires a .proto extension and that the resolved path stays under the
    allowed proto directory (default: internal/gen/mlflowpb).
    """
    root = (allowed_dir if allowed_dir is not None else default_proto_dir()).resolve()
    candidate = Path(path_arg).expanduser().resolve()
    if candidate.suffix != ".proto":
        raise ValueError(f"path must have a .proto extension: {path_arg}")
    try:
        candidate.relative_to(root)
    except ValueError as err:
        raise ValueError(f"path must be under {root}: {path_arg}") from err
    if not candidate.is_file():
        raise ValueError(f"proto file not found: {candidate}")
    return candidate


def remove_rpc_blocks(text: str) -> str:
    """Remove target RPCs and any immediately preceding comment/blank lines."""
    # End at the RPC's own closing brace so following kept-RPC comments survive.
    pattern = re.compile(
        rf"  rpc (?:{RPC_NAMES})\b.*?\n  \}}\n?",
        re.DOTALL,
    )
    while True:
        match = pattern.search(text)
        if not match:
            break
        start = match.start()
        lines = text[:start].splitlines(keepends=True)
        i = len(lines)
        while i > 0:
            stripped = lines[i - 1].strip()
            if stripped == "" or stripped.startswith("//"):
                i -= 1
                continue
            break
        text = "".join(lines[:i]) + text[match.end() :]
    return text


def _is_section_banner(line: str) -> bool:
    """Return True for // =====... banner / section-header comment lines."""
    if not line.startswith("//"):
        return False
    body = line[2:].strip()
    return body.startswith("=") or "====" in body


def remove_messages(text: str) -> str:
    """Remove target messages and an immediately preceding banner header only.

    Ordinary // documentation that belongs to the next retained message is
    preserved by ending each match at the target message's own closing brace.
    """
    for name in MESSAGE_NAMES:
        # Top-level messages close with '}' at column 0; nested message Response
        # blocks close with indented '  }', so they are not truncated early.
        pattern = re.compile(
            rf"message {name} \{{.*?\n\}}\n?",
            re.DOTALL,
        )
        while True:
            match = pattern.search(text)
            if not match:
                break
            start = match.start()
            lines = text[:start].splitlines(keepends=True)
            i = len(lines)
            while i > 0:
                stripped = lines[i - 1].strip()
                if stripped == "" or _is_section_banner(stripped):
                    i -= 1
                    continue
                break
            text = "".join(lines[:i]) + text[match.end() :]
    return text


def cleanup_orphans(text: str) -> str:
    """Remove any leftover headers and collapse blank lines from removals."""
    for orphan in [
        r"\n  // Issue RPCs\n+",
        r"\n  // Label Schema RPCs\n+",
        r"\n  // Review Queue RPCs\n+",
        r"\n// ========== Prompt Optimization API Messages ==========+\n+",
    ]:
        text = re.sub(orphan, "\n", text)
    return re.sub(r"\n{3,}", "\n\n", text)


def main() -> int:
    if len(sys.argv) != 2:
        print(f"usage: {sys.argv[0]} <service.proto>", file=sys.stderr)
        return 2

    try:
        path = validate_proto_path(sys.argv[1])
    except ValueError as err:
        print(f"ERROR: {err}", file=sys.stderr)
        return 2

    with open(path) as f:
        content = f.read()

    for name in UNVENDORED_IMPORTS:
        content = re.sub(
            rf'^import "{re.escape(name)}";\n', "", content, flags=re.MULTILINE
        )
    content = remove_rpc_blocks(content)
    content = remove_messages(content)
    content = cleanup_orphans(content)

    for token in FORBIDDEN:
        if token in content:
            print(
                f"ERROR: service.proto still contains {token!r} after post-processing",
                file=sys.stderr,
            )
            return 1

    with open(path, "w") as f:
        f.write(content)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
