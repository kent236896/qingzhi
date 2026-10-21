$ErrorActionPreference = "Stop"

if (-not $env:GOTOOLCHAIN) {
    $env:GOTOOLCHAIN = "auto"
}

if (Get-Command go -ErrorAction SilentlyContinue) {
    return
}

$candidates = @(
    "C:\Go\bin",
    (Join-Path $env:ProgramFiles "Go\bin"),
    (Join-Path ${env:ProgramFiles(x86)} "Go\bin"),
    (Join-Path $env:LocalAppData "Programs\Go\bin")
)

foreach ($dir in $candidates) {
    if ($dir -and (Test-Path (Join-Path $dir "go.exe"))) {
        $env:Path = "$dir;$env:Path"
        return
    }
}

throw "go.exe was not found. Install Go or add its bin directory to PATH."
