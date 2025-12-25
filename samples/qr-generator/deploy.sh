#!/bin/bash
#
# Deploy the QR Code Generator to Azure Functions
#
# This script creates all necessary Azure resources and deploys the Go-based
# QR Code Generator function app.
#
# Usage:
#   ./deploy.sh -n <function-app-name> [-g <resource-group>] [-l <location>]
#
# Examples:
#   ./deploy.sh -n myqrgen123
#   ./deploy.sh -n myqrgen123 -g my-rg -l westus2
#   ./deploy.sh -n myqrgen123 --skip-resources  # Redeploy only
#
# Prerequisites:
#   - Azure CLI installed and logged in (az login)
#   - Azure Functions Core Tools v4
#   - Go 1.21 or later

set -e

# ============================================================================
# Configuration
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_GROUP="qr-generator-rg"
LOCATION="eastus"
FUNCTION_APP_NAME=""
STORAGE_ACCOUNT_NAME=""
PLAN="consumption"
SKU=""
SKIP_BUILD=false
SKIP_RESOURCES=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ============================================================================
# Helper Functions
# ============================================================================

print_step() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_success() {
    echo -e "${GREEN}  ✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}  ℹ $1${NC}"
}

print_error() {
    echo -e "${RED}  ✗ $1${NC}"
}

usage() {
    cat << EOF
Usage: $(basename "$0") -n <function-app-name> [OPTIONS]

Required:
  -n, --name NAME          Globally unique name for the function app

Options:
  -g, --resource-group RG  Resource group name (default: qr-generator-rg)
  -l, --location LOC       Azure region (default: eastus)
  -s, --storage NAME       Storage account name (auto-generated if not specified)
  -p, --plan PLAN          Hosting plan: consumption (default), premium, dedicated
  --sku SKU                SKU for premium/dedicated (EP1, EP2, EP3, B1, S1, P1v2, etc.)
  --skip-build             Skip Go build step
  --skip-resources         Skip Azure resource creation (redeploy only)
  -h, --help               Show this help message

Plan Options:
  consumption              Pay-per-execution, auto-scale, cold starts (default)
  premium                  No cold starts, VNET support (SKU: EP1, EP2, EP3)
  dedicated                App Service Plan with reserved capacity (SKU: B1, S1, P1v2, etc.)

Examples:
  $(basename "$0") -n myqrgen123
  $(basename "$0") -n myqrgen123 -p premium --sku EP1
  $(basename "$0") -n myqrgen123 -p dedicated --sku S1
  $(basename "$0") -n myqrgen123 --skip-resources
EOF
    exit 1
}

# ============================================================================
# Parse Arguments
# ============================================================================

while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            FUNCTION_APP_NAME="$2"
            shift 2
            ;;
        -g|--resource-group)
            RESOURCE_GROUP="$2"
            shift 2
            ;;
        -l|--location)
            LOCATION="$2"
            shift 2
            ;;
        -s|--storage)
            STORAGE_ACCOUNT_NAME="$2"
            shift 2
            ;;
        -p|--plan)
            PLAN=$(echo "$2" | tr '[:upper:]' '[:lower:]')
            shift 2
            ;;
        --sku)
            SKU="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-resources)
            SKIP_RESOURCES=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Validate required arguments
if [[ -z "$FUNCTION_APP_NAME" ]]; then
    print_error "Function app name is required"
    usage
fi

# Generate storage account name if not provided
if [[ -z "$STORAGE_ACCOUNT_NAME" ]]; then
    # Remove non-alphanumeric chars and convert to lowercase
    STORAGE_ACCOUNT_NAME=$(echo "$FUNCTION_APP_NAME" | tr -dc 'a-zA-Z0-9' | tr '[:upper:]' '[:lower:]')
    # Truncate to 20 chars and add suffix
    STORAGE_ACCOUNT_NAME="${STORAGE_ACCOUNT_NAME:0:20}stor"
fi

# Set default SKU based on plan
if [[ -z "$SKU" ]]; then
    case $PLAN in
        premium) SKU="EP1" ;;
        dedicated) SKU="B1" ;;
        *) SKU="" ;;  # Consumption doesn't need SKU
    esac
