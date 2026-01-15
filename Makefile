# ABOUTME: Build and development automation for the MLflow Go SDK.
# ABOUTME: Provides targets for testing, code generation, and local MLflow server management.

.PHONY: test/unit test/integration gen dev/up dev/down dev/reset help lint vet fmt tidy check run-sample

# Configuration
MLFLOW_PORT ?= 5000
MLFLOW_DATA ?= $(shell pwd)/.mlflow
LOCALBIN ?= $(shell pwd)/bin
UV ?= $(LOCALBIN)/uv
PROTOC_GEN_GO ?= $(LOCALBIN)/protoc-gen-go
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.63.4

# Help target
help:
	@echo "MLflow Go SDK - Development Commands"
	@echo ""
	@echo "Testing:"
	@echo "  make test/unit        - Run unit tests with race detector"
	@echo "  make test/integration - Run integration tests (run dev/up in another terminal)"
	@echo "  make check            - Run all checks (lint, vet, test)"
	@echo ""
	@echo "Linting:"
	@echo "  make lint             - Run golangci-lint"
	@echo "  make vet              - Run go vet"
	@echo "  make fmt              - Format code with gofmt"
	@echo "  make tidy             - Run go mod tidy"
	@echo ""
	@echo "Development:"
	@echo "  make dev/up           - Start local MLflow server (foreground, Ctrl+C to stop)"
	@echo "  make dev/down         - Stop local MLflow server"
	@echo "  make dev/reset        - Reset MLflow server (nuke DB, restart, seed)"
	@echo ""
	@echo "Code Generation:"
	@echo "  make gen              - Generate protobuf types from MLflow protos"
	@echo ""
	@echo "Sample:"
	@echo "  make run-sample       - Run sample app (requires dev/up)"

# Testing targets
test/unit:
	go test -v -race ./... -tags=!integration

test/integration:
	MLFLOW_TRACKING_URI=http://localhost:$(MLFLOW_PORT) \
	MLFLOW_INSECURE_SKIP_TLS_VERIFY=true \
	go test -v -race -tags=integration ./...

# Linting targets
$(GOLANGCI_LINT):
	@mkdir -p $(LOCALBIN)
	@echo "Installing golangci-lint..."
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

vet:
	go vet ./...

fmt:
	gofmt -w -s .

tidy:
	go mod tidy

check: lint vet test/unit
	@echo "All checks passed!"

# Protoc-gen-go installation (lazy install)
$(PROTOC_GEN_GO):
	@mkdir -p $(LOCALBIN)
	@echo "Installing protoc-gen-go..."
	GOBIN=$(LOCALBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Code generation
gen: tools/proto/fetch-protos.sh $(PROTOC_GEN_GO)
	@echo "Fetching MLflow protos..."
	@./tools/proto/fetch-protos.sh
	@echo "Generating Go types..."
	@which protoc > /dev/null || (echo "Error: protoc not installed. Install via: brew install protobuf" && exit 1)
	PATH=$(LOCALBIN):$$PATH protoc \
		--proto_path=internal/gen/mlflowpb \
		--proto_path=tools/proto/stubs \
		--go_out=internal/gen/mlflowpb \
		--go_opt=paths=source_relative \
		--go_opt=Mmodel_registry.proto=github.com/ederign/mlflow-go/internal/gen/mlflowpb \
		--go_opt=Mdatabricks.proto=github.com/ederign/mlflow-go/internal/gen/mlflowpb \
		model_registry.proto databricks.proto

# UV installation (lazy install)
$(UV):
	@mkdir -p $(LOCALBIN)
	@echo "Installing uv..."
	@curl -LsSf https://astral.sh/uv/install.sh | CARGO_HOME=$(LOCALBIN)/.cargo UV_INSTALL_DIR=$(LOCALBIN) sh
	@test -f $(UV) || (echo "uv installation failed" && exit 1)

# Development server targets
dev/up: $(UV)
	@mkdir -p $(MLFLOW_DATA)
	@echo "Starting MLflow server on port $(MLFLOW_PORT)... (Ctrl+C to stop)"
	$(UV) run --with mlflow mlflow server \
		--host 127.0.0.1 \
		--port $(MLFLOW_PORT) \
		--backend-store-uri sqlite:///$(MLFLOW_DATA)/mlflow.db \
		--default-artifact-root $(MLFLOW_DATA)/artifacts

dev/down:
	@echo "Stopping MLflow server..."
	@lsof -t -i :$(MLFLOW_PORT) | xargs kill 2>/dev/null || true
	@echo "MLflow server stopped."

dev/reset: dev/down
	@echo "Nuking MLflow data..."
	@rm -rf $(MLFLOW_DATA)
	@$(MAKE) dev/up
	@echo "Seeding sample prompts..."
	@./scripts/seed-prompts.sh || echo "Note: seed script not yet created"
	@echo "MLflow reset complete."

# Sample app
run-sample:
	@echo "Running sample app..."
	cd sample-app && go run .
