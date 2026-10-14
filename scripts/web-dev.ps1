$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $Root "apps\admin-web")
try {
    pnpm dev
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
