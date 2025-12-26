#!/bin/bash
# Azure Functions Go Worker - QR Generator Deploy Script
# Deploys a Go Function App to Azure using Custom Handler

set -e

# Default values
RESOURCE_GROUP=""
LOCATION=""
FUNCTION_APP_NAME=""
PLAN="Consumption"
SKU=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -g|--resource-group)
            RESOURCE_GROUP="$2"
            shift 2
            ;;
        -l|--location)
            LOCATION="$2"
            shift 2
            ;;
        -n|--name)
            FUNCTION_APP_NAME="$2"
            shift 2
            ;;
        -p|--plan)
            PLAN="$2"
            shift 2
            ;;
        -s|--sku)
            SKU="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 -g <resource-group> -l <location> [-n <app-name>] [-p <plan>] [-s <sku>]"
            echo ""
            echo "Options:"
            echo "  -g, --resource-group  Resource group name (required)"
            echo "  -l, --location        Azure region (required)"
            echo "  -n, --name            Function app name (auto-generated if not provided)"
            echo "  -p, --plan            Hosting plan: Consumption, Premium, Dedicated (default: Consumption)"
            echo "  -s, --sku             SKU: Y1, EP1-EP3, B1-P3v3 (default based on plan)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Validate required parameters
if [ -z "$RESOURCE_GROUP" ] || [ -z "$LOCATION" ]; then
    echo "Error: Resource group and location are required"
    echo "Usage: $0 -g <resource-group> -l <location>"
    exit 1
fi

# Validate and set SKU based on plan
case $PLAN in
    Consumption)
        if [ -n "$SKU" ] && [ "$SKU" != "Y1" ]; then
            echo "Warning: Consumption plan only supports Y1 SKU, ignoring provided SKU"
        fi
        SKU="Y1"
        ;;
    Premium)
        if [ -z "$SKU" ]; then SKU="EP1"; fi
        if ! [[ "$SKU" =~ ^EP[123]$ ]]; then
            echo "Error: Premium plan requires SKU: EP1, EP2, or EP3. Got: $SKU"
            exit 1
        fi
        ;;
    Dedicated)
        if [ -z "$SKU" ]; then SKU="B1"; fi
        ;;
    *)
        echo "Error: Invalid plan. Must be: Consumption, Premium, or Dedicated"
        exit 1
        ;;
esac

# Generate unique name if not provided
if [ -z "$FUNCTION_APP_NAME" ]; then
    SUFFIX=$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 8 | head -n 1)
    FUNCTION_APP_NAME="func-qr-$SUFFIX"
fi

STORAGE_ACCOUNT_NAME="stqr$(cat /dev/urandom | tr -dc 'a-z' | fold -w 10 | head -n 1)"

echo "=== QR Generator - Custom Handler Deployment ==="
echo "Resource Group: $RESOURCE_GROUP"
echo "Location: $LOCATION"
echo "Function App: $FUNCTION_APP_NAME"
echo "Storage Account: $STORAGE_ACCOUNT_NAME"
echo "Plan: $PLAN ($SKU)"
echo ""

# Build the Go binary for Linux
echo "Step 1: Building Go binary for Linux..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o handler .
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

# Create Function App based on plan type
echo "Step 4: Creating Function App ($PLAN plan)..."

if [ "$PLAN" == "Consumption" ]; then
    az functionapp create \
        --name "$FUNCTION_APP_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --storage-account "$STORAGE_ACCOUNT_NAME" \
        --consumption-plan-location "$LOCATION" \
        --runtime custom \
        --functions-version 4 \
        --os-type Linux \
        --output none
elif [ "$PLAN" == "Premium" ]; then
    PLAN_NAME="$FUNCTION_APP_NAME-plan"
    az functionapp plan create \
        --name "$PLAN_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --sku "$SKU" \
        --is-linux \
        --output none
    
    az functionapp create \
        --name "$FUNCTION_APP_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --storage-account "$STORAGE_ACCOUNT_NAME" \
        --plan "$PLAN_NAME" \
        --runtime custom \
        --functions-version 4 \
        --os-type Linux \
        --output none
else
    # Dedicated
    PLAN_NAME="$FUNCTION_APP_NAME-plan"
    az appservice plan create \
        --name "$PLAN_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --sku "$SKU" \
        --is-linux \
        --output none
    
    az functionapp create \
        --name "$FUNCTION_APP_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --storage-account "$STORAGE_ACCOUNT_NAME" \
        --plan "$PLAN_NAME" \
        --runtime custom \
        --functions-version 4 \
        --os-type Linux \
        --output none
fi
echo "  Done!"

# Deploy using func CLI
echo "Step 5: Deploying to Azure..."
func azure functionapp publish "$FUNCTION_APP_NAME" --no-build --custom
echo "  Done!"

# Get the URL
URL=$(az functionapp show --name "$FUNCTION_APP_NAME" --resource-group "$RESOURCE_GROUP" --query "defaultHostName" -o tsv)

echo ""
echo "=== Deployment Complete! ==="
echo ""
echo "Function App URL: https://$URL"
echo ""
echo "Try it out:"
echo "  Landing Page:   https://$URL/"
echo "  Health check:   curl https://$URL/health"
echo ""
echo "API usage:"
echo "  curl -X POST https://$URL/generate \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"content\": \"https://example.com\", \"size\": 256}'"
echo ""
echo "To delete all resources:"
echo "  az group delete --name $RESOURCE_GROUP --yes"
