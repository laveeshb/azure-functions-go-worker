# Azure Functions Go Worker - Makefile

.PHONY: all build generate test clean fmt lint vet example help

# Default target
all: generate build

# Build the worker
build:
	go build -o bin/worker.exe ./cmd/worker

# Build the example
example:
	go build -o examples/httpTrigger/worker.exe ./examples/httpTrigger

# Generate protobuf code
generate:
ifeq ($(OS),Windows_NT)
	powershell -ExecutionPolicy Bypass -File ./scripts/generate-proto.ps1
else
	./scripts/generate-proto.sh
endif

# Run go generate
go-generate:
	go generate ./...

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f examples/httpTrigger/worker.exe
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Run go vet
vet:
	go vet ./...

# Tidy dependencies
tidy:
	go mod tidy

# Download dependencies
deps:
	go mod download

# Install development tools
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run the example locally (requires Azure Functions Core Tools)
run-example: example
	cd examples/httpTrigger && func start

# Show help
help:
	@echo "Azure Functions Go Worker - Build Targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all            - Generate protobuf and build (default)"
	@echo "  build          - Build the worker binary"
	@echo "  example        - Build the example function app"
	@echo "  generate       - Generate Go code from protobuf files"
	@echo "  go-generate    - Run go generate"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  clean          - Remove build artifacts"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linter"
	@echo "  vet            - Run go vet"
	@echo "  tidy           - Tidy go.mod"
	@echo "  deps           - Download dependencies"
	@echo "  tools          - Install development tools"
	@echo "  run-example    - Run the example locally"
	@echo "  help           - Show this help"
