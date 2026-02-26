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

# Helper: POST to MLflow API
mlflow_post() {
    local endpoint="$1"
    local payload="$2"
    curl -sSf -X POST "${MLFLOW_URI}/api/2.0/mlflow/${endpoint}" \
        -H "Content-Type: application/json" \
        -d "${payload}" 2>/dev/null
}

# ============================================================
# Prompt Registry
# ============================================================

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
echo "Registered prompts:"
for pname in greeting-prompt dog-walker-prompt qa-prompt; do
    result=$(curl -s "${MLFLOW_URI}/api/2.0/mlflow/registered-models/get?name=${pname}" 2>/dev/null)
    if echo "${result}" | jq -e '.registered_model' >/dev/null 2>&1; then
        versions=$(echo "${result}" | jq -r '.registered_model.latest_versions | length // 0')
        echo "  - ${pname} (${versions} versions)"
    fi
done

# ============================================================
# Experiment Tracking
# ============================================================

echo ""
echo "Seeding experiments and runs..."

TIMESTAMP_MS=$(date +%s)000

# --- Experiment 1: Bella walk prediction ---
echo "  Creating experiment: bella-walk-prediction"
response=$(mlflow_post "experiments/create" \
    '{"name": "bella-walk-prediction"}' 2>/dev/null) || {
    echo "    (experiment may already exist, skipping...)"
    response=""
}

if [[ -n "$response" ]]; then
    EXP1_ID=$(echo "$response" | jq -r '.experiment_id')
    echo "    ID: ${EXP1_ID}"

    # Tag the experiment
    mlflow_post "experiments/set-experiment-tag" \
        "{\"experiment_id\": \"${EXP1_ID}\", \"key\": \"team\", \"value\": \"ml-platform\"}" > /dev/null
    mlflow_post "experiments/set-experiment-tag" \
        "{\"experiment_id\": \"${EXP1_ID}\", \"key\": \"dog\", \"value\": \"bella\"}" > /dev/null

    # Run 1: Random Forest baseline
    echo "    Creating run: random-forest-baseline"
    run_response=$(mlflow_post "runs/create" \
        "{\"experiment_id\": \"${EXP1_ID}\", \"run_name\": \"random-forest-baseline\", \"start_time\": ${TIMESTAMP_MS}, \"tags\": [{\"key\": \"model\", \"value\": \"random-forest\"}, {\"key\": \"stage\", \"value\": \"baseline\"}]}")
    RUN1_ID=$(echo "$run_response" | jq -r '.run.info.run_id')

    mlflow_post "runs/log-batch" "{\"run_id\": \"${RUN1_ID}\", \"params\": [{\"key\": \"n_estimators\", \"value\": \"100\"}, {\"key\": \"max_depth\", \"value\": \"5\"}, {\"key\": \"random_state\", \"value\": \"42\"}], \"metrics\": [{\"key\": \"rmse\", \"value\": 0.85, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}, {\"key\": \"rmse\", \"value\": 0.72, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 2}, {\"key\": \"rmse\", \"value\": 0.65, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 3}, {\"key\": \"accuracy\", \"value\": 0.78, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}, {\"key\": \"f1_score\", \"value\": 0.75, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}], \"tags\": [{\"key\": \"framework\", \"value\": \"scikit-learn\"}]}" > /dev/null

    mlflow_post "runs/update" \
        "{\"run_id\": \"${RUN1_ID}\", \"status\": \"FINISHED\", \"end_time\": ${TIMESTAMP_MS}}" > /dev/null
    echo "      Logged 3 params, 5 metrics, status=FINISHED"

    # Run 2: Gradient Boosting
    echo "    Creating run: gradient-boosting-tuned"
    run_response=$(mlflow_post "runs/create" \
        "{\"experiment_id\": \"${EXP1_ID}\", \"run_name\": \"gradient-boosting-tuned\", \"start_time\": ${TIMESTAMP_MS}, \"tags\": [{\"key\": \"model\", \"value\": \"gradient-boosting\"}, {\"key\": \"stage\", \"value\": \"tuning\"}]}")
    RUN2_ID=$(echo "$run_response" | jq -r '.run.info.run_id')

    mlflow_post "runs/log-batch" "{\"run_id\": \"${RUN2_ID}\", \"params\": [{\"key\": \"n_estimators\", \"value\": \"200\"}, {\"key\": \"learning_rate\", \"value\": \"0.1\"}, {\"key\": \"max_depth\", \"value\": \"7\"}, {\"key\": \"random_state\", \"value\": \"42\"}], \"metrics\": [{\"key\": \"rmse\", \"value\": 0.68, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}, {\"key\": \"rmse\", \"value\": 0.52, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 2}, {\"key\": \"rmse\", \"value\": 0.41, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 3}, {\"key\": \"accuracy\", \"value\": 0.89, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}, {\"key\": \"f1_score\", \"value\": 0.87, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}], \"tags\": [{\"key\": \"framework\", \"value\": \"scikit-learn\"}]}" > /dev/null

    mlflow_post "runs/update" \
        "{\"run_id\": \"${RUN2_ID}\", \"status\": \"FINISHED\", \"end_time\": ${TIMESTAMP_MS}}" > /dev/null
    echo "      Logged 4 params, 5 metrics, status=FINISHED"

    # Run 3: Neural network (still running)
    echo "    Creating run: neural-net-experiment"
    run_response=$(mlflow_post "runs/create" \
        "{\"experiment_id\": \"${EXP1_ID}\", \"run_name\": \"neural-net-experiment\", \"start_time\": ${TIMESTAMP_MS}, \"tags\": [{\"key\": \"model\", \"value\": \"neural-network\"}, {\"key\": \"stage\", \"value\": \"experiment\"}]}")
    RUN3_ID=$(echo "$run_response" | jq -r '.run.info.run_id')

    mlflow_post "runs/log-batch" "{\"run_id\": \"${RUN3_ID}\", \"params\": [{\"key\": \"hidden_layers\", \"value\": \"128,64\"}, {\"key\": \"learning_rate\", \"value\": \"0.001\"}, {\"key\": \"optimizer\", \"value\": \"adam\"}, {\"key\": \"epochs\", \"value\": \"50\"}], \"metrics\": [{\"key\": \"loss\", \"value\": 1.2, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}, {\"key\": \"loss\", \"value\": 0.8, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 10}, {\"key\": \"loss\", \"value\": 0.45, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 25}, {\"key\": \"accuracy\", \"value\": 0.91, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 25}], \"tags\": [{\"key\": \"framework\", \"value\": \"pytorch\"}]}" > /dev/null
    echo "      Logged 4 params, 4 metrics, status=RUNNING"
