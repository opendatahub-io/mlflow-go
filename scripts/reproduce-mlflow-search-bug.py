#!/usr/bin/env python3
"""
Reproducer for MLflow OSS model-versions/search returning empty.
Run with: uv run --with mlflow python scripts/reproduce-mlflow-search-bug.py

Tested with: MLflow 3.8.1
Expected: Returns model versions matching filter
Actual: Returns empty list
"""
import mlflow
from mlflow import MlflowClient

# Requires: make dev/up running on localhost:5000
mlflow.set_tracking_uri("http://localhost:5000")
client = MlflowClient()

# First, create a test prompt
MODEL_NAME = "test-search-bug"

# Create registered model
try:
    client.create_registered_model(
        MODEL_NAME,
        tags=[{"key": "mlflow.prompt.is_prompt", "value": "true"}]
    )
    print(f"Created model: {MODEL_NAME}")
except Exception as e:
    if "ALREADY_EXISTS" in str(e):
        print(f"Model {MODEL_NAME} already exists")
    else:
        raise

# Create a version
client.create_model_version(
    name=MODEL_NAME,
    source=f"mlflow-artifacts:/{MODEL_NAME}",
    tags=[{"key": "mlflow.prompt.text", "value": "Hello {{name}}!"}]
)
print("Created version")

# Verify the version exists via direct get
v = client.get_model_version(MODEL_NAME, "1")
print(f"Direct get works: v{v.version}")

# Now try search - THIS FAILS
print("\nTrying search_model_versions...")
versions = list(client.search_model_versions(filter_string=f"name='{MODEL_NAME}'"))
print(f"Search returned: {len(versions)} versions (expected: >= 1)")

if len(versions) == 0:
    print("\n*** BUG CONFIRMED: search_model_versions returns empty ***")
    print("The endpoint returns {} regardless of filter.")
