# QR Code Generator - gRPC Worker (Azure Container Apps)

A sample Azure Functions app that generates QR codes, built with the **native gRPC worker**.

> **Deploy to:** Azure Container Apps
> 
> This sample uses the full gRPC protocol for communication with the Functions host.
> For a simpler deployment to Azure Functions (PaaS), see [../qr-generator-custom-handler/](../qr-generator-custom-handler/).

## Table of Contents

- [Why gRPC?](#why-grpc)
- [Features](#features)
- [Local Development](#local-development)
- [Deploy to Azure Container Apps](#deploy-to-azure-container-apps)
- [API Reference](#api-reference)
- [Container Architecture](#container-architecture)
- [Privacy](#privacy)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Why gRPC?

This sample demonstrates the **native gRPC worker** approach. Here's when to choose each option:

| Feature | gRPC (this sample) | Custom Handler |
|---------|-------------------|----------------|
| **Deploy to** | Azure Container Apps | Azure Functions (PaaS) |
| **Triggers** | All (HTTP, Queue, Blob, Timer) | HTTP only (practical) |
| **Binding Support** | Full - typed bindings | Manual JSON parsing |
| **Protocol** | gRPC bidirectional streaming | HTTP |
| **Setup** | Container + ACA | Simple script |
| **Cold Start** | ~2-5s (container) | ~500ms-2s |

### Why Can't gRPC Deploy to Azure Functions?

Azure Functions (PaaS) only supports languages on its **runtime allowlist**:

| Runtime | Status |
|---------|--------|
| `dotnet`, `node`, `python`, `java`, `powershell` | ✅ Built-in |
| `custom` (HTTP) | ✅ Supported |
| `go` (gRPC) | ❌ Not on allowlist |

**Solution:** We bundle the Azure Functions Host with our Go worker in a container and deploy to Azure Container Apps, where we control the runtime.

## Features

- **Native gRPC** - Uses the same protocol as Python/Node workers
- **Interactive Web UI** - User-friendly landing page at `/api/generate`
- Generate QR codes from any text or URL
- Configurable image size (up to 1024px)
- Download generated QR codes as PNG
- Returns base64-encoded PNG images via API
- Health check endpoint
- **Full Binding Support** - Ready for Queue, Blob, Timer triggers

## Local Development

The gRPC worker requires the Go worker to be registered with the Functions host. 
For quick local testing, you can either:

### Option 1: Use the Custom Handler version

For simple local testing, use the [Custom Handler version](../qr-generator-custom-handler/) which works out of the box:

```bash
cd ../qr-generator-custom-handler
go build -o handler.exe .
func start
```

### Option 2: Run with Docker locally

```bash
# Build from repository root
cd ../..
docker build -t qr-generator-grpc -f samples/qr-generator-grpc/Dockerfile .

# Run locally
docker run -p 7071:80 qr-generator-grpc
```

### Option 3: Register the worker with Core Tools

See [docs/design/ARCHITECTURE.md](../../docs/design/ARCHITECTURE.md) for instructions on registering the Go worker with Azure Functions Core Tools.

## Deploy to Azure Container Apps

### Prerequisites

- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
- [Docker](https://docs.docker.com/get-docker/)
- Azure subscription

### Quick Deploy

```powershell
# PowerShell
.\deploy-aca.ps1 -ResourceGroupName "rg-qr-grpc" -Location "eastus"
```

```bash
# Bash
./deploy-aca.sh -g "rg-qr-grpc" -l "eastus"
```

### Manual Deployment

1. **Build the container:**

```bash
docker build -t qr-generator-grpc .
```

2. **Create Azure Container Registry:**

```bash
az acr create --name myregistry --resource-group rg-qr-grpc --sku Basic
az acr login --name myregistry
```

3. **Push the image:**

```bash
docker tag qr-generator-grpc myregistry.azurecr.io/qr-generator-grpc:latest
docker push myregistry.azurecr.io/qr-generator-grpc:latest
```

4. **Deploy to Azure Container Apps:**

```bash
az containerapp create \
  --name qr-generator \
  --resource-group rg-qr-grpc \
  --image myregistry.azurecr.io/qr-generator-grpc:latest \
  --target-port 80 \
  --ingress external \
  --registry-server myregistry.azurecr.io
```

## API Reference

### GET /api/generate

Serves an interactive web page for generating QR codes.

### POST /api/generate

Generate a QR code from text or URL.

**Request:**
```json
{
  "content": "https://example.com",
  "size": 256
}
```

**Response:**
```json
{
  "image": "iVBORw0KGgoAAAANSUhEUg...",
  "content": "https://example.com",
  "size": 256
}
```

### GET /api/health

Health check endpoint.

```json
{
  "status": "healthy",
  "service": "qr-generator"
}
```

## Container Architecture

The container bundles everything needed to run the Go worker:

```
┌─────────────────────────────────────────────────────────────┐
│  Container                                                   │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Azure Functions Host (Microsoft base image)            ││
│  │  - Reads workers/go/worker.config.json                  ││
│  │  - Starts Go worker with gRPC environment variables     ││
│  │  - Routes HTTP requests to functions                    ││
│  └─────────────────────────────────────────────────────────┘│
│                           ▲                                  │
│                           │ gRPC (localhost)                 │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Go Worker (our binary)                                 ││
│  │  - Connects to host's gRPC endpoint                     ││
│  │  - Handles function invocations                         ││
│  │  - Returns responses via gRPC stream                    ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### Dockerfile

The Dockerfile must be built from the repository root to access internal packages:

```dockerfile
# Build from repository root:
# docker build -t qr-generator-grpc -f samples/qr-generator-grpc/Dockerfile .

FROM golang:1.21-alpine AS builder
# ... builds the Go worker

FROM mcr.microsoft.com/azure-functions/base:4
# Adds Go worker to the Functions Host
COPY --from=builder /build/worker /azure-functions-host/workers/go/worker
# Copies function definitions
COPY samples/qr-generator-grpc/host.json /home/site/wwwroot/host.json
# ...
```

See the full [Dockerfile](Dockerfile) for details.

## Privacy

🔒 **Your data is never stored.**

- No input logging
- No data retention
- No cookies or tracking
- All processing in-memory
- Data discarded after response

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Container fails to start | Check `docker logs <container-id>` for worker registration errors |
| gRPC connection refused | Verify `FUNCTIONS_GRPC_HOST` and `FUNCTIONS_GRPC_PORT` environment variables |
| Functions not discovered | Check function.json files exist in subdirectories |
| 500 errors | Check Application Insights for detailed error messages |

### Logs

```bash
# Docker logs
docker logs <container-id>

# Azure Container Apps logs
az containerapp logs show --name qr-generator --resource-group rg-qr-grpc
```

## Project Structure

```
qr-generator-grpc/
├── main.go              # gRPC worker with function handlers
├── go.mod               # Go module (references azure-functions-go-worker)
├── host.json            # Functions host configuration
├── local.settings.json  # Local development settings
├── Dockerfile           # Container definition
├── deploy-aca.ps1       # PowerShell deployment script
├── deploy-aca.sh        # Bash deployment script
├── worker.config.json   # Worker registration for Core Tools
├── Generate/
│   └── function.json    # Generate function binding
└── Health/
    └── function.json    # Health function binding
```

## License

MIT License - See [LICENSE](../../LICENSE) for details.
