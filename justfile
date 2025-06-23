set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

set positional-arguments := false

fawa-server-bin := "fawa-server"

fawa-server-dir := "."

# just command list
default:
    @just --list

# build fawa server
build:
    @echo "Building fawa server..."
    go build -v -o {{fawa-server-bin}} {{fawa-server-dir}}

# run fawa server
run:
    @echo "Running fawa server..."
    go run {{fawa-server-dir}}

# run unit tests
test:
    @echo "Running unit tests..."
    go test -v -cover ./...

# go mod tidy
tidy:
    @echo "Tidying go modules..."
    go mod tidy

# go fmt ./...
fmt:
    @echo "Formatting go files..."
    go fmt ./...

# run golangci-lint
lint:
    @echo "Linting code..."
    # check if golangci-lint command exists.
    @if ! command -v golangci-lint &> /dev/null; then \
        go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6; \
    fi
    golangci-lint run ./...

# generate protobuf files
generate:
    @echo "Generating protobuf files..."
    rm -rf gen/
    buf generate

# clean fawa server
clean:
    @echo "Cleaning up..."
    @if [ -f {{fawa-server-bin}} ]; then \
        rm {{fawa-server-bin}}; \
    fi
    rm -rf gen/

# check license header
check:
    @echo "Checking license header..."
    license-eye -c .licenserc.yaml header check
