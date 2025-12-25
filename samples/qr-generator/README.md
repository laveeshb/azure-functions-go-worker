# QR Code Generator - Azure Functions Go Sample

A sample Azure Functions app that generates QR codes, built with the Go gRPC worker.

## Table of Contents

- [Features](#features)
- [Privacy](#privacy)
- [Prerequisites](#prerequisites)
- [Local Development](#local-development)
- [API Reference](#api-reference)
- [Deploy to Azure](#deploy-to-azure)
- [Using the QR Code](#using-the-qr-code)
- [Project Structure](#project-structure)
- [License](#license)

## Features

- **Interactive Web UI** - User-friendly landing page at `/api/generate`
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
- Azure subscription (for deployment)

## Local Development

### Build

```bash
# From the repository root
go build -o samples/qr-generator/worker.exe ./samples/qr-generator
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
http://localhost:7071/api/generate
```

**API - Generate a QR code:**

```bash
curl -X POST http://localhost:7071/api/generate \
  -H "Content-Type: application/json" \
  -d '{"content": "https://github.com/laveeshb/azure-functions-go-worker", "size": 256}'
```

**Health check:**

```bash
curl http://localhost:7071/api/health
```

## API Reference

### GET /api/generate

Serves an interactive web page where users can:
- Enter text or URL to encode
- Select QR code size
- Generate and preview the QR code
- Download the QR code as PNG

Simply open `http://localhost:7071/api/generate` in your browser.

### POST /api/generate

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

### GET /api/health

Health check endpoint.

**Response:**

```json
{
  "status": "healthy",
  "service": "qr-generator"
}
```

## Deploy to Azure

### Using Azure CLI

```bash
# Create a resource group
az group create --name qr-generator-rg --location eastus

# Create a storage account
az storage account create \
  --name qrgenstorage \
  --resource-group qr-generator-rg \
  --location eastus \
  --sku Standard_LRS

# Create a function app
az functionapp create \
  --name qr-generator-func \
  --resource-group qr-generator-rg \
  --storage-account qrgenstorage \
  --consumption-plan-location eastus \
  --runtime custom \
  --functions-version 4

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o samples/qr-generator/worker ./samples/qr-generator

# Deploy
cd samples/qr-generator
func azure functionapp publish qr-generator-func
```

## Using the QR Code

The API returns a base64-encoded PNG. To display it in HTML:

```html
<img src="data:image/png;base64,{image}" alt="QR Code" />
```

Or save it to a file (using jq and base64):

```bash
curl -s -X POST http://localhost:7071/api/generate \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello, World!"}' \
  | jq -r '.image' | base64 -d > qrcode.png
```

## Project Structure

```
qr-generator/
├── main.go              # Function handlers
├── go.mod               # Go module definition
├── host.json            # Azure Functions host configuration
├── local.settings.json  # Local development settings
├── worker.config.json   # Worker discovery config
├── Generate/
│   └── function.json    # Generate function binding
└── Health/
    └── function.json    # Health check function binding
```

## License

MIT - See [LICENSE](../../LICENSE) for details.
