# QR Code Generator - Azure Functions Go Sample

A sample Azure Functions app that generates QR codes, built with Go Custom Handlers.

## Table of Contents

- [Features](#features)
- [Privacy](#privacy)
- [Prerequisites](#prerequisites)
- [How It Works](#how-it-works)
- [Local Development](#local-development)
- [API Reference](#api-reference)
- [Deploy to Azure](#deploy-to-azure)
  - [Quick Start (Turnkey Script)](#quick-start-turnkey-script)
  - [Script Options](#script-options)
  - [Hosting Plans](#hosting-plans)
  - [Manual Deployment](#manual-deployment)
  - [Redeployment](#redeployment)
  - [Clean Up](#clean-up)
  - [Troubleshooting](#troubleshooting)
- [Using the QR Code](#using-the-qr-code)
- [Project Structure](#project-structure)
- [License](#license)

## Features

- **Interactive Web UI** - User-friendly landing page at `/generate`
- Generate QR codes from any text or URL
- Configurable image size (up to 1024px)
- Download generated QR codes as PNG
- Returns base64-encoded PNG images via API
- Health check endpoint

## Privacy

🔒 **Your data is never stored.**

This application does not log, save, or transmit any user data to third parties. All QR code generation happens in-memory on the server and your data is immediately discarded after the response is sent.

- No cookies
- No tracking
- No data retention
- No analytics

## Prerequisites

- Go 1.21 or later
- Azure Functions Core Tools v4
- Azure CLI (for deployment)
- Azure subscription (for deployment)

## How It Works

This sample uses the [Custom Handler](https://learn.microsoft.com/azure/azure-functions/functions-custom-handlers) pattern:

1. The Go binary runs as an HTTP server
2. Azure Functions Host forwards HTTP requests to the Go binary
3. The `host.json` configures the binary path and settings

```json
{
  "customHandler": {
    "description": {
      "defaultExecutablePath": "handler"
    },
    "enableForwardingHttpRequest": true
  }
}
```

## Local Development

### Build

```powershell
# Windows
cd samples/qr-generator
go build -o handler.exe .
```

```bash
# Linux/Mac
cd samples/qr-generator
go build -o handler .
```

### Run

```bash
cd samples/qr-generator
func start
```

### Test

**Web UI (recommended):**

Open your browser and navigate to:
```
http://localhost:7071/generate
```

**API - Generate a QR code:**

```bash
curl -X POST http://localhost:7071/generate \
  -H "Content-Type: application/json" \
  -d '{"content": "https://github.com/laveeshb/azure-functions-go-worker", "size": 256}'
```

**Health check:**

```bash
curl http://localhost:7071/health
```

## API Reference

### GET /generate

Serves an interactive web page where users can:
- Enter text or URL to encode
- Select QR code size
- Generate and preview the QR code
- Download the QR code as PNG

Simply open `http://localhost:7071/generate` in your browser.

### POST /generate

Generate a QR code from text or URL.

**Request Body:**

```json
{
  "content": "https://example.com",
  "size": 256
}
```

| Field     | Type   | Required | Description                              |
|-----------|--------|----------|------------------------------------------|
| `content` | string | Yes      | The text or URL to encode                |
| `size`    | int    | No       | Image size in pixels (default: 256, max: 1024) |

**Response:**

```json
{
  "image": "iVBORw0KGgoAAAANSUhEUg...",
  "content": "https://example.com",
  "size": 256
}
```

| Field     | Type   | Description                        |
|-----------|--------|------------------------------------|
| `image`   | string | Base64-encoded PNG image           |
| `content` | string | The original content that was encoded |
| `size`    | int    | The size of the generated image    |

### GET /health

Health check endpoint.

**Response:**

```json
{
  "status": "healthy",
  "service": "qr-generator"
}
```

## Deploy to Azure

### Quick Start (Turnkey Script)

The easiest way to deploy is using the provided deployment scripts:

**PowerShell (Windows):**

```powershell
cd samples/qr-generator
.\deploy.ps1 -FunctionAppName "my-qr-generator-123"
```

**Bash (Linux/macOS):**

```bash
cd samples/qr-generator
chmod +x deploy.sh
./deploy.sh -n my-qr-generator-123
```

The scripts handle everything: prerequisites check, cross-compilation, Azure resource creation, and deployment.

### Script Options

| Option | PowerShell | Bash | Description |
|--------|------------|------|-------------|
| Function App Name | `-FunctionAppName` | `-n, --name` | **Required.** Globally unique name |
| Resource Group | `-ResourceGroup` | `-g, --resource-group` | Default: `qr-generator-rg` |
| Location | `-Location` | `-l, --location` | Default: `eastus` |
| Storage Account | `-StorageAccountName` | `-s, --storage` | Auto-generated if not specified |
| Plan | `-Plan` | `-p, --plan` | `Consumption` (default), `Premium`, `Dedicated` |
| SKU | `-Sku` | `--sku` | SKU for Premium/Dedicated plans |
| Skip Build | `-SkipBuild` | `--skip-build` | Use existing binary |
| Skip Resources | `-SkipResourceCreation` | `--skip-resources` | Redeploy only |

### Hosting Plans

| Plan | SKU Options | Pricing | Cold Start | Use Case |
|------|-------------|---------|------------|----------|
| **Consumption** | N/A (Dynamic) | Pay-per-execution | Yes (10-30s) | Low traffic, cost-sensitive |
| **Premium** | EP1, EP2, EP3 | ~$150-600/month | No | Production, low latency |
| **Dedicated** | B1, S1, P1v2, etc. | ~$50-500/month | No | Predictable workloads |

**Examples:**

```powershell
# Consumption (default) - free tier covers 1M requests/month
.\deploy.ps1 -FunctionAppName "myqrgen123"

# Premium EP1 - no cold starts, VNET integration
.\deploy.ps1 -FunctionAppName "myqrgen123" -Plan Premium -Sku EP1

# Dedicated S1 - reserved App Service capacity
.\deploy.ps1 -FunctionAppName "myqrgen123" -Plan Dedicated -Sku S1
```

```bash
# Consumption (default)
./deploy.sh -n myqrgen123

# Premium EP1
./deploy.sh -n myqrgen123 -p premium --sku EP1

# Dedicated S1
./deploy.sh -n myqrgen123 -p dedicated --sku S1
```

### Manual Deployment

If you prefer to deploy manually, follow these steps:

#### Step 1: Prerequisites

Ensure you have these tools installed:

```bash
# Check Azure CLI
az --version

# Check Azure Functions Core Tools
func --version

# Check Go
go version
```

#### Step 2: Login to Azure

```bash
az login
az account set --subscription "Your Subscription Name"
```

#### Step 3: Create Azure Resources

```bash
# Set variables (customize these)
RESOURCE_GROUP="qr-generator-rg"
LOCATION="eastus"
FUNCTION_APP="qr-generator-$(openssl rand -hex 4)"  # Must be globally unique
STORAGE_ACCOUNT="${FUNCTION_APP//[^a-z0-9]/}stor"

# Create resource group
az group create \
  --name $RESOURCE_GROUP \
  --location $LOCATION

# Create storage account
az storage account create \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --sku Standard_LRS

# Create function app with custom runtime
az functionapp create \
  --name $FUNCTION_APP \
  --resource-group $RESOURCE_GROUP \
  --storage-account $STORAGE_ACCOUNT \
  --consumption-plan-location $LOCATION \
  --runtime custom \
  --functions-version 4 \
  --os-type Linux

# Configure runtime
az functionapp config appsettings set \
  --name $FUNCTION_APP \
  --resource-group $RESOURCE_GROUP \
  --settings "FUNCTIONS_WORKER_RUNTIME=custom"
```

#### Step 4: Build for Linux

Azure Functions runs on Linux by default. Cross-compile from any OS:

**Windows (PowerShell):**

```powershell
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o handler .
```

**Linux/macOS:**

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o handler .
```

> **Note:** The `-ldflags="-s -w"` flags strip debug information, reducing binary size by ~30%.

#### Step 5: Deploy

```bash
cd samples/qr-generator
# The --custom flag is required because Go is not a built-in runtime
func azure functionapp publish $FUNCTION_APP --no-build --custom
```

#### Step 6: Test Your Deployment

```bash
# Health check
curl https://$FUNCTION_APP.azurewebsites.net/health

# Generate a QR code
curl -X POST https://$FUNCTION_APP.azurewebsites.net/generate \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello from Azure!", "size": 256}'

# Or open the web UI in your browser
echo "https://$FUNCTION_APP.azurewebsites.net/generate"
```

### Redeployment

After making code changes, redeploy with:

```bash
# Rebuild and redeploy
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o handler .
func azure functionapp publish $FUNCTION_APP --no-build

# Or use the script
./deploy.sh -n $FUNCTION_APP --skip-resources
```

### Clean Up

To delete all Azure resources:

```bash
az group delete --name qr-generator-rg --yes --no-wait
```

### Troubleshooting

| Issue | Solution |
|-------|----------|
| "Function app not found" | Ensure the function app name is globally unique |
| "Can't determine project language" | Add `--custom` flag: `func azure functionapp publish <name> --custom` |
| "Worker runtime cannot be 'None'" | Set app setting: `FUNCTIONS_WORKER_RUNTIME=custom` |
| Binary won't start | Verify you built for Linux (`GOOS=linux GOARCH=amd64`) |
| Functions not discovered | Check `function.json` files exist in subdirectories |
| 500 errors | Check logs: `func azure functionapp logstream $FUNCTION_APP` |
| Cold start slow | First request after idle may take 10-30s on Consumption plan |
| Storage account policy error | Your subscription may require `--allow-blob-public-access false` |

## Using the QR Code

The API returns a base64-encoded PNG. To display it in HTML:

```html
<img src="data:image/png;base64,{image}" alt="QR Code" />
```

Or save it to a file (using jq and base64):

```bash
curl -s -X POST http://localhost:7071/generate \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello, World!"}' \
  | jq -r '.image' | base64 -d > qrcode.png
```

## Project Structure

```
qr-generator/
├── main.go              # Function handlers
├── go.mod               # Go module definition
├── go.sum               # Go dependencies checksum
├── host.json            # Azure Functions host configuration
├── local.settings.json  # Local development settings
├── deploy.ps1           # PowerShell deployment script (Windows)
├── deploy.sh            # Bash deployment script (Linux/macOS)
├── Generate/
│   └── function.json    # Generate function binding
├── Health/
│   └── function.json    # Health check function binding
└── Root/
    └── function.json    # Root path handler
```

## License

MIT - See [LICENSE](../../LICENSE) for details.