fi

# --- Experiment 2: Dora activity classification ---
echo "  Creating experiment: dora-activity-classification"
response=$(mlflow_post "experiments/create" \
    '{"name": "dora-activity-classification"}' 2>/dev/null) || {
    echo "    (experiment may already exist, skipping...)"
    response=""
}

if [[ -n "$response" ]]; then
    EXP2_ID=$(echo "$response" | jq -r '.experiment_id')
    echo "    ID: ${EXP2_ID}"

    mlflow_post "experiments/set-experiment-tag" \
        "{\"experiment_id\": \"${EXP2_ID}\", \"key\": \"team\", \"value\": \"ml-platform\"}" > /dev/null
    mlflow_post "experiments/set-experiment-tag" \
        "{\"experiment_id\": \"${EXP2_ID}\", \"key\": \"dog\", \"value\": \"dora\"}" > /dev/null

    # Run 1: SVM classifier
    echo "    Creating run: svm-classifier"
    run_response=$(mlflow_post "runs/create" \
        "{\"experiment_id\": \"${EXP2_ID}\", \"run_name\": \"svm-classifier\", \"start_time\": ${TIMESTAMP_MS}, \"tags\": [{\"key\": \"model\", \"value\": \"svm\"}, {\"key\": \"task\", \"value\": \"classification\"}]}")
    RUN_ID=$(echo "$run_response" | jq -r '.run.info.run_id')

    mlflow_post "runs/log-batch" "{\"run_id\": \"${RUN_ID}\", \"params\": [{\"key\": \"kernel\", \"value\": \"rbf\"}, {\"key\": \"C\", \"value\": \"1.0\"}, {\"key\": \"gamma\", \"value\": \"scale\"}], \"metrics\": [{\"key\": \"accuracy\", \"value\": 0.82, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}, {\"key\": \"precision\", \"value\": 0.80, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}, {\"key\": \"recall\", \"value\": 0.79, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}], \"tags\": [{\"key\": \"framework\", \"value\": \"scikit-learn\"}]}" > /dev/null

    mlflow_post "runs/update" \
        "{\"run_id\": \"${RUN_ID}\", \"status\": \"FINISHED\", \"end_time\": ${TIMESTAMP_MS}}" > /dev/null
    echo "      Logged 3 params, 3 metrics, status=FINISHED"

    # Run 2: XGBoost (failed run)
    echo "    Creating run: xgboost-failed"
    run_response=$(mlflow_post "runs/create" \
        "{\"experiment_id\": \"${EXP2_ID}\", \"run_name\": \"xgboost-failed\", \"start_time\": ${TIMESTAMP_MS}, \"tags\": [{\"key\": \"model\", \"value\": \"xgboost\"}, {\"key\": \"task\", \"value\": \"classification\"}]}")
    RUN_ID=$(echo "$run_response" | jq -r '.run.info.run_id')

    mlflow_post "runs/log-batch" "{\"run_id\": \"${RUN_ID}\", \"params\": [{\"key\": \"n_estimators\", \"value\": \"500\"}, {\"key\": \"learning_rate\", \"value\": \"0.3\"}], \"metrics\": [{\"key\": \"accuracy\", \"value\": 0.45, \"timestamp\": ${TIMESTAMP_MS}, \"step\": 1}], \"tags\": [{\"key\": \"framework\", \"value\": \"xgboost\"}, {\"key\": \"error\", \"value\": \"OOM on large feature set\"}]}" > /dev/null

    mlflow_post "runs/update" \
        "{\"run_id\": \"${RUN_ID}\", \"status\": \"FAILED\", \"end_time\": ${TIMESTAMP_MS}}" > /dev/null
    echo "      Logged 2 params, 1 metric, status=FAILED"
fi

echo ""
echo "Seeding complete!"
