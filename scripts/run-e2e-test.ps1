# E2E Test Script for Azure Functions Go Worker
# This script starts the gRPC worker and runs E2E tests
#
# Prerequisites:
# - Azure Functions Core Tools 4.x installed
# - Go 1.21+ installed
# - worker.config.json installed in func tools workers/go directory
#
# Usage:
#   .\scripts\run-e2e-test.ps1

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if (-not (Test-Path "$projectRoot\go.mod")) {
    $projectRoot = Split-Path -Parent $PSScriptRoot
}

$sampleDir = Join-Path $projectRoot "samples\hello-world-grpc"
$workerExe = Join-Path $sampleDir "worker.exe"

Write-Host "Project root: $projectRoot" -ForegroundColor Cyan
Write-Host "Sample dir: $sampleDir" -ForegroundColor Cyan

# Build the worker
Write-Host "`nBuilding worker..." -ForegroundColor Yellow
Push-Location $sampleDir
try {
    go build -o worker.exe .
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to build worker"
    }
    Write-Host "Worker built successfully" -ForegroundColor Green
}
finally {
    Pop-Location
}

# Copy worker to func tools directory
$funcToolsDir = Join-Path (Split-Path (Get-Command func).Source) "workers\go"
if (-not (Test-Path $funcToolsDir)) {
    New-Item -ItemType Directory -Path $funcToolsDir -Force | Out-Null
}

Write-Host "`nCopying worker to func tools..." -ForegroundColor Yellow
Copy-Item $workerExe (Join-Path $funcToolsDir "worker.exe") -Force

# Ensure worker.config.json exists
$workerConfig = Join-Path $funcToolsDir "worker.config.json"
if (-not (Test-Path $workerConfig)) {
    Write-Host "Creating worker.config.json..." -ForegroundColor Yellow
    @{
        description = @{
            language = "go"
            extensions = @(".go")
            defaultExecutablePath = "worker.exe"
        }
    } | ConvertTo-Json -Depth 3 | Set-Content $workerConfig -Encoding UTF8
}

# Start func.exe in background
Write-Host "`nStarting Azure Functions Host..." -ForegroundColor Yellow
$funcPath = (Get-Command func -ErrorAction SilentlyContinue).Source
if (-not $funcPath) {
    throw "func.exe not found in PATH. Please install Azure Functions Core Tools."
}
Write-Host "Using func at: $funcPath" -ForegroundColor Gray

# Use cmd.exe to start func in background
$job = Start-Job -ScriptBlock {
    param($dir)
    Set-Location $dir
    & func start 2>&1
} -ArgumentList $sampleDir

# Wait for func to be ready
Write-Host "Waiting for Functions Host to be ready..." -ForegroundColor Yellow
$maxWait = 60
$waited = 0
$ready = $false

# Wait a few seconds for initial startup
Start-Sleep -Seconds 5

while ($waited -lt $maxWait -and -not $ready) {
    Start-Sleep -Seconds 1
    $waited++
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:7071/api/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            $ready = $true
        }
    }
    catch {
        # Check if the job is still running
        if ($job.State -ne "Running") {
            Write-Host "  Job stopped unexpectedly. State: $($job.State)" -ForegroundColor Red
            break
        }
        Write-Host "  Waiting... ($waited/$maxWait)" -ForegroundColor Gray
    }
}

if (-not $ready) {
    Write-Host "Functions Host did not start in time" -ForegroundColor Red
    Write-Host "Job output:" -ForegroundColor Yellow
    Receive-Job -Job $job
    Stop-Job -Job $job -ErrorAction SilentlyContinue
    Remove-Job -Job $job -Force -ErrorAction SilentlyContinue
    exit 1
}

Write-Host "Functions Host is ready!" -ForegroundColor Green

# Run tests
$testsPassed = 0
$testsFailed = 0

function Test-Endpoint {
    param (
        [string]$Name,
        [string]$Url,
        [string]$ExpectedContent,
        [string]$Method = "GET",
        [string]$Body = $null,
        [string]$ContentType = "application/json"
    )
    
    Write-Host "`nTest: $Name" -ForegroundColor Cyan
    try {
        $params = @{
            Uri = $Url
            Method = $Method
            UseBasicParsing = $true
        }
        
        if ($Body) {
            $params.Body = $Body
            $params.ContentType = $ContentType
        }
        
        $response = Invoke-WebRequest @params
        
        if ($response.Content -like "*$ExpectedContent*") {
            Write-Host "  PASS: Response contains expected content" -ForegroundColor Green
            Write-Host "  Response: $($response.Content)" -ForegroundColor Gray
            return $true
        } else {
            Write-Host "  FAIL: Expected content not found" -ForegroundColor Red
            Write-Host "  Expected: $ExpectedContent" -ForegroundColor Red
            Write-Host "  Got: $($response.Content)" -ForegroundColor Red
            return $false
        }
    }
    catch {
        Write-Host "  FAIL: Request failed - $_" -ForegroundColor Red
        return $false
    }
}

# Test 1: Health check
if (Test-Endpoint -Name "Health check" -Url "http://localhost:7071/api/health" -ExpectedContent "healthy") {
    $testsPassed++
} else {
    $testsFailed++
}

# Test 2: Hello with name in query string
if (Test-Endpoint -Name "Hello with name" -Url "http://localhost:7071/api/hello?name=E2ETest" -ExpectedContent "Hello, E2ETest!") {
    $testsPassed++
} else {
    $testsFailed++
}

# Test 3: Hello without name (default)
if (Test-Endpoint -Name "Hello default" -Url "http://localhost:7071/api/hello" -ExpectedContent "Hello, World!") {
    $testsPassed++
} else {
    $testsFailed++
}

# Test 4: Echo POST
$echoBody = '{"message": "test"}'
if (Test-Endpoint -Name "Echo POST" -Url "http://localhost:7071/api/echo" -ExpectedContent "test" -Method "POST" -Body $echoBody) {
    $testsPassed++
} else {
    $testsFailed++
}

# Cleanup
Write-Host "`nStopping Functions Host..." -ForegroundColor Yellow
Stop-Job -Job $job -ErrorAction SilentlyContinue
Remove-Job -Job $job -Force -ErrorAction SilentlyContinue
# Also kill any remaining func processes
Get-Process -Name "func" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

# Summary
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "E2E Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Passed: $testsPassed" -ForegroundColor Green
Write-Host "Failed: $testsFailed" -ForegroundColor $(if ($testsFailed -eq 0) { "Green" } else { "Red" })

if ($testsFailed -eq 0) {
    Write-Host "`nAll E2E tests passed!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "`nSome E2E tests failed" -ForegroundColor Red
    exit 1
}
