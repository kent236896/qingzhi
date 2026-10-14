$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot

Push-Location (Join-Path $Root "services\enterprise-api")
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Pop-Location
}

Push-Location (Join-Path $Root "apps\admin-web")
try {
    pnpm typecheck
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    pnpm lint
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    pnpm test
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    pnpm build
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Pop-Location
}
