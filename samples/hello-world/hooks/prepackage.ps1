# Pre-package hook for azd - builds the Go binary for Linux
Write-Host "Building Go binary for Linux..."

Push-Location src

# Build for Linux (Azure Functions runs on Linux)
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o handler .

if ($LASTEXITCODE -ne 0) {
    Pop-Location
    throw "Failed to build Go binary"
}

Write-Host "Go binary built successfully"
Get-ChildItem handler

Pop-Location
