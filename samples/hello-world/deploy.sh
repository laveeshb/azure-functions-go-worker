#!/bin/bash
# Azure Functions Go Worker - Simple Deploy Script
# Alternative to azd for quick deployment using Azure CLI
set -e

# Parse arguments
RESOURCE_GROUP=""
LOCATION=""
FUNCTION_APP_NAME=""

usage() {
    echo "Usage: $0 -g <resource-group> -l <location> [-n <function-app-name>]"
    echo ""
    echo "Options:"
    echo "  -g    Resource group name (required)"
    echo "  -l    Azure location (required, e.g., eastus, westus2)"
    echo "  -n    Function app name (optional, auto-generated if not provided)"
    exit 1
}

while getopts "g:l:n:" opt; do
    case $opt in
        g) RESOURCE_GROUP="$OPTARG" ;;
        l) LOCATION="$OPTARG" ;;
        n) FUNCTION_APP_NAME="$OPTARG" ;;
        *) usage ;;
    esac
done

if [ -z "$RESOURCE_GROUP" ] || [ -z "$LOCATION" ]; then
    usage
fi

# Generate unique names if not provided
if [ -z "$FUNCTION_APP_NAME" ]; then
    FUNCTION_APP_NAME="func-go-$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 8 | head -n 1)"
fi
STORAGE_ACCOUNT_NAME="stgo$(cat /dev/urandom | tr -dc 'a-z' | fold -w 10 | head -n 1)"

echo "=== Azure Functions Go Worker Deployment ==="
echo "Resource Group: $RESOURCE_GROUP"
echo "Location: $LOCATION"
echo "Function App: $FUNCTION_APP_NAME"
echo "Storage Account: $STORAGE_ACCOUNT_NAME"
echo ""

# Build the Go binary for Linux
echo "Step 1: Building Go binary for Linux..."
cd src
GOOS=linux GOARCH=amd64 go build -o handler .
cd ..
echo "  Done!"

# Create Resource Group
echo "Step 2: Creating Resource Group..."
az group create --name "$RESOURCE_GROUP" --location "$LOCATION" --output none
echo "  Done!"

# Create Storage Account
echo "Step 3: Creating Storage Account..."
az storage account create \
    --name "$STORAGE_ACCOUNT_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --location "$LOCATION" \
    --sku Standard_LRS \
    --allow-blob-public-access false \
    --output none
echo "  Done!"

# Create Function App
echo "Step 4: Creating Function App..."
az functionapp create \
    --name "$FUNCTION_APP_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --storage-account "$STORAGE_ACCOUNT_NAME" \
    --consumption-plan-location "$LOCATION" \
    --runtime custom \
    --functions-version 4 \
    --os-type Linux \
    --output none
echo "  Done!"

# Deploy using func CLI (handles permissions correctly)
echo "Step 5: Deploying to Azure..."
cd src
func azure functionapp publish "$FUNCTION_APP_NAME" --no-build
cd ..
echo "  Done!"

# Get the URL
URL=$(az functionapp show --name "$FUNCTION_APP_NAME" --resource-group "$RESOURCE_GROUP" --query "defaultHostName" -o tsv)

echo ""
echo "=== Deployment Complete! ==="
echo ""
echo "Function App URL: https://$URL"
echo ""
echo "Test your functions:"
echo "  curl https://$URL/api/hello?name=World"
echo "  curl https://$URL/api/health"
echo "  curl https://$URL/api/echo?foo=bar"
echo ""
echo "To delete all resources:"
echo "  az group delete --name $RESOURCE_GROUP --yes"
