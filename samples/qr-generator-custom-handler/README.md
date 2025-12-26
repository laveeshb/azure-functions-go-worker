# QR Code Generator - Custom Handler

A complete, deployable QR Code Generator Azure Function using **Custom Handler** (HTTP-based).

> **Deploy to:** Azure Functions (PaaS) - Works today!
> 
> For the gRPC version with full binding support (deploys to Azure Container Apps), 
> see [../qr-generator-grpc/](../qr-generator-grpc/).

## Table of Contents

- [Quick Start](#quick-start)
- [Features](#features)
- [API Reference](#api-reference)
- [Deployment Options](#deployment-options)
- [Local Development](#local-development)
- [Why Custom Handler?](#why-custom-handler)
- [Troubleshooting](#troubleshooting)
- [Privacy](#privacy)

## Quick Start

### Deploy to Azure

```powershell
# PowerShell (Windows)
.\deploy.ps1 -ResourceGroupName "rg-qr-generator" -Location "eastus"
```

```bash
# Bash (Linux/Mac)
./deploy.sh -g "rg-qr-generator" -l "eastus"
```

That's it! The script will:
1. Cross-compile the Go binary for Linux
2. Create a Resource Group
3. Create a Storage Account
4. Create a Function App
5. Deploy the code

### Test It

After deployment, visit the URL shown in the output:

```
Interactive UI: https://func-qr-xxxxx.azurewebsites.net/api/generate
Health check:   https://func-qr-xxxxx.azurewebsites.net/api/health
```

## Features

| Feature | Description |
|---------|-------------|
| 🔳 **QR Generation** | Generate QR codes from any text or URL |
| 🌐 **Interactive UI** | Beautiful web interface for manual use |
| 🔌 **REST API** | JSON API for programmatic access |
| 🔒 **Privacy First** | No data logging or retention |
| ☁️ **Azure Ready** | Deploys directly to Azure Functions |
| 📦 **Single Binary** | No external dependencies at runtime |

## API Reference

### GET /api/generate

Returns an interactive HTML page for generating QR codes in the browser.

### POST /api/generate

Generates a QR code and returns it as a base64-encoded PNG.

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
  "image": "iVBORw0KGgoAAAANSUhEUgAA...",
  "content": "https://example.com",
  "size": 256
}
```

**Parameters:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | string | Yes | Text or URL to encode |
| `size` | int | No | Image size in pixels (default: 256, max: 1024) |

### GET /api/health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "service": "qr-generator",
  "timestamp": "2025-12-25T12:00:00Z",
  "runtime": "Custom Handler (Go)"
}
```

## Deployment Options

### Hosting Plans

The deployment scripts support all Azure Functions hosting plans:

```powershell
# Consumption (default) - Pay per execution
.\deploy.ps1 -ResourceGroupName "rg-qr" -Location "eastus"

# Premium - Pre-warmed instances, VNET support
.\deploy.ps1 -ResourceGroupName "rg-qr" -Location "eastus" -Plan Premium -Sku EP1

# Dedicated - Run on App Service Plan
.\deploy.ps1 -ResourceGroupName "rg-qr" -Location "eastus" -Plan Dedicated -Sku B1
```

| Plan | SKUs | Best For |
|------|------|----------|
| Consumption | Y1 | Development, low-traffic |
| Premium | EP1, EP2, EP3 | Production, VNET, no cold starts |
| Dedicated | B1-P3v3 | Consistent workloads, App Service features |

### Manual Deployment

If you prefer manual deployment:

```powershell
# 1. Cross-compile for Linux
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o handler .

# 2. Create resources (use Azure Portal or CLI)

# 3. Deploy with --custom flag
func azure functionapp publish <your-function-app-name> --no-build --custom
```

> **Important:** Always use `--custom` flag when deploying Custom Handler apps.

## Local Development

```powershell
# Build
go build -o handler.exe .

# Run with Azure Functions Core Tools
func start
```

Then visit:
- Interactive UI: http://localhost:7071/api/generate
- Health check: http://localhost:7071/api/health

### Using curl

```bash
# Generate a QR code
curl -X POST http://localhost:7071/api/generate \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello World!", "size": 256}'

# Health check
curl http://localhost:7071/api/health
```

## Why Custom Handler?

This sample uses **Custom Handler** instead of gRPC because Azure Functions (PaaS) only supports languages on its runtime allowlist.

| Runtime | Status |
|---------|--------|
| dotnet, node, python, java, powershell | ✅ Built-in |
| custom (HTTP) | ✅ Supported |
| go (gRPC) | ❌ Not on allowlist |

**Custom Handler** works by:
1. Running your Go binary as an HTTP server
2. Azure Functions host forwards requests to your server
3. Your server processes and returns responses

For **gRPC with full binding support** (Queue, Blob, Timer triggers), see the 
[qr-generator-grpc](../qr-generator-grpc/) sample which deploys to Azure Container Apps.

### Comparison

| Aspect | Custom Handler (this sample) | gRPC (container) |
|--------|------------------------------|------------------|
| Deploy to | Azure Functions | Azure Container Apps |
| Triggers | HTTP only (practical) | All (HTTP, Queue, Blob, Timer) |
| Setup | Simple | More complex |
| Cold start | ~500ms-2s | ~2-5s |
| Dependencies | None | gRPC, protobuf |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `func azure functionapp publish` fails | Add `--custom` flag |
| 404 on function endpoints | Check function.json files exist |
| Binary not found | Ensure `handler` (Linux) binary exists, not `handler.exe` |
| CORS errors | Add CORS settings in Azure Portal |

### Logs

```bash
# View live logs
func azure functionapp logstream <app-name>

# View in Azure Portal
# Function App → Functions → Generate → Monitor
```

## Privacy

🔒 **Your data is never stored.**

- No input logging
- No data retention
- No cookies or tracking
- All processing in-memory
- Data discarded after response

## Project Structure

```
qr-generator-custom-handler/
├── main.go              # HTTP server with handlers
├── go.mod               # Go module
├── host.json            # Custom Handler configuration
├── local.settings.json  # Local development settings
├── deploy.ps1           # PowerShell deployment script
├── deploy.sh            # Bash deployment script
├── Generate/
│   └── function.json    # Generate function binding
└── Health/
    └── function.json    # Health function binding
```

## License

MIT License - See [LICENSE](../../LICENSE) for details.
