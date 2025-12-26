<#
.SYNOPSIS
    Deploy the QR Code Generator to Azure Functions.

.DESCRIPTION
    This script creates all necessary Azure resources and deploys the Go-based
    QR Code Generator function app. It handles:
    - Resource group creation
    - Storage account creation
    - Function app creation with custom runtime
    - Cross-compilation for Linux
    - Deployment via Azure Functions Core Tools

.PARAMETER ResourceGroup
    Name of the Azure resource group (default: qr-generator-rg)

.PARAMETER Location
    Azure region for resources (default: eastus)

.PARAMETER FunctionAppName
    Name of the function app (must be globally unique)

.PARAMETER StorageAccountName
    Name of the storage account (must be globally unique, 3-24 chars, lowercase alphanumeric)

.PARAMETER Plan
    Hosting plan type. Options:
    - Consumption (default): Pay-per-execution, auto-scale, cold starts
    - Premium: No cold starts, VNET support, always-warm instances (EP1, EP2, EP3)
    - Dedicated: App Service Plan with reserved capacity (B1, S1, P1v2, etc.)

.PARAMETER Sku
    SKU for Premium or Dedicated plans:
    - Premium: EP1 (default), EP2, EP3
    - Dedicated: B1, B2, B3, S1, S2, S3, P1v2, P2v2, P3v2
    Ignored for Consumption plan.

.PARAMETER SkipBuild
    Skip the Go build step (use if already built)

.PARAMETER SkipResourceCreation
    Skip Azure resource creation (use for redeployment)

.EXAMPLE
    .\deploy.ps1 -FunctionAppName "myqrgen123"

.EXAMPLE
    .\deploy.ps1 -FunctionAppName "myqrgen123" -Location "westus2" -ResourceGroup "my-rg"

.EXAMPLE
    .\deploy.ps1 -FunctionAppName "myqrgen123" -SkipResourceCreation

.EXAMPLE
    .\deploy.ps1 -FunctionAppName "myqrgen123" -Plan Premium -Sku EP1

.EXAMPLE
    .\deploy.ps1 -FunctionAppName "myqrgen123" -Plan Dedicated -Sku S1

.NOTES
    Prerequisites:
    - Azure CLI installed and logged in (az login)
    - Azure Functions Core Tools v4
    - Go 1.21 or later
    - PowerShell 7+ recommended

    Author: Azure Functions Go Worker
    Version: 1.0.0
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $false)]
    [string]$ResourceGroup = "qr-generator-rg",

    [Parameter(Mandatory = $false)]
    [string]$Location = "eastus",

    [Parameter(Mandatory = $true, HelpMessage = "Globally unique name for the function app")]
    [string]$FunctionAppName,

    [Parameter(Mandatory = $false)]
    [string]$StorageAccountName = "",

    [Parameter(Mandatory = $false)]
    [ValidateSet("Consumption", "Premium", "Dedicated")]
    [string]$Plan = "Consumption",

    [Parameter(Mandatory = $false)]
    [string]$Sku = "",

    [switch]$SkipBuild,

    [switch]$SkipResourceCreation
)

# ============================================================================
# Configuration
# ============================================================================

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Generate storage account name if not provided
if ([string]::IsNullOrEmpty($StorageAccountName)) {
    # Create a valid storage account name (lowercase, alphanumeric, 3-24 chars)
    $StorageAccountName = ($FunctionAppName -replace '[^a-zA-Z0-9]', '').ToLower()
    if ($StorageAccountName.Length -gt 20) {
        $StorageAccountName = $StorageAccountName.Substring(0, 20)
    }
    $StorageAccountName = "${StorageAccountName}stor"
}

# Set default SKU based on plan
if ([string]::IsNullOrEmpty($Sku)) {
    switch ($Plan) {
        "Premium" { $Sku = "EP1" }
        "Dedicated" { $Sku = "B1" }
        default { $Sku = "" }  # Consumption doesn't need SKU
    }
}

# Validate SKU for the selected plan
$validPremiumSkus = @("EP1", "EP2", "EP3")
$validDedicatedSkus = @("B1", "B2", "B3", "S1", "S2", "S3", "P1v2", "P2v2", "P3v2", "P1v3", "P2v3", "P3v3")

