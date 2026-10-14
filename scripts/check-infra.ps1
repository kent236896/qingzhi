$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$EnvFile = Join-Path $Root ".env"
if (-not (Test-Path $EnvFile)) {
    Write-Error "Missing .env. Copy .env.example to .env first."
}
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Error "docker is not available; compose config was not verified."
}
docker compose --env-file $EnvFile --profile edge -f (Join-Path $Root "deploy\docker-compose.yml") config
exit $LASTEXITCODE
