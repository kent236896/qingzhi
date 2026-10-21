$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "use-go.ps1")
$Root = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $Root "services\enterprise-api")
try {
    go run ./cmd/server
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
