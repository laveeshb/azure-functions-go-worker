# Azure Functions Go Worker - QR Generator Deploy Script
# Deploys a Go Function App to Azure using Custom Handler
#
# This sample uses Custom Handler (HTTP-based) which works with Azure Functions.
# For the gRPC version that deploys to Azure Container Apps, see ../qr-generator-grpc/
param(
    [Parameter(Mandatory=$true)]
    [string]$ResourceGroupName,
    
    [Parameter(Mandatory=$true)]
    [string]$Location,
    
    [string]$FunctionAppName = "",
    
    [ValidateSet("Consumption", "Premium", "Dedicated")]
    [string]$Plan = "Consumption",
    
    [string]$Sku = ""
)

$ErrorActionPreference = "Stop"

# Validate and set SKU based on plan
switch ($Plan) {
    "Consumption" { 
        if ($Sku -and $Sku -ne "Y1") {
            Write-Warning "Consumption plan only supports Y1 SKU, ignoring provided SKU"
        }
        $Sku = "Y1"
    }
    "Premium" { 
        if (-not $Sku) { $Sku = "EP1" }
        if ($Sku -notmatch "^EP[123]$") {
            throw "Premium plan requires SKU: EP1, EP2, or EP3. Got: $Sku"
        }
    }
    "Dedicated" { 
        if (-not $Sku) { $Sku = "B1" }
        $validSkus = @("B1","B2","B3","S1","S2","S3","P1v2","P2v2","P3v2","P1v3","P2v3","P3v3")
        if ($Sku -notin $validSkus) {
            throw "Dedicated plan requires valid App Service SKU. Got: $Sku. Valid: $($validSkus -join ', ')"
        }
    }
}

# Generate unique name if not provided
if (-not $FunctionAppName) {
    $suffix = -join ((48..57) + (97..122) | Get-Random -Count 8 | ForEach-Object {[char]$_})
    $FunctionAppName = "func-qr-$suffix"
}

$StorageAccountName = "stqr$(-join ((97..122) | Get-Random -Count 10 | ForEach-Object {[char]$_}))"

Write-Host "=== QR Generator - Custom Handler Deployment ===" -ForegroundColor Cyan
Write-Host "Resource Group: $ResourceGroupName" -ForegroundColor Gray
Write-Host "Location: $Location" -ForegroundColor Gray
Write-Host "Function App: $FunctionAppName" -ForegroundColor Gray
Write-Host "Storage Account: $StorageAccountName" -ForegroundColor Gray
Write-Host "Plan: $Plan ($Sku)" -ForegroundColor Gray
Write-Host ""

# Build the Go binary for Linux
Write-Host "Step 1: Building Go binary for Linux..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -o handler .
if ($LASTEXITCODE -ne 0) {
    throw "Failed to build Go binary"
}
Write-Host "  Done!" -ForegroundColor Green

# Create Resource Group
Write-Host "Step 2: Creating Resource Group..." -ForegroundColor Yellow
az group create --name $ResourceGroupName --location $Location --output none
Write-Host "  Done!" -ForegroundColor Green

# Create Storage Account
Write-Host "Step 3: Creating Storage Account..." -ForegroundColor Yellow
az storage account create `
    --name $StorageAccountName `
    --resource-group $ResourceGroupName `
    --location $Location `
    --sku Standard_LRS `
    --allow-blob-public-access false `
    --output none
Write-Host "  Done!" -ForegroundColor Green

# Create Function App based on plan type
Write-Host "Step 4: Creating Function App ($Plan plan)..." -ForegroundColor Yellow

if ($Plan -eq "Consumption") {
    az functionapp create `
        --name $FunctionAppName `
        --resource-group $ResourceGroupName `
        --storage-account $StorageAccountName `
        --consumption-plan-location $Location `
        --runtime custom `
        --functions-version 4 `
        --os-type Linux `
        --output none
}
elseif ($Plan -eq "Premium") {
    $planName = "$FunctionAppName-plan"
    az functionapp plan create `
        --name $planName `
        --resource-group $ResourceGroupName `
        --location $Location `
        --sku $Sku `
        --is-linux `
        --output none
    
    az functionapp create `
        --name $FunctionAppName `
        --resource-group $ResourceGroupName `
        --storage-account $StorageAccountName `
        --plan $planName `
        --runtime custom `
        --functions-version 4 `
        --os-type Linux `
        --output none
}
else {
    # Dedicated
    $planName = "$FunctionAppName-plan"
    az appservice plan create `
        --name $planName `
        --resource-group $ResourceGroupName `
        --location $Location `
        --sku $Sku `
        --is-linux `
        --output none
    
    az functionapp create `
        --name $FunctionAppName `
        --resource-group $ResourceGroupName `
        --storage-account $StorageAccountName `
        --plan $planName `
        --runtime custom `
        --functions-version 4 `
        --os-type Linux `
        --output none
}
Write-Host "  Done!" -ForegroundColor Green

# Deploy using func CLI
Write-Host "Step 5: Deploying to Azure..." -ForegroundColor Yellow
func azure functionapp publish $FunctionAppName --no-build --custom
Write-Host "  Done!" -ForegroundColor Green

# Get the URL
$url = az functionapp show --name $FunctionAppName --resource-group $ResourceGroupName --query "defaultHostName" -o tsv

Write-Host ""
Write-Host "=== Deployment Complete! ===" -ForegroundColor Green
Write-Host ""
Write-Host "Function App URL: https://$url" -ForegroundColor Cyan
Write-Host ""
Write-Host "Try it out:" -ForegroundColor Yellow
Write-Host "  Interactive UI: https://$url/api/generate"
Write-Host "  Health check:   curl https://$url/api/health"
Write-Host ""
Write-Host "API usage:" -ForegroundColor Yellow
Write-Host "  curl -X POST https://$url/api/generate \"
Write-Host "    -H 'Content-Type: application/json' \"
Write-Host "    -d '{\"content\": \"https://example.com\", \"size\": 256}'"
Write-Host ""
Write-Host "To delete all resources:" -ForegroundColor Yellow
Write-Host "  az group delete --name $ResourceGroupName --yes"
