# Azure Functions Go Worker - Hello World Sample

A complete, deployable Azure Functions application written in Go.

## Prerequisites

- [Go 1.21+](https://golang.org/dl/)
- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
- [Azure Developer CLI (azd)](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd)
- [Azure Functions Core Tools v4](https://docs.microsoft.com/azure/azure-functions/functions-run-local)

## Quick Start

### One-Click Deploy to Azure

```bash
# Login to Azure
azd auth login

# Deploy everything (infrastructure + code)
azd up
```

That's it! The command will:
1. Create a resource group
2. Deploy Azure Storage, App Service Plan, Application Insights
3. Deploy the Function App with Custom Handler
4. Build your Go code for Linux
5. Deploy the binary to Azure

### Local Development

```bash
# Build for local testing
cd src
go build -o handler.exe .   # Windows
go build -o handler .       # Linux/Mac

# Run locally with Azure Functions Core Tools
func start
```

Then visit:
- http://localhost:7071/api/hello?name=World
- http://localhost:7071/api/health
- http://localhost:7071/api/echo?foo=bar

## Functions

| Function | Method | Description |
|----------|--------|-------------|
| `/api/hello` | GET, POST | Greets the user by name |
| `/api/health` | GET | Health check endpoint |
| `/api/echo` | GET, POST, PUT, DELETE | Echoes back request details |

## Project Structure

```
hello-world/
├── azure.yaml              # azd project configuration
├── infra/                  # Bicep templates for Azure resources
│   ├── main.bicep          # Main infrastructure template
│   ├── abbreviations.json  # Resource naming abbreviations
│   └── modules/
│       └── function-app.bicep  # Function App module
├── hooks/                  # azd lifecycle hooks
│   ├── prepackage.ps1      # Windows build script
│   └── prepackage.sh       # Linux/Mac build script
└── src/                    # Function App source code
    ├── main.go             # Go application code
    ├── go.mod              # Go module file
    ├── host.json           # Functions host configuration
    ├── local.settings.json # Local development settings
    ├── hello/              # Hello function
    │   └── function.json
    ├── health/             # Health function
    │   └── function.json
    └── echo/               # Echo function
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

## Useful Commands

```bash
# Deploy only code changes (faster)
azd deploy

# View deployed resources
azd show

# View logs
func azure functionapp logstream <app-name>

# Delete all resources
azd down
```

## Customization

### Adding a New Function

1. Create a new folder under `src/` (e.g., `src/myfunction/`)
2. Add `function.json` with bindings
3. Add handler in `main.go`
4. Redeploy with `azd deploy`

### Example: Timer Trigger

```json
// src/mytimer/function.json
{
  "bindings": [
    {
      "name": "timer",
      "type": "timerTrigger",
      "direction": "in",
      "schedule": "0 */5 * * * *"
    }
  ]
}
```

## Troubleshooting

**Build fails on Windows:**
Make sure you have Go installed and in your PATH.

**Deployment fails:**
Check `azd` is logged in: `azd auth login`

**Function not responding:**
Check Application Insights logs in Azure Portal.

## License

MIT License - see [LICENSE](../../LICENSE) for details.
