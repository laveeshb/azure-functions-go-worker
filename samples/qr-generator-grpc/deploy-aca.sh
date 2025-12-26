#!/bin/bash
# Azure Functions Go Worker - QR Generator (gRPC) - Deploy to Azure Container Apps
#
# This script deploys the Go gRPC worker as a container to Azure Container Apps.
# This approach allows the full gRPC protocol since we bundle the Functions Host.
#
# For simpler Azure Functions deployment, use ../qr-generator-custom-handler/deploy.sh

set -e

# Default values
RESOURCE_GROUP=""
LOCATION=""
APP_NAME=""
REGISTRY_NAME=""

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
            APP_NAME="$2"
            shift 2
            ;;
        -r|--registry)
            REGISTRY_NAME="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 -g <resource-group> -l <location> [-n <app-name>] [-r <registry-name>]"
            echo ""
            echo "Options:"
            echo "  -g, --resource-group  Resource group name (required)"
            echo "  -l, --location        Azure region (required)"
            echo "  -n, --name            Container app name (auto-generated if not provided)"
            echo "  -r, --registry        Container registry name (auto-generated if not provided)"
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

# Generate unique names if not provided
if [ -z "$APP_NAME" ]; then
    SUFFIX=$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 6 | head -n 1)
    APP_NAME="qr-grpc-$SUFFIX"
fi

if [ -z "$REGISTRY_NAME" ]; then
    REGISTRY_NAME="acr$(cat /dev/urandom | tr -dc 'a-z' | fold -w 10 | head -n 1)"
fi

ENVIRONMENT_NAME="$APP_NAME-env"
IMAGE_NAME="$REGISTRY_NAME.azurecr.io/qr-generator-grpc:latest"

echo "=== QR Generator (gRPC) - Azure Container Apps Deployment ==="
echo "Resource Group: $RESOURCE_GROUP"
echo "Location: $LOCATION"
echo "Container App: $APP_NAME"
echo "Container Registry: $REGISTRY_NAME"
echo ""

# Check prerequisites
echo "Step 1: Checking prerequisites..."
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is required but not installed."
    exit 1
fi
docker --version
echo "  Done!"

# Create Resource Group
echo "Step 2: Creating Resource Group..."
az group create --name "$RESOURCE_GROUP" --location "$LOCATION" --output none
echo "  Done!"

# Create Azure Container Registry
echo "Step 3: Creating Azure Container Registry..."
az acr create \
    --name "$REGISTRY_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --location "$LOCATION" \
    --sku Basic \
    --admin-enabled true \
    --output none
echo "  Done!"

# Build the Docker image (from repository root)
echo "Step 4: Building Docker image..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../.."
docker build -t qr-generator-grpc -f samples/qr-generator-grpc/Dockerfile .
cd "$SCRIPT_DIR"
echo "  Done!"

# Login to ACR
echo "Step 5: Logging into Container Registry..."
az acr login --name "$REGISTRY_NAME"
echo "  Done!"

# Tag and push the image
echo "Step 6: Pushing image to registry..."
docker tag qr-generator-grpc "$IMAGE_NAME"
docker push "$IMAGE_NAME"
echo "  Done!"

# Get ACR credentials
echo "Step 7: Getting registry credentials..."
ACR_PASSWORD=$(az acr credential show --name "$REGISTRY_NAME" --query "passwords[0].value" -o tsv)
echo "  Done!"

# Create Container Apps Environment
echo "Step 8: Creating Container Apps Environment..."
az containerapp env create \
    --name "$ENVIRONMENT_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --location "$LOCATION" \
    --output none
echo "  Done!"

# Create Container App
echo "Step 9: Creating Container App..."
az containerapp create \
    --name "$APP_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --environment "$ENVIRONMENT_NAME" \
    --image "$IMAGE_NAME" \
    --target-port 80 \
    --ingress external \
    --registry-server "$REGISTRY_NAME.azurecr.io" \
    --registry-username "$REGISTRY_NAME" \
    --registry-password "$ACR_PASSWORD" \
    --min-replicas 0 \
    --max-replicas 10 \
    --cpu 0.5 \
    --memory 1.0Gi \
    --output none
echo "  Done!"

# Get the URL
URL=$(az containerapp show \
    --name "$APP_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --query "properties.configuration.ingress.fqdn" -o tsv)

echo ""
echo "=== Deployment Complete! ==="
echo ""
echo "Container App URL: https://$URL"
echo ""
echo "Try it out:"
echo "  Interactive UI: https://$URL/api/generate"
echo "  Health check:   curl https://$URL/api/health"
echo ""
echo "API usage:"
echo "  curl -X POST https://$URL/api/generate \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"content\": \"https://example.com\", \"size\": 256}'"
echo ""
echo "View logs:"
echo "  az containerapp logs show --name $APP_NAME --resource-group $RESOURCE_GROUP --follow"
echo ""
echo "To delete all resources:"
echo "  az group delete --name $RESOURCE_GROUP --yes"
