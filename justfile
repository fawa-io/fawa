set shell := ["bash", "-eu", "-o", "pipefail", "-c"]
set positional-arguments := false

# --- Workspace-aware Commands ---

# Run unit tests for all services in the workspace
test:
    @echo "Running unit tests for all modules..."
    go test -v -cover ./...

# Tidy dependencies for all modules in the workspace
tidy:
    @echo "Tidying go modules in workspace..."
    @awk '/^\t\./ { sub("^[ \t]+", ""); print }' go.work | xargs -I {} bash -c 'echo "Tidying {}..."; cd {} && go mod tidy'

# Format all go files in the workspace
fmt:
    @echo "Formatting go files..."
    go fmt ./...

# Lint all code in the workspace
lint:
    @echo "Linting code..."
    # check if golangci-lint command exists.
    @if ! command -v golangci-lint &> /dev/null; then \
        go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
    fi
    golangci-lint run ./...

# --- Service-specific Commands ---

# Build a specific service. Usage: just build <service-name>
# Example: just build fileservice
build service:
    @echo "Building service: {{service}}..."
    go build -v -o bin/{{service}} ./services/{{service}}

# Run a specific service. Usage: just run <service-name>
# Example: just run fileservice
run service:
    @echo "Running service: {{service}}..."
    go run ./services/{{service}}

# --- Other Commands ---

# Generate protobuf files
generate:
    @echo "Generating protobuf files..."
    rm -rf gen/
    buf generate

# Clean all build artifacts
clean:
    @echo "Cleaning up..."
    rm -rf bin/
    rm -rf gen/

# Check license header
check:
    @echo "Checking license header..."
    license-eye -c .licenserc.yaml header check

# List all available commands
default:
    @just --list