# Azure Functions Go Worker - QR Generator (gRPC) - Deploy to Azure Container Apps
#
# This script deploys the Go gRPC worker as a container to Azure Container Apps.
# This approach allows the full gRPC protocol since we bundle the Functions Host.
#
# For simpler Azure Functions deployment, use ../qr-generator-custom-handler/deploy.ps1
param(
    [Parameter(Mandatory=$true)]
    [string]$ResourceGroupName,
    
    [Parameter(Mandatory=$true)]
    [string]$Location,
    
    [string]$AppName = "",
    
    [string]$RegistryName = ""
)

$ErrorActionPreference = "Stop"

# Generate unique names if not provided
if (-not $AppName) {
    $suffix = -join ((48..57) + (97..122) | Get-Random -Count 6 | ForEach-Object {[char]$_})
    $AppName = "qr-grpc-$suffix"
}

if (-not $RegistryName) {
    $RegistryName = "acr$(-join ((97..122) | Get-Random -Count 10 | ForEach-Object {[char]$_}))"
}

$EnvironmentName = "$AppName-env"
$ImageName = "$RegistryName.azurecr.io/qr-generator-grpc:latest"

Write-Host "=== QR Generator (gRPC) - Azure Container Apps Deployment ===" -ForegroundColor Cyan
Write-Host "Resource Group: $ResourceGroupName" -ForegroundColor Gray
Write-Host "Location: $Location" -ForegroundColor Gray
Write-Host "Container App: $AppName" -ForegroundColor Gray
Write-Host "Container Registry: $RegistryName" -ForegroundColor Gray
Write-Host ""

# Check prerequisites
Write-Host "Step 1: Checking prerequisites..." -ForegroundColor Yellow
$dockerVersion = docker --version 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "Docker is required but not installed. Please install Docker Desktop."
}
Write-Host "  Docker: $dockerVersion" -ForegroundColor Gray
Write-Host "  Done!" -ForegroundColor Green

# Create Resource Group
Write-Host "Step 2: Creating Resource Group..." -ForegroundColor Yellow
az group create --name $ResourceGroupName --location $Location --output none
Write-Host "  Done!" -ForegroundColor Green

# Create Azure Container Registry
Write-Host "Step 3: Creating Azure Container Registry..." -ForegroundColor Yellow
az acr create `
    --name $RegistryName `
    --resource-group $ResourceGroupName `
    --location $Location `
    --sku Basic `
    --admin-enabled true `
    --output none
Write-Host "  Done!" -ForegroundColor Green

# Build the Docker image (from repository root)
Write-Host "Step 4: Building Docker image..." -ForegroundColor Yellow
Push-Location "$PSScriptRoot/../.."
docker build -t qr-generator-grpc -f samples/qr-generator-grpc/Dockerfile .
if ($LASTEXITCODE -ne 0) {
    Pop-Location
    throw "Failed to build Docker image"
}
Pop-Location
Write-Host "  Done!" -ForegroundColor Green

# Login to ACR
Write-Host "Step 5: Logging into Container Registry..." -ForegroundColor Yellow
az acr login --name $RegistryName
Write-Host "  Done!" -ForegroundColor Green

# Tag and push the image
Write-Host "Step 6: Pushing image to registry..." -ForegroundColor Yellow
docker tag qr-generator-grpc $ImageName
docker push $ImageName
if ($LASTEXITCODE -ne 0) {
    throw "Failed to push Docker image"
}
Write-Host "  Done!" -ForegroundColor Green

# Get ACR credentials
Write-Host "Step 7: Getting registry credentials..." -ForegroundColor Yellow
$acrPassword = az acr credential show --name $RegistryName --query "passwords[0].value" -o tsv
Write-Host "  Done!" -ForegroundColor Green

# Create Container Apps Environment
Write-Host "Step 8: Creating Container Apps Environment..." -ForegroundColor Yellow
az containerapp env create `
    --name $EnvironmentName `
    --resource-group $ResourceGroupName `
    --location $Location `
    --output none
Write-Host "  Done!" -ForegroundColor Green

# Create Container App
Write-Host "Step 9: Creating Container App..." -ForegroundColor Yellow
az containerapp create `
    --name $AppName `
    --resource-group $ResourceGroupName `
    --environment $EnvironmentName `
    --image $ImageName `
    --target-port 80 `
    --ingress external `
    --registry-server "$RegistryName.azurecr.io" `
    --registry-username $RegistryName `
    --registry-password $acrPassword `
    --min-replicas 0 `
    --max-replicas 10 `
    --cpu 0.5 `
    --memory 1.0Gi `
    --output none
Write-Host "  Done!" -ForegroundColor Green

# Get the URL
$url = az containerapp show `
    --name $AppName `
    --resource-group $ResourceGroupName `
    --query "properties.configuration.ingress.fqdn" -o tsv

Write-Host ""
Write-Host "=== Deployment Complete! ===" -ForegroundColor Green
Write-Host ""
Write-Host "Container App URL: https://$url" -ForegroundColor Cyan
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
Write-Host "View logs:" -ForegroundColor Yellow
Write-Host "  az containerapp logs show --name $AppName --resource-group $ResourceGroupName --follow"
Write-Host ""
Write-Host "To delete all resources:" -ForegroundColor Yellow
Write-Host "  az group delete --name $ResourceGroupName --yes"
