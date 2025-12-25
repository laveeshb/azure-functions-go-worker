# Azure Functions Go Worker - Hello World Sample

A complete, deployable Azure Functions application written in Go using Custom Handlers.

## Prerequisites

- [Go 1.21+](https://golang.org/dl/)
- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
- [Azure Functions Core Tools v4](https://docs.microsoft.com/azure/azure-functions/functions-run-local) - for local testing
- An Azure subscription with permissions to create resources

## Quick Start

### Deploy with Azure CLI Scripts

These scripts handle cross-compilation and deployment automatically:

```powershell
# PowerShell (Windows)
cd samples/hello-world
.\deploy.ps1 -ResourceGroupName "rg-gofunc" -Location "eastus"
```

```bash
# Bash (Linux/Mac)
cd samples/hello-world
./deploy.sh -g "rg-gofunc" -l "eastus"
```

These scripts auto-generate unique names for the Function App and Storage Account.

### Local Development

```powershell
# Windows - Build and run locally
cd src
go build -o handler.exe .
func start
```

```bash
# Linux/Mac - Build and run locally
cd src
go build -o handler .
func start
```

Then visit:
- http://localhost:7071/api/hello?name=World
- http://localhost:7071/api/health
- http://localhost:7071/api/echo?foo=bar

### Manual Deployment (Cross-Compilation)

If deploying manually without the scripts, you must cross-compile for Linux since Azure Functions Consumption Plan runs on Linux:

```powershell
# Windows PowerShell - Cross-compile for Linux
cd src
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o handler .
func azure functionapp publish <your-function-app-name> --no-build
```

```bash
# Linux/Mac - Build for Linux
cd src
GOOS=linux GOARCH=amd64 go build -o handler .
func azure functionapp publish <your-function-app-name> --no-build
```

**Important:** The binary must be named `handler` (not `handler.exe`) to match the `host.json` configuration.

## Functions

| Function | Method | Description |
|----------|--------|-------------|
| `/api/hello` | GET, POST | Greets the user by name |
| `/api/health` | GET | Health check endpoint |
| `/api/echo` | GET, POST, PUT, DELETE | Echoes back request details |

## Project Structure

```
hello-world/
├── deploy.ps1          # Windows deployment script
├── deploy.sh           # Linux/Mac deployment script
└── src/                # Function App source code
    ├── main.go         # Go application code
    ├── go.mod          # Go module file
    ├── host.json       # Functions host configuration
    ├── local.settings.json # Local development settings
    ├── hello/          # Hello function
    │   └── function.json
    ├── health/         # Health function
    │   └── function.json
    └── echo/           # Echo function
        └── function.json
```

## Azure Resources Created

| Resource | Description |
|----------|-------------|
| Resource Group | Container for all resources |
| Storage Account | Required for Functions runtime |
| App Service Plan | Consumption (serverless) plan |
| Function App | Custom Handler with Go binary |
| Application Insights | Monitoring and logging |

## What Gets Deployed

When you deploy, the following is packaged and uploaded to Azure:

```
📦 Deployment Package
├── handler              ← Compiled Go binary (Linux amd64)
├── host.json            ← Functions host configuration
├── hello/               ← Hello function definition
│   └── function.json
├── health/              ← Health function definition
│   └── function.json
└── echo/                ← Echo function definition
    └── function.json
```

### Cross-Compilation

Since Azure Functions Consumption Plan runs on **Linux**, the Go binary must be compiled for Linux even when building on Windows or Mac:

```powershell
# Windows (PowerShell)
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o handler .
```

```bash
# Linux/Mac
GOOS=linux GOARCH=amd64 go build -o handler .
```

This is handled automatically by the deploy scripts (`deploy.ps1` / `deploy.sh`).

### No Go Runtime Required on Azure

The compiled binary is **self-contained**. Go does not need to be installed on Azure - the binary includes everything it needs to run.

## Customization

### Adding a New Function

1. Create a new folder under `src/` (e.g., `src/myfunction/`)
2. Add `function.json` with bindings
3. Add handler in `main.go`
4. Redeploy

## Troubleshooting

**Build fails on Windows:**
Make sure you have Go installed and in your PATH:
```powershell
$env:Path += ";C:\Program Files\Go\bin"
go version
```

**Function returns 500 error after deployment:**
1. Check the binary was built for Linux: `file handler` should show "ELF 64-bit LSB executable"
2. Verify `host.json` has correct `defaultExecutablePath`: should be `"handler"` (not `"handler.exe"`)
3. Check Application Insights logs in Azure Portal

**Function not responding:**
1. Check the Function App is running in Azure Portal
2. View live logs: `func azure functionapp logstream <app-name>`
3. Check Application Insights for errors

**Local testing fails:**
1. Build with correct extension: `handler.exe` on Windows, `handler` on Linux/Mac
2. Ensure `host.json` points to correct binary name for your OS
3. Check port 7071 is not in use: `netstat -an | findstr 7071`

## License

MIT License - see [LICENSE](../../LICENSE) for details.
