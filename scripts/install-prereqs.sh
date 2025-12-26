#!/bin/bash
# install-prereqs.sh - Installs prerequisites for Azure Functions Go Worker development
#
# Usage:
#   ./install-prereqs.sh              # Install Go and Azure Functions Core Tools
#   ./install-prereqs.sh --with-az    # Also install Azure CLI

set -e

INCLUDE_AZ=false
if [[ "$1" == "--with-az" ]]; then
    INCLUDE_AZ=true
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

status() {
    local level=$1
    local msg=$2
    case $level in
        OK)    echo -e "${GREEN}[OK]${NC} $msg" ;;
        WARN)  echo -e "${YELLOW}[WARN]${NC} $msg" ;;
        ERROR) echo -e "${RED}[ERROR]${NC} $msg" ;;
        *)     echo -e "${CYAN}[INFO]${NC} $msg" ;;
    esac
}

command_exists() {
    command -v "$1" &> /dev/null
}

get_go_version() {
    if command_exists go; then
        go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1
    fi
}

get_func_version() {
    if command_exists func; then
        func --version 2>/dev/null | grep -oP '^\d+' | head -1
    fi
}

version_gte() {
    # Returns 0 if $1 >= $2
    printf '%s\n%s\n' "$2" "$1" | sort -V -C
}

# Header
echo ""
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN} Azure Functions Go Worker - Prerequisites${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""

ALL_GOOD=true

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)     PLATFORM="linux" ;;
    Darwin*)    PLATFORM="mac" ;;
    *)          PLATFORM="unknown" ;;
esac

# Check Go
echo -e "Checking Go..."
GO_VERSION=$(get_go_version)
if [[ -z "$GO_VERSION" ]]; then
    status WARN "Go not found"
    
    read -p "Install Go? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ "$PLATFORM" == "mac" ]] && command_exists brew; then
            status INFO "Installing Go via Homebrew..."
            brew install go
            status OK "Go installed"
        elif [[ "$PLATFORM" == "linux" ]]; then
            status INFO "Installing Go..."
            # Download and install Go
            GO_LATEST="1.21.5"
            curl -LO "https://go.dev/dl/go${GO_LATEST}.linux-amd64.tar.gz"
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf "go${GO_LATEST}.linux-amd64.tar.gz"
            rm "go${GO_LATEST}.linux-amd64.tar.gz"
            
            # Add to PATH if not already there
            if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
                echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            fi
            export PATH=$PATH:/usr/local/go/bin
            status OK "Go installed. Run 'source ~/.bashrc' or restart terminal."
        else
            status ERROR "Please install Go manually from https://go.dev/dl/"
            ALL_GOOD=false
        fi
    else
        status WARN "Skipping Go installation"
        ALL_GOOD=false
    fi
elif ! version_gte "$GO_VERSION" "1.21"; then
    status WARN "Go $GO_VERSION found, but 1.21+ required"
    status ERROR "Please update Go from https://go.dev/dl/"
    ALL_GOOD=false
else
    status OK "Go $GO_VERSION found"
fi

# Check Azure Functions Core Tools
echo ""
echo -e "Checking Azure Functions Core Tools..."
FUNC_VERSION=$(get_func_version)
if [[ -z "$FUNC_VERSION" ]]; then
    status WARN "Azure Functions Core Tools not found"
    
    read -p "Install Azure Functions Core Tools? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if command_exists npm; then
            status INFO "Installing Azure Functions Core Tools via npm..."
            npm install -g azure-functions-core-tools@4 --unsafe-perm true
            status OK "Azure Functions Core Tools installed"
        elif [[ "$PLATFORM" == "mac" ]] && command_exists brew; then
            status INFO "Installing Azure Functions Core Tools via Homebrew..."
            brew tap azure/functions
            brew install azure-functions-core-tools@4
            status OK "Azure Functions Core Tools installed"
        else
            status ERROR "Please install npm or see https://learn.microsoft.com/azure/azure-functions/functions-run-local"
            ALL_GOOD=false
        fi
    else
        status WARN "Skipping Azure Functions Core Tools installation"
        ALL_GOOD=false
    fi
elif [[ "$FUNC_VERSION" -lt 4 ]]; then
    status WARN "Azure Functions Core Tools v$FUNC_VERSION found, but v4 required"
    
    read -p "Update Azure Functions Core Tools? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if command_exists npm; then
            npm install -g azure-functions-core-tools@4 --unsafe-perm true
            status OK "Azure Functions Core Tools updated"
        else
            status ERROR "npm not available. Please update manually."
            ALL_GOOD=false
        fi
    else
        status WARN "Skipping update"
        ALL_GOOD=false
    fi
else
    status OK "Azure Functions Core Tools v$FUNC_VERSION found"
fi

# Check Azure CLI (optional)
if [[ "$INCLUDE_AZ" == true ]]; then
    echo ""
    echo -e "Checking Azure CLI..."
    if command_exists az; then
        AZ_VERSION=$(az version --output tsv --query '"azure-cli"' 2>/dev/null)
        status OK "Azure CLI $AZ_VERSION found"
    else
        status WARN "Azure CLI not found"
        
        read -p "Install Azure CLI? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if [[ "$PLATFORM" == "mac" ]] && command_exists brew; then
                brew install azure-cli
                status OK "Azure CLI installed"
            elif [[ "$PLATFORM" == "linux" ]]; then
                curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
                status OK "Azure CLI installed"
            else
                status ERROR "Please install manually from https://learn.microsoft.com/cli/azure/install-azure-cli"
                ALL_GOOD=false
            fi
        else
            status WARN "Skipping Azure CLI installation"
        fi
    fi
fi

# Summary
echo ""
echo -e "${CYAN}========================================${NC}"
if [[ "$ALL_GOOD" == true ]]; then
    status OK "All prerequisites are installed!"
    echo ""
    echo -e "Next steps:"
    echo "  cd samples/hello-world-custom-handler"
    echo "  go build -o handler ."
    echo "  func start"
else
    status WARN "Some prerequisites are missing. Please install them before continuing."
fi
echo ""
