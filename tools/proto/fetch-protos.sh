#!/bin/bash
# ABOUTME: Fetches MLflow protobuf files from the pinned commit.
# ABOUTME: Downloads only the proto files needed for Go SDK code generation.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
PROTO_VERSION_FILE="${SCRIPT_DIR}/PROTO_VERSION"

# Read the pinned commit SHA
if [[ ! -f "${PROTO_VERSION_FILE}" ]]; then
    echo "Error: PROTO_VERSION file not found at ${PROTO_VERSION_FILE}"
    exit 1
fi

# shellcheck source=/dev/null
source "${PROTO_VERSION_FILE}"

if [[ -z "${MLFLOW_COMMIT:-}" ]]; then
    echo "Error: MLFLOW_COMMIT not set in PROTO_VERSION"
    exit 1
fi

# Output directory for proto files
OUTPUT_DIR="${PROJECT_ROOT}/internal/gen/mlflowpb"
mkdir -p "${OUTPUT_DIR}"

# Base URL for raw proto files
BASE_URL="https://raw.githubusercontent.com/mlflow/mlflow/${MLFLOW_COMMIT}/mlflow/protos"

# Proto files needed for Prompt Registry (Model Registry for OSS MLflow)
PROTO_FILES=(
    "model_registry.proto"
)

echo "Fetching MLflow protos from commit ${MLFLOW_COMMIT}..."

for proto in "${PROTO_FILES[@]}"; do
    echo "  Downloading ${proto}..."
    curl -sSfL "${BASE_URL}/${proto}" -o "${OUTPUT_DIR}/${proto}"
done

# Post-process: Remove scalapb import and options (Scala-specific, not needed for Go)
echo "  Post-processing: removing scalapb references..."
for proto in "${PROTO_FILES[@]}"; do
    sed -i '' \
        -e '/import "scalapb\/scalapb.proto";/d' \
        -e '/option (scalapb/d' \
        -e '/(scalapb.message)/d' \
        "${OUTPUT_DIR}/${proto}"
done

echo "Proto files downloaded to ${OUTPUT_DIR}"
echo ""
echo "Next steps:"
echo "  1. Run 'make gen' to generate Go types"
echo "  2. Or run: protoc --go_out=. --go_opt=paths=source_relative ${OUTPUT_DIR}/*.proto"