if ($Plan -eq "Premium" -and $Sku -notin $validPremiumSkus) {
    Write-Host "Invalid SKU '$Sku' for Premium plan. Valid options: $($validPremiumSkus -join ', ')" -ForegroundColor Red
    exit 1
}
if ($Plan -eq "Dedicated" -and $Sku -notin $validDedicatedSkus) {
    Write-Host "Invalid SKU '$Sku' for Dedicated plan. Valid options: $($validDedicatedSkus -join ', ')" -ForegroundColor Red
    exit 1
}

# App Service Plan name (for Premium and Dedicated)
$AppServicePlanName = "${FunctionAppName}-plan"

# ============================================================================
# Helper Functions
# ============================================================================

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    Write-Host "  $Message" -ForegroundColor Cyan
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "  ✓ $Message" -ForegroundColor Green
}

function Write-Info {
    param([string]$Message)
    Write-Host "  ℹ $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "  ✗ $Message" -ForegroundColor Red
}

function Test-Command {
    param([string]$Command)
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

# ============================================================================
# Prerequisites Check
# ============================================================================

Write-Step "Checking Prerequisites"

# Check Azure CLI
if (-not (Test-Command "az")) {
    Write-Error "Azure CLI is not installed. Install from: https://docs.microsoft.com/cli/azure/install-azure-cli"
    exit 1
}
Write-Success "Azure CLI found"

# Check Azure login
$account = az account show 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Info "Not logged in to Azure. Running 'az login'..."
    az login
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to log in to Azure"
        exit 1
    }
}
$accountInfo = $account | ConvertFrom-Json
Write-Success "Logged in to Azure subscription: $($accountInfo.name)"

# Check Azure Functions Core Tools
if (-not (Test-Command "func")) {
    Write-Error "Azure Functions Core Tools not installed. Install from: https://docs.microsoft.com/azure/azure-functions/functions-run-local"
    exit 1
}
$funcVersion = func --version
Write-Success "Azure Functions Core Tools v$funcVersion"

# Check Go
if (-not (Test-Command "go")) {
    Write-Error "Go is not installed. Install from: https://golang.org/dl/"
    exit 1
}
$goVersion = go version
Write-Success "Go: $goVersion"

# ============================================================================
# Build for Linux
# ============================================================================

if (-not $SkipBuild) {
    Write-Step "Building Go Worker for Linux (Azure)"

    Push-Location $ScriptDir
    try {
        # Set environment for Linux cross-compilation
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $env:CGO_ENABLED = "0"

        Write-Info "Cross-compiling for linux/amd64..."
        go build -ldflags="-s -w" -o worker .

        if ($LASTEXITCODE -ne 0) {
            Write-Error "Go build failed"
            exit 1
        }

        # Verify the binary was created
        if (-not (Test-Path "worker")) {
            Write-Error "Worker binary was not created"
            exit 1
        }

        $fileSize = (Get-Item "worker").Length / 1MB
        Write-Success "Built worker binary ($([math]::Round($fileSize, 2)) MB)"
    }
    finally {
        # Reset environment
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
        Pop-Location
    }
}
else {
    Write-Step "Skipping Build (--SkipBuild specified)"
    if (-not (Test-Path "$ScriptDir\worker")) {
        Write-Error "No worker binary found. Run without -SkipBuild first."
        exit 1
    }
    Write-Success "Using existing worker binary"
}

# ============================================================================
# Create Azure Resources
# ============================================================================

