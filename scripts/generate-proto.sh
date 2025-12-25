#!/bin/bash
# Generate Go code from protobuf files
# Run from repository root: ./scripts/generate-proto.sh

set -e

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT}/proto"
OUT_DIR="${ROOT}/internal/rpc/proto"

# Create output directory if it doesn't exist
mkdir -p "${OUT_DIR}"

echo "Generating Go code from protobuf files..."
echo "Proto dir: ${PROTO_DIR}"
echo "Output dir: ${OUT_DIR}"

# Generate Go code
# Note: We use module prefix to ensure all files go to the same package
protoc \
    --proto_path="${PROTO_DIR}" \
    --go_out="${ROOT}" \
    --go_opt=module=github.com/Azure/azure-functions-go-worker \
    --go-grpc_out="${ROOT}" \
    --go-grpc_opt=module=github.com/Azure/azure-functions-go-worker \
    "${PROTO_DIR}/shared/NullableTypes.proto" \
    "${PROTO_DIR}/identity/ClaimsIdentityRpc.proto" \
    "${PROTO_DIR}/FunctionRpc.proto"

echo "Proto generation completed successfully!"
