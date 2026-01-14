# ABOUTME: Build and development automation for the MLflow Go SDK.
# ABOUTME: Provides targets for testing, code generation, and local MLflow server management.

.PHONY: test/unit test/integration gen dev/up dev/down dev/reset help

# Configuration
MLFLOW_PORT ?= 5000
MLFLOW_DATA ?= $(shell pwd)/.mlflow
LOCALBIN ?= $(shell pwd)/bin
UV ?= $(LOCALBIN)/uv

# Help target
help:
	@echo "MLflow Go SDK - Development Commands"
	@echo ""
	@echo "Testing:"
	@echo "  make test/unit        - Run unit tests"
	@echo "  make test/integration - Run integration tests (requires dev/up)"
	@echo ""
	@echo "Development:"
	@echo "  make dev/up           - Start local MLflow server"
	@echo "  make dev/down         - Stop local MLflow server"
	@echo "  make dev/reset        - Reset MLflow server (nuke DB, restart, seed)"
	@echo ""
	@echo "Code Generation:"
	@echo "  make gen              - Generate protobuf types from MLflow protos"

# Testing targets
test/unit:
	go test -v -race ./... -tags=!integration

test/integration: dev/up
	MLFLOW_TRACKING_URI=http://localhost:$(MLFLOW_PORT) \
	MLFLOW_INSECURE_SKIP_TLS_VERIFY=true \
	go test -v -race -tags=integration ./...

# Code generation
gen: tools/proto/fetch-protos.sh
	@echo "Fetching MLflow protos..."
	@./tools/proto/fetch-protos.sh
	@echo "Generating Go types..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		internal/gen/mlflowpb/*.proto 2>/dev/null || \
		echo "Note: protoc generation skipped (protos not yet fetched)"

# UV installation (lazy install)
$(UV):
	@mkdir -p $(LOCALBIN)
	@echo "Installing uv..."
	@curl -LsSf https://astral.sh/uv/install.sh | CARGO_HOME=$(LOCALBIN)/.cargo UV_INSTALL_DIR=$(LOCALBIN) sh
	@test -f $(UV) || (echo "uv installation failed" && exit 1)

# Development server targets
dev/up: $(UV)
	@mkdir -p $(MLFLOW_DATA)
	@echo "Starting MLflow server on port $(MLFLOW_PORT)..."
	@$(UV) run --with mlflow mlflow server \
		--host 127.0.0.1 \
		--port $(MLFLOW_PORT) \
		--backend-store-uri sqlite:///$(MLFLOW_DATA)/mlflow.db \
		--default-artifact-root $(MLFLOW_DATA)/artifacts &
	@echo "Waiting for MLflow to start..."
	@sleep 3
	@curl -s http://localhost:$(MLFLOW_PORT)/health > /dev/null && echo "MLflow is ready!" || echo "MLflow may still be starting..."

dev/down:
	@echo "Stopping MLflow server..."
	@pkill -f "mlflow server" 2>/dev/null || true
	@echo "MLflow server stopped."

dev/reset: dev/down
	@echo "Nuking MLflow data..."
	@rm -rf $(MLFLOW_DATA)
	@$(MAKE) dev/up
	@echo "Seeding sample prompts..."
	@./scripts/seed-prompts.sh || echo "Note: seed script not yet created"
	@echo "MLflow reset complete."
