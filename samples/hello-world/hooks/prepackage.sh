#!/bin/bash
# Pre-package hook for azd - builds the Go binary for Linux
set -e

echo "Building Go binary for Linux..."
cd src

# Build for Linux (Azure Functions runs on Linux)
GOOS=linux GOARCH=amd64 go build -o handler .

echo "Go binary built successfully"
ls -la handler