fi

# Validate plan
case $PLAN in
    consumption|premium|dedicated) ;;
    *)
        print_error "Invalid plan: $PLAN. Must be: consumption, premium, or dedicated"
        exit 1
        ;;
esac

# Validate SKU for the selected plan
if [[ "$PLAN" == "premium" ]]; then
    case $SKU in
        EP1|EP2|EP3) ;;
        *)
            print_error "Invalid SKU '$SKU' for Premium plan. Valid options: EP1, EP2, EP3"
            exit 1
            ;;
    esac
fi

if [[ "$PLAN" == "dedicated" ]]; then
    case $SKU in
        B1|B2|B3|S1|S2|S3|P1v2|P2v2|P3v2|P1v3|P2v3|P3v3) ;;
        *)
            print_error "Invalid SKU '$SKU' for Dedicated plan. Valid options: B1, B2, B3, S1, S2, S3, P1v2, P2v2, P3v2, P1v3, P2v3, P3v3"
            exit 1
            ;;
    esac
fi

# App Service Plan name (for Premium and Dedicated)
APP_SERVICE_PLAN_NAME="${FUNCTION_APP_NAME}-plan"

# ============================================================================
# Prerequisites Check
# ============================================================================

print_step "Checking Prerequisites"

# Check Azure CLI
if ! command -v az &> /dev/null; then
    print_error "Azure CLI is not installed. Install from: https://docs.microsoft.com/cli/azure/install-azure-cli"
    exit 1
fi
print_success "Azure CLI found"

# Check Azure login
if ! az account show &> /dev/null; then
    print_info "Not logged in to Azure. Running 'az login'..."
    az login
fi
SUBSCRIPTION=$(az account show --query name -o tsv)
print_success "Logged in to Azure subscription: $SUBSCRIPTION"

# Check Azure Functions Core Tools
if ! command -v func &> /dev/null; then
    print_error "Azure Functions Core Tools not installed"
    print_error "Install from: https://docs.microsoft.com/azure/azure-functions/functions-run-local"
    exit 1
fi
FUNC_VERSION=$(func --version)
print_success "Azure Functions Core Tools v$FUNC_VERSION"

# Check Go
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Install from: https://golang.org/dl/"
    exit 1
fi
GO_VERSION=$(go version)
print_success "Go: $GO_VERSION"

# ============================================================================
# Build for Linux
# ============================================================================

if [[ "$SKIP_BUILD" != "true" ]]; then
    print_step "Building Go Worker for Linux (Azure)"

    cd "$SCRIPT_DIR"

    print_info "Cross-compiling for linux/amd64..."
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o worker .

    if [[ ! -f "worker" ]]; then
        print_error "Worker binary was not created"
        exit 1
    fi

    FILE_SIZE=$(du -h worker | cut -f1)
    print_success "Built worker binary ($FILE_SIZE)"
else
    print_step "Skipping Build (--skip-build specified)"
    if [[ ! -f "$SCRIPT_DIR/worker" ]]; then
        print_error "No worker binary found. Run without --skip-build first."
        exit 1
    fi
    print_success "Using existing worker binary"
fi

# ============================================================================
# Create Azure Resources
# ============================================================================

