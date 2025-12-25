# QR Code Generator - Azure Functions Go Sample

A sample Azure Functions app that generates QR codes, built with the Go worker.

## Features

- Generate QR codes from any text or URL
- Configurable image size (up to 1024px)
- Returns base64-encoded PNG images
- Health check endpoint

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

Generate a QR code:

```bash
curl -X POST http://localhost:7071/api/generate \
  -H "Content-Type: application/json" \
  -d '{"content": "https://github.com/laveeshb/azure-functions-go-worker", "size": 256}'
```

Health check:

```bash
curl http://localhost:7071/api/health
```

## API Reference

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
├── host.json            # Azure Functions host configuration
├── local.settings.json  # Local development settings
├── README.md            # This file
├── Generate/
│   └── function.json    # Generate function binding
└── Health/
    └── function.json    # Health check function binding
```

## License

MIT - See [LICENSE](../../LICENSE) for details.
