# Azure Functions Go Worker - QR Generator (gRPC)
#
# This container bundles the Azure Functions Host with the Go gRPC worker,
# enabling deployment to Azure Container Apps.
#
# Build from repository root:
#   docker build -t qr-generator-grpc -f samples/qr-generator-grpc/Dockerfile .
#
# Run locally:
#   docker run -p 7071:80 -e AzureWebJobsStorage="UseDevelopmentStorage=true" qr-generator-grpc

# Stage 1: Build the Go worker
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy the entire project (needed for internal packages)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the qr-generator sample for Linux (static binary)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o worker ./samples/qr-generator-grpc

# Stage 2: Azure Functions Host with Go worker
FROM mcr.microsoft.com/azure-functions/base:4

# Create worker directory
RUN mkdir -p /azure-functions-host/workers/go

# Copy Go worker binary
COPY --from=builder /build/worker /azure-functions-host/workers/go/worker
RUN chmod +x /azure-functions-host/workers/go/worker

# Copy worker configuration
COPY samples/qr-generator-grpc/worker.config.json /azure-functions-host/workers/go/worker.config.json

# Copy function app files
COPY samples/qr-generator-grpc/host.json /home/site/wwwroot/host.json
COPY samples/qr-generator-grpc/Generate/ /home/site/wwwroot/Generate/
COPY samples/qr-generator-grpc/Health/ /home/site/wwwroot/Health/

# Set environment variables
ENV AzureWebJobsScriptRoot=/home/site/wwwroot \
    FUNCTIONS_WORKER_RUNTIME=go \
    languageWorkers__workersDirectory=/azure-functions-host/workers

# Expose the Functions port
EXPOSE 80

# The base image handles starting the Functions Host
