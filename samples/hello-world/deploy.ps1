# Azure Functions Go Worker - Simple Deploy Script
# Alternative to azd for quick deployment using Azure CLI
param(
    [Parameter(Mandatory=$true)]
    [string]$ResourceGroupName,
    
    [Parameter(Mandatory=$true)]
    [string]$Location,
    
    [string]$FunctionAppName = ""
)

$ErrorActionPreference = "Stop"

# Generate unique name if not provided
if (-not $FunctionAppName) {
    $suffix = -join ((48..57) + (97..122) | Get-Random -Count 8 | ForEach-Object {[char]$_})
    $FunctionAppName = "func-go-$suffix"
}

$StorageAccountName = "stgo$(-join ((97..122) | Get-Random -Count 10 | ForEach-Object {[char]$_}))"

Write-Host "=== Azure Functions Go Worker Deployment ===" -ForegroundColor Cyan
Write-Host "Resource Group: $ResourceGroupName" -ForegroundColor Gray
Write-Host "Location: $Location" -ForegroundColor Gray
Write-Host "Function App: $FunctionAppName" -ForegroundColor Gray
Write-Host "Storage Account: $StorageAccountName" -ForegroundColor Gray
Write-Host ""

# Build the Go binary for Linux
Write-Host "Step 1: Building Go binary for Linux..." -ForegroundColor Yellow
Push-Location src
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o handler .
if ($LASTEXITCODE -ne 0) {
    Pop-Location
    throw "Failed to build Go binary"
}
Pop-Location
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
    --output none
Write-Host "  Done!" -ForegroundColor Green

# Create Function App
Write-Host "Step 4: Creating Function App..." -ForegroundColor Yellow
az functionapp create `
    --name $FunctionAppName `
    --resource-group $ResourceGroupName `
    --storage-account $StorageAccountName `
    --consumption-plan-location $Location `
    --runtime custom `
    --functions-version 4 `
    --os-type Linux `
    --output none
Write-Host "  Done!" -ForegroundColor Green

# Create deployment package
Write-Host "Step 5: Creating deployment package..." -ForegroundColor Yellow
$zipPath = Join-Path $env:TEMP "go-func-deploy.zip"
if (Test-Path $zipPath) { Remove-Item $zipPath }

Push-Location src
Compress-Archive -Path @(
    "handler",
    "host.json",
    "hello",
    "health", 
    "echo"
) -DestinationPath $zipPath
Pop-Location
Write-Host "  Done!" -ForegroundColor Green

# Deploy the package
Write-Host "Step 6: Deploying to Azure..." -ForegroundColor Yellow
az functionapp deployment source config-zip `
    --name $FunctionAppName `
    --resource-group $ResourceGroupName `
    --src $zipPath `
    --output none
Write-Host "  Done!" -ForegroundColor Green

# Get the URL
$url = az functionapp show --name $FunctionAppName --resource-group $ResourceGroupName --query "defaultHostName" -o tsv

Write-Host ""
Write-Host "=== Deployment Complete! ===" -ForegroundColor Green
Write-Host ""
Write-Host "Function App URL: https://$url" -ForegroundColor Cyan
Write-Host ""
Write-Host "Test your functions:" -ForegroundColor Yellow
Write-Host "  curl https://$url/api/hello?name=World"
Write-Host "  curl https://$url/api/health"
Write-Host "  curl https://$url/api/echo?foo=bar"
Write-Host ""
Write-Host "To delete all resources:" -ForegroundColor Yellow
Write-Host "  az group delete --name $ResourceGroupName --yes"
