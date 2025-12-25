# E2E Test Script for Azure Functions Go Worker
# This script starts the Custom Handler and runs E2E tests
#
# Prerequisites:
# - Azure Functions Core Tools 4.x installed
# - Go 1.21+ installed
#
# Usage:
#   .\scripts\run-e2e-test.ps1

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if (-not (Test-Path "$projectRoot\go.mod")) {
    $projectRoot = Split-Path -Parent $PSScriptRoot
}

$exampleDir = Join-Path $projectRoot "examples\httpHandler"
$handlerExe = Join-Path $exampleDir "handler.exe"

Write-Host "Project root: $projectRoot" -ForegroundColor Cyan
Write-Host "Example dir: $exampleDir" -ForegroundColor Cyan

# Build the handler
Write-Host "`nBuilding handler..." -ForegroundColor Yellow
Push-Location $exampleDir
try {
    go build -o handler.exe .
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to build handler"
    }
    Write-Host "Handler built successfully" -ForegroundColor Green
}
finally {
    Pop-Location
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
} -ArgumentList $exampleDir

# Wait for func to be ready - give it more time since it needs to download extensions
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
        $response = Invoke-WebRequest -Uri "http://localhost:7071/api/HttpTrigger" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
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
        [string]$Body = $null
    )
    
    Write-Host "`nTest: $Name" -ForegroundColor Cyan
    try {
        if ($Method -eq "POST" -and $Body) {
            $response = Invoke-WebRequest -Uri $Url -Method POST -Body $Body -ContentType "text/plain" -UseBasicParsing
        } else {
            $response = Invoke-WebRequest -Uri $Url -UseBasicParsing
        }
        
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

# Test 1: HttpTrigger with name
if (Test-Endpoint -Name "HttpTrigger with name" -Url "http://localhost:7071/api/HttpTrigger?name=E2ETest" -ExpectedContent "Hello, E2ETest!") {
    $testsPassed++
} else {
    $testsFailed++
}

# Test 2: HttpTrigger without name (default)
if (Test-Endpoint -Name "HttpTrigger default" -Url "http://localhost:7071/api/HttpTrigger" -ExpectedContent "Hello, World!") {
    $testsPassed++
} else {
    $testsFailed++
}

# Test 3: HelloWorld JSON response
if (Test-Endpoint -Name "HelloWorld JSON" -Url "http://localhost:7071/api/HelloWorld" -ExpectedContent "Hello from Azure Functions Go Worker") {
    $testsPassed++
} else {
    $testsFailed++
}

# Test 4: POST with body
if (Test-Endpoint -Name "HttpTrigger POST" -Url "http://localhost:7071/api/HttpTrigger" -ExpectedContent "Hello, PostBody!" -Method "POST" -Body "PostBody") {
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
