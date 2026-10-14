$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$EnvFile = Join-Path $Root ".env"
if (-not (Test-Path $EnvFile)) {
    Write-Error "Missing .env. Copy .env.example to .env first."
}
docker compose --env-file $EnvFile -f (Join-Path $Root "deploy\docker-compose.yml") logs -f
exit $LASTEXITCODE