if (-not $SkipResourceCreation) {
    Write-Step "Creating Azure Resources"

    # Create resource group
    Write-Info "Creating resource group: $ResourceGroup..."
    az group create `
        --name $ResourceGroup `
        --location $Location `
        --output none

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create resource group"
        exit 1
    }
    Write-Success "Resource group created: $ResourceGroup"

    # Create storage account
    Write-Info "Creating storage account: $StorageAccountName..."
    az storage account create `
        --name $StorageAccountName `
        --resource-group $ResourceGroup `
        --location $Location `
        --sku Standard_LRS `
        --allow-blob-public-access false `
        --output none

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create storage account"
        exit 1
    }
    Write-Success "Storage account created: $StorageAccountName"

    # Create function app based on plan type
    switch ($Plan) {
        "Consumption" {
            Write-Info "Creating function app: $FunctionAppName (Consumption plan)..."
            az functionapp create `
                --name $FunctionAppName `
                --resource-group $ResourceGroup `
                --storage-account $StorageAccountName `
                --consumption-plan-location $Location `
                --runtime custom `
                --functions-version 4 `
                --os-type Linux `
                --output none
        }
        "Premium" {
            Write-Info "Creating App Service Plan: $AppServicePlanName ($Sku)..."
            az functionapp plan create `
                --name $AppServicePlanName `
                --resource-group $ResourceGroup `
                --location $Location `
                --sku $Sku `
                --is-linux `
                --output none

            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed to create App Service Plan"
                exit 1
            }
            Write-Success "App Service Plan created: $AppServicePlanName ($Sku)"

            Write-Info "Creating function app: $FunctionAppName (Premium plan)..."
            az functionapp create `
                --name $FunctionAppName `
                --resource-group $ResourceGroup `
                --storage-account $StorageAccountName `
                --plan $AppServicePlanName `
                --runtime custom `
                --functions-version 4 `
                --os-type Linux `
                --output none
        }
        "Dedicated" {
            Write-Info "Creating App Service Plan: $AppServicePlanName ($Sku)..."
            az appservice plan create `
                --name $AppServicePlanName `
                --resource-group $ResourceGroup `
                --location $Location `
                --sku $Sku `
                --is-linux `
                --output none

            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed to create App Service Plan"
                exit 1
            }
            Write-Success "App Service Plan created: $AppServicePlanName ($Sku)"

            Write-Info "Creating function app: $FunctionAppName (Dedicated plan)..."
            az functionapp create `
                --name $FunctionAppName `
                --resource-group $ResourceGroup `
                --storage-account $StorageAccountName `
                --plan $AppServicePlanName `
                --runtime custom `
                --functions-version 4 `
                --os-type Linux `
                --output none
        }
    }

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create function app"
        exit 1
    }
    Write-Success "Function app created: $FunctionAppName"

    # Configure the function app
    Write-Info "Configuring function app settings..."
    az functionapp config appsettings set `
        --name $FunctionAppName `
        --resource-group $ResourceGroup `
        --settings "FUNCTIONS_WORKER_RUNTIME=custom" `
        --output none

    Write-Success "Function app configured"
}
else {
    Write-Step "Skipping Resource Creation (--SkipResourceCreation specified)"
    Write-Info "Using existing resources"
}

# ============================================================================
# Deploy
# ============================================================================

Write-Step "Deploying to Azure"

Push-Location $ScriptDir
try {
    Write-Info "Publishing function app..."

    # The --custom flag is required because Go is not a built-in Azure Functions runtime.
    # This tells func to look for a custom handler configuration.
    func azure functionapp publish $FunctionAppName --no-build --custom

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Deployment failed"
        exit 1
    }

    Write-Success "Deployment completed successfully!"
}
finally {
    Pop-Location
}

# ============================================================================
# Summary
# ============================================================================

Write-Step "Deployment Summary"

$functionUrl = "https://$FunctionAppName.azurewebsites.net"

Write-Host ""
Write-Host "  Your QR Code Generator is now live!" -ForegroundColor Green
Write-Host ""
Write-Host "  🌐 Web UI:        $functionUrl/api/generate" -ForegroundColor White
Write-Host "  📊 Health Check:  $functionUrl/api/health" -ForegroundColor White
Write-Host "  📦 Resource Group: $ResourceGroup" -ForegroundColor Gray
Write-Host "  📍 Location:       $Location" -ForegroundColor Gray
Write-Host "  💰 Plan:          $Plan$(if ($Sku) { " ($Sku)" } else { '' })" -ForegroundColor Gray
Write-Host ""
Write-Host "  Test with curl:" -ForegroundColor Yellow
Write-Host "  curl -X POST $functionUrl/api/generate \" -ForegroundColor Gray
Write-Host "    -H 'Content-Type: application/json' \" -ForegroundColor Gray
Write-Host "    -d '{\"content\": \"Hello from Azure!\", \"size\": 256}'" -ForegroundColor Gray
Write-Host ""

# Open in browser
$openBrowser = Read-Host "  Open in browser? (Y/n)"
if ($openBrowser -ne "n" -and $openBrowser -ne "N") {
    Start-Process "$functionUrl/api/generate"
}

Write-Host ""
Write-Host "  Done! 🎉" -ForegroundColor Green
Write-Host ""
