<#
.SYNOPSIS
    Installs prerequisites for Azure Functions Go Worker development.

.DESCRIPTION
    Checks for and installs:
    - Go 1.21 or later
    - Azure Functions Core Tools v4
    - Azure CLI (optional, for deployment)

.EXAMPLE
    .\install-prereqs.ps1
    
.EXAMPLE
    .\install-prereqs.ps1 -IncludeAzureCLI
#>

param(
    [switch]$IncludeAzureCLI
)

$ErrorActionPreference = "Stop"

function Write-Status {
    param([string]$Message, [string]$Status = "INFO")
    $color = switch ($Status) {
        "OK"    { "Green" }
        "WARN"  { "Yellow" }
        "ERROR" { "Red" }
        default { "Cyan" }
    }
    Write-Host "[$Status] " -ForegroundColor $color -NoNewline
    Write-Host $Message
}

function Test-Command {
    param([string]$Command)
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

function Get-GoVersion {
    try {
        $output = & go version 2>&1
        if ($output -match "go(\d+\.\d+)") {
            return [version]$Matches[1]
        }
    } catch {}
    return $null
}

function Get-FuncVersion {
    try {
        $output = & func --version 2>&1
        if ($output -match "^(\d+)\.") {
            return [int]$Matches[1]
        }
    } catch {}
    return $null
}

function Install-WithWinget {
    param([string]$PackageId, [string]$Name)
    
    if (-not (Test-Command "winget")) {
        Write-Status "winget not available. Please install $Name manually." "ERROR"
        return $false
    }
    
    Write-Status "Installing $Name via winget..."
    winget install --id $PackageId --accept-source-agreements --accept-package-agreements
    return $LASTEXITCODE -eq 0
}

function Install-WithNpm {
    param([string]$Package, [string]$Name)
    
    if (-not (Test-Command "npm")) {
        Write-Status "npm not available. Please install Node.js first, or install $Name manually." "ERROR"
        return $false
    }
    
    Write-Status "Installing $Name via npm..."
    npm install -g $Package
    return $LASTEXITCODE -eq 0
}

# Header
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Azure Functions Go Worker - Prerequisites" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$allGood = $true

# Check Go
Write-Host "Checking Go..." -ForegroundColor White
$goVersion = Get-GoVersion
if ($null -eq $goVersion) {
    Write-Status "Go not found" "WARN"
    
    $install = Read-Host "Install Go? (y/N)"
    if ($install -eq "y" -or $install -eq "Y") {
        if (Install-WithWinget "GoLang.Go" "Go") {
            Write-Status "Go installed. Please restart your terminal." "OK"
        } else {
            Write-Status "Failed to install Go. Download from https://go.dev/dl/" "ERROR"
            $allGood = $false
        }
    } else {
        Write-Status "Skipping Go installation" "WARN"
        $allGood = $false
    }
} elseif ($goVersion -lt [version]"1.21") {
    Write-Status "Go $goVersion found, but 1.21+ required" "WARN"
    
    $install = Read-Host "Update Go? (y/N)"
    if ($install -eq "y" -or $install -eq "Y") {
        if (Install-WithWinget "GoLang.Go" "Go") {
            Write-Status "Go updated. Please restart your terminal." "OK"
        } else {
            Write-Status "Failed to update Go. Download from https://go.dev/dl/" "ERROR"
            $allGood = $false
        }
    } else {
        Write-Status "Skipping Go update" "WARN"
        $allGood = $false
    }
} else {
    Write-Status "Go $goVersion found" "OK"
}

# Check Azure Functions Core Tools
Write-Host ""
Write-Host "Checking Azure Functions Core Tools..." -ForegroundColor White
$funcVersion = Get-FuncVersion
if ($null -eq $funcVersion) {
    Write-Status "Azure Functions Core Tools not found" "WARN"
    
    $install = Read-Host "Install Azure Functions Core Tools? (y/N)"
    if ($install -eq "y" -or $install -eq "Y") {
        # Try winget first, then npm
        if (Test-Command "winget") {
            if (Install-WithWinget "Microsoft.Azure.FunctionsCoreTools" "Azure Functions Core Tools") {
                Write-Status "Azure Functions Core Tools installed" "OK"
            } else {
                Write-Status "winget install failed, trying npm..." "WARN"
                if (Install-WithNpm "azure-functions-core-tools@4" "Azure Functions Core Tools") {
                    Write-Status "Azure Functions Core Tools installed via npm" "OK"
                } else {
                    Write-Status "Failed to install. See https://learn.microsoft.com/azure/azure-functions/functions-run-local" "ERROR"
                    $allGood = $false
                }
            }
        } elseif (Test-Command "npm") {
            if (Install-WithNpm "azure-functions-core-tools@4" "Azure Functions Core Tools") {
                Write-Status "Azure Functions Core Tools installed via npm" "OK"
            } else {
                Write-Status "Failed to install. See https://learn.microsoft.com/azure/azure-functions/functions-run-local" "ERROR"
                $allGood = $false
            }
        } else {
            Write-Status "Neither winget nor npm available. Please install manually." "ERROR"
            $allGood = $false
        }
    } else {
        Write-Status "Skipping Azure Functions Core Tools installation" "WARN"
        $allGood = $false
    }
} elseif ($funcVersion -lt 4) {
    Write-Status "Azure Functions Core Tools v$funcVersion found, but v4 required" "WARN"
    
    $install = Read-Host "Update Azure Functions Core Tools? (y/N)"
    if ($install -eq "y" -or $install -eq "Y") {
        if (Test-Command "npm") {
            if (Install-WithNpm "azure-functions-core-tools@4" "Azure Functions Core Tools") {
                Write-Status "Azure Functions Core Tools updated" "OK"
            } else {
                $allGood = $false
            }
        } else {
            Write-Status "npm not available for update. Please update manually." "ERROR"
            $allGood = $false
        }
    } else {
        Write-Status "Skipping update" "WARN"
        $allGood = $false
    }
} else {
    Write-Status "Azure Functions Core Tools v$funcVersion found" "OK"
}

# Check Azure CLI (optional)
if ($IncludeAzureCLI) {
    Write-Host ""
    Write-Host "Checking Azure CLI..." -ForegroundColor White
    if (Test-Command "az") {
        $azVersion = (az version --output json | ConvertFrom-Json).'azure-cli'
        Write-Status "Azure CLI $azVersion found" "OK"
    } else {
        Write-Status "Azure CLI not found" "WARN"
        
        $install = Read-Host "Install Azure CLI? (y/N)"
        if ($install -eq "y" -or $install -eq "Y") {
            if (Install-WithWinget "Microsoft.AzureCLI" "Azure CLI") {
                Write-Status "Azure CLI installed. Please restart your terminal." "OK"
            } else {
                Write-Status "Failed to install. See https://learn.microsoft.com/cli/azure/install-azure-cli" "ERROR"
                $allGood = $false
            }
        } else {
            Write-Status "Skipping Azure CLI installation" "WARN"
        }
    }
}

# Summary
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
if ($allGood) {
    Write-Status "All prerequisites are installed!" "OK"
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor White
    Write-Host "  cd samples/hello-world-custom-handler"
    Write-Host "  go build -o handler.exe ."
    Write-Host "  func start"
} else {
    Write-Status "Some prerequisites are missing. Please install them before continuing." "WARN"
}
Write-Host ""
