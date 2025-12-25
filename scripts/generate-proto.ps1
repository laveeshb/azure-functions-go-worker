# Generate Go code from protobuf files
# Run from repository root: .\scripts\generate-proto.ps1

$ErrorActionPreference = "Stop"

$ROOT = Split-Path -Parent $PSScriptRoot
if (-not $ROOT) {
    $ROOT = Get-Location
}

$PROTO_DIR = Join-Path $ROOT "proto"
$OUT_DIR = Join-Path $ROOT "internal\rpc\proto"

# Create output directory if it doesn't exist
if (-not (Test-Path $OUT_DIR)) {
    New-Item -ItemType Directory -Path $OUT_DIR -Force | Out-Null
}

Write-Host "Generating Go code from protobuf files..."
Write-Host "Proto dir: $PROTO_DIR"
Write-Host "Output dir: $OUT_DIR"

# Find protobuf include directory (installed via winget)
$PROTOBUF_INCLUDE = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages\Google.Protobuf_Microsoft.Winget.Source_8wekyb3d8bbwe\include"
if (-not (Test-Path $PROTOBUF_INCLUDE)) {
    Write-Host "Warning: Could not find protobuf includes at $PROTOBUF_INCLUDE" -ForegroundColor Yellow
    Write-Host "Trying to find protobuf includes..." -ForegroundColor Yellow
    $PROTOBUF_INCLUDE = Get-ChildItem -Path "$env:LOCALAPPDATA\Microsoft\WinGet\Packages" -Filter "Google.Protobuf*" -Directory | 
        Select-Object -First 1 -ExpandProperty FullName | 
        ForEach-Object { Join-Path $_ "include" }
}

Write-Host "Protobuf includes: $PROTOBUF_INCLUDE"

# Generate Go code
# Note: We use module prefix to ensure all files go to the same package
protoc `
    --proto_path="$PROTO_DIR" `
    --proto_path="$PROTOBUF_INCLUDE" `
    --go_out="$ROOT" `
    --go_opt=module=github.com/Azure/azure-functions-go-worker `
    --go-grpc_out="$ROOT" `
    --go-grpc_opt=module=github.com/Azure/azure-functions-go-worker `
    "$PROTO_DIR\shared\NullableTypes.proto" `
    "$PROTO_DIR\identity\ClaimsIdentityRpc.proto" `
    "$PROTO_DIR\FunctionRpc.proto"

if ($LASTEXITCODE -eq 0) {
    Write-Host "Proto generation completed successfully!" -ForegroundColor Green
} else {
    Write-Host "Proto generation failed!" -ForegroundColor Red
    exit 1
}
