set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

set positional-arguments := false

binary-name := "fawa"

main-package := "./cmd/server/"

# just command list
default:
    @just --list

# build fawa
build:
    @echo "Building fawa..."
    go build -v -o {{binary-name}} {{main-package}}

# run fawa
run:
    @echo "Running the application..."
    go run {{main-package}}

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
proto:generate:
    @echo "Generating protobuf files..."
    buf generate

# clean fawa
clean:
    @echo "Cleaning up..."
    @if [ -f {{binary-name}} ]; then \
        rm {{binary-name}}; \
    fi
