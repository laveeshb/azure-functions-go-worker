# Azure Functions Go Worker

Run Go functions on Azure Functions using the [Custom Handler](https://learn.microsoft.com/en-us/azure/azure-functions/functions-custom-handlers) pattern.

## Overview

Custom Handlers allow you to write Azure Functions in any language that can run an HTTP server. This project provides samples and patterns for writing Go functions that run on Azure Functions.

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
)

func main() {
    listenAddr := ":8080"
    if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
        listenAddr = ":" + val
    }

    http.HandleFunc("/api/HelloWorld", helloHandler)
    http.ListenAndServe(listenAddr, nil)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    if name == "" {
        name = "World"
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "message": fmt.Sprintf("Hello, %s!", name),
    })
}
```

## Features

- **Pure Go** - Use standard `net/http`, no special SDK required
- **Fast cold starts** - Compiled Go binaries start in milliseconds
- **Azure Functions hosting** - Deploy to Consumption, Premium, or Dedicated plans
- **Simple deployment** - Single binary + `host.json` + function metadata

## Samples

| Sample | Description |
|--------|-------------|
| [hello-world](samples/hello-world/) | Basic "Hello World" HTTP function |
| [qr-generator](samples/qr-generator/) | QR code generator with health check endpoint |

## Quick Start

Follow these steps in order:

### Step 1: Install Prerequisites

```powershell
# Windows - run from repo root
.\scripts\install-prereqs.ps1 -IncludeAzureCLI
```

```bash
# Linux/Mac - run from repo root
chmod +x ./scripts/install-prereqs.sh
./scripts/install-prereqs.sh --with-az
```

This checks for and installs:
- Go 1.21+
- Azure Functions Core Tools v4
- Azure CLI (optional, for deployment)

### Step 2: Build

```powershell
# Navigate to a sample
cd samples/qr-generator

# Build for local development (Windows)
go build -o handler.exe .
```

### Step 3: Run Locally

```powershell
func start
```

Then visit: http://localhost:7071/api/health

### Step 4: Deploy to Azure

```powershell
# Deploy with the included script
.\deploy.ps1 -ResourceGroupName "rg-my-functions" -Location "eastus"
```

The deploy script will:
1. Build the Go binary for Linux
2. Create Azure resources (resource group, storage account, function app)
3. Deploy the function app

---

## Detailed Guide

### Prerequisites

- Go 1.21 or later
- [Azure Functions Core Tools v4](https://learn.microsoft.com/en-us/azure/azure-functions/functions-run-local)
- Azure subscription (for deployment)
- Azure CLI (for deployment)

### Local Development

```bash
# Clone the repository
git clone https://github.com/laveeshb/azure-functions-go-worker.git
cd azure-functions-go-worker

# Install prerequisites
.\scripts\install-prereqs.ps1    # Windows
./scripts/install-prereqs.sh     # Linux/Mac

# Navigate to a sample
cd samples/hello-world/src

# Build the Go binary
go build -o handler.exe .    # Windows
go build -o handler .        # Linux/Mac

# Run locally
func start
```

Then visit: http://localhost:7071/api/HelloWorld?name=Gopher

### Project Structure

```
samples/hello-world/
├── deploy.ps1             # Windows deployment script
├── deploy.sh              # Linux/Mac deployment script
├── README.md              # Sample-specific documentation
└── src/                   # Function App source code
    ├── main.go            # Go HTTP server
    ├── go.mod             # Go module file
    ├── host.json          # Custom Handler configuration
    ├── local.settings.json
    ├── hello/
    │   └── function.json  # Hello function trigger
    ├── health/
    │   └── function.json  # Health check trigger
    └── echo/
        └── function.json  # Echo function trigger
```

### Key Files

**host.json** - Configure the Custom Handler:
```json
{
  "version": "2.0",
  "customHandler": {
    "description": {
      "defaultExecutablePath": "handler",
      "workingDirectory": "",
      "arguments": []
    },
    "enableForwardingHttpRequest": true
  }
}
```

**function.json** - Define the function trigger:
```json
{
  "bindings": [
    {
      "authLevel": "anonymous",
      "type": "httpTrigger",
      "direction": "in",
      "name": "req",
      "methods": ["get", "post"],
      "route": "HelloWorld"
    },
    {
      "type": "http",
      "direction": "out",
      "name": "res"
    }
  ]
}
```

## Deploy to Azure

### Quick Deploy

```powershell
cd samples/qr-generator
.\deploy.ps1 -ResourceGroupName "rg-my-functions" -Location "eastus"
```

### Manual Deployment

```powershell
# 1. Build for Linux
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o handler .

# 2. Create Azure resources
az group create --name rg-my-functions --location eastus
az functionapp create --name my-go-func --resource-group rg-my-functions `
    --storage-account mystorageacct --consumption-plan-location eastus `
    --runtime custom --functions-version 4

# 3. Deploy
func azure functionapp publish my-go-func
```

## How Custom Handlers Work

```
HTTP Request → Azure Functions Host → Custom Handler (Go binary) → Your Code
                       ↓
              function.json defines
              triggers and bindings
```

1. The Azure Functions Host receives incoming requests
2. It routes them to your Go binary based on `function.json` metadata
3. Your Go code handles the request using standard `net/http`
4. Response flows back through the Host

With `enableForwardingHttpRequest: true`, HTTP requests are forwarded as-is to your handler, making it easy to use standard Go HTTP patterns.

## Limitations

Custom Handlers work great for **HTTP triggers**. For other trigger types (Queue, Blob, Timer), the request/response format is different - see [Microsoft's documentation](https://learn.microsoft.com/en-us/azure/azure-functions/functions-custom-handlers#request-payload).

For typical HTTP APIs and webhooks, Custom Handlers provide an excellent developer experience.

## Related Resources

- [Azure Functions Custom Handlers](https://learn.microsoft.com/en-us/azure/azure-functions/functions-custom-handlers)
- [Azure Functions Core Tools](https://learn.microsoft.com/en-us/azure/azure-functions/functions-run-local)
- [Azure Functions Host](https://github.com/Azure/azure-functions-host)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
