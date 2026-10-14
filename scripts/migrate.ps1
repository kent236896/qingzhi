param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidateSet("up", "down", "version")]
    [string]$Command
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $Root "services\enterprise-api")
try {
    go run ./cmd/migrate $Command
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