if [[ "$SKIP_RESOURCES" != "true" ]]; then
    print_step "Creating Azure Resources"

    # Create resource group
    print_info "Creating resource group: $RESOURCE_GROUP..."
    az group create \
        --name "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --output none
    print_success "Resource group created: $RESOURCE_GROUP"

    # Create storage account
    print_info "Creating storage account: $STORAGE_ACCOUNT_NAME..."
    az storage account create \
        --name "$STORAGE_ACCOUNT_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --sku Standard_LRS \
        --output none
    print_success "Storage account created: $STORAGE_ACCOUNT_NAME"

    # Create function app based on plan type
    case $PLAN in
        consumption)
            print_info "Creating function app: $FUNCTION_APP_NAME (Consumption plan)..."
            az functionapp create \
                --name "$FUNCTION_APP_NAME" \
                --resource-group "$RESOURCE_GROUP" \
                --storage-account "$STORAGE_ACCOUNT_NAME" \
                --consumption-plan-location "$LOCATION" \
                --runtime custom \
                --functions-version 4 \
                --os-type Linux \
                --output none
            ;;
        premium)
            print_info "Creating App Service Plan: $APP_SERVICE_PLAN_NAME ($SKU)..."
            az functionapp plan create \
                --name "$APP_SERVICE_PLAN_NAME" \
                --resource-group "$RESOURCE_GROUP" \
                --location "$LOCATION" \
                --sku "$SKU" \
                --is-linux \
                --output none
            print_success "App Service Plan created: $APP_SERVICE_PLAN_NAME ($SKU)"

            print_info "Creating function app: $FUNCTION_APP_NAME (Premium plan)..."
            az functionapp create \
                --name "$FUNCTION_APP_NAME" \
                --resource-group "$RESOURCE_GROUP" \
                --storage-account "$STORAGE_ACCOUNT_NAME" \
                --plan "$APP_SERVICE_PLAN_NAME" \
                --runtime custom \
                --functions-version 4 \
                --os-type Linux \
                --output none
            ;;
        dedicated)
            print_info "Creating App Service Plan: $APP_SERVICE_PLAN_NAME ($SKU)..."
            az appservice plan create \
                --name "$APP_SERVICE_PLAN_NAME" \
                --resource-group "$RESOURCE_GROUP" \
                --location "$LOCATION" \
                --sku "$SKU" \
                --is-linux \
                --output none
            print_success "App Service Plan created: $APP_SERVICE_PLAN_NAME ($SKU)"

            print_info "Creating function app: $FUNCTION_APP_NAME (Dedicated plan)..."
            az functionapp create \
                --name "$FUNCTION_APP_NAME" \
                --resource-group "$RESOURCE_GROUP" \
                --storage-account "$STORAGE_ACCOUNT_NAME" \
                --plan "$APP_SERVICE_PLAN_NAME" \
                --runtime custom \
                --functions-version 4 \
                --os-type Linux \
                --output none
            ;;
    esac
    print_success "Function app created: $FUNCTION_APP_NAME"

    # Configure the function app
    print_info "Configuring function app settings..."
    az functionapp config appsettings set \
        --name "$FUNCTION_APP_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --settings "FUNCTIONS_WORKER_RUNTIME=custom" \
        --output none
    print_success "Function app configured"
else
    print_step "Skipping Resource Creation (--skip-resources specified)"
    print_info "Using existing resources"
fi

# ============================================================================
# Deploy
# ============================================================================

print_step "Deploying to Azure"

cd "$SCRIPT_DIR"
print_info "Publishing function app..."

func azure functionapp publish "$FUNCTION_APP_NAME" --no-build

print_success "Deployment completed successfully!"

# ============================================================================
# Summary
# ============================================================================

print_step "Deployment Summary"

FUNCTION_URL="https://$FUNCTION_APP_NAME.azurewebsites.net"

echo ""
echo -e "${GREEN}  Your QR Code Generator is now live!${NC}"
echo ""
echo -e "  🌐 Web UI:        $FUNCTION_URL/api/generate"
echo -e "  📊 Health Check:  $FUNCTION_URL/api/health"
echo -e "  📦 Resource Group: $RESOURCE_GROUP"
echo -e "  📍 Location:       $LOCATION"
PLAN_DISPLAY=$(echo "$PLAN" | sed 's/.*/\u&/')  # Capitalize first letter
if [[ -n "$SKU" && "$PLAN" != "consumption" ]]; then
    echo -e "  💰 Plan:          $PLAN_DISPLAY ($SKU)"
else
    echo -e "  💰 Plan:          $PLAN_DISPLAY"
fi
echo ""
echo -e "${YELLOW}  Test with curl:${NC}"
echo "  curl -X POST $FUNCTION_URL/api/generate \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"content\": \"Hello from Azure!\", \"size\": 256}'"
echo ""
echo -e "${GREEN}  Done! 🎉${NC}"
echo ""
