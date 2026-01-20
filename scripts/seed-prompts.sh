#!/bin/bash
set -euo pipefail

MLFLOW_URI="${MLFLOW_TRACKING_URI:-http://localhost:5000}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTDATA="${SCRIPT_DIR}/../testdata/prompts.json"

if [[ ! -f "${TESTDATA}" ]]; then
    echo "Error: Test data file not found at ${TESTDATA}"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed"
    exit 1
fi

echo "Seeding prompts to ${MLFLOW_URI}..."

# Process each prompt in the JSON array
jq -c '.prompts[]' "${TESTDATA}" | while read -r prompt; do
    name=$(echo "${prompt}" | jq -r '.name')
    description=$(echo "${prompt}" | jq -r '.description // ""')
    use_case=$(echo "${prompt}" | jq -r '.use_case // ""')

    echo "  Creating prompt: ${name} (use_case: ${use_case})"

    # Create RegisteredModel with prompt tag and use_case (ignore error if already exists)
    create_model_payload=$(jq -n \
        --arg name "$name" \
        --arg desc "$description" \
        --arg use_case "$use_case" \
        '{
            name: $name,
            description: $desc,
            tags: [
                {key: "mlflow.prompt.is_prompt", value: "true"},
                {key: "use_case", value: $use_case}
            ]
        }')

    curl -sSf -X POST "${MLFLOW_URI}/api/2.0/mlflow/registered-models/create" \
        -H "Content-Type: application/json" \
        -d "${create_model_payload}" \
        2>/dev/null || echo "    (prompt may already exist, continuing...)"

    # Create each version as ModelVersion with prompt text tag
    echo "${prompt}" | jq -c '.versions[]' | while read -r version; do
        template=$(echo "${version}" | jq -r '.template')
        ver_description=$(echo "${version}" | jq -r '.description // ""')

        # Build tags array including the template and is_prompt marker
        tags=$(echo "${version}" | jq -c --arg template "$template" '
            [
                {key: "mlflow.prompt.text", value: $template},
                {key: "_mlflow_prompt_type", value: "text"},
                {key: "mlflow.prompt.is_prompt", value: "true"}
            ] + (.tags // [])
        ')

        echo "    Creating version: ${ver_description:0:40}..."

        create_version_payload=$(jq -n \
            --arg name "$name" \
            --arg desc "$ver_description" \
            --argjson tags "$tags" \
            '{
                name: $name,
                source: ("mlflow-artifacts:/" + $name),
                description: $desc,
                tags: $tags
            }')

        response=$(curl -sSf -X POST "${MLFLOW_URI}/api/2.0/mlflow/model-versions/create" \
            -H "Content-Type: application/json" \
            -d "${create_version_payload}" 2>&1) || {
            echo "      Warning: Failed to create version - ${response}"
            continue
        }

        version_num=$(echo "${response}" | jq -r '.model_version.version // "?"')
        echo "      Created version ${version_num}"
    done
done

echo ""
echo "Seeding complete. Verifying..."

# Verify prompts were created
echo ""
echo "Registered prompts:"
for pname in greeting-prompt dog-walker-prompt qa-prompt; do
    result=$(curl -s "${MLFLOW_URI}/api/2.0/mlflow/registered-models/get?name=${pname}" 2>/dev/null)
    if echo "${result}" | jq -e '.registered_model' >/dev/null 2>&1; then
        versions=$(echo "${result}" | jq -r '.registered_model.latest_versions | length // 0')
        echo "  - ${pname} (${versions} versions)"
    fi
done

echo ""
echo "Done!"
