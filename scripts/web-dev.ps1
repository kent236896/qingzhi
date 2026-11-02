$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot

# Windows 上 `pnpm` 是无无扩展名的 shim / .ps1 / .cmd，不是 .exe。
# ProcessStartInfo + UseShellExecute=$false 只能启动真正的可执行文件，
# 直接 FileName="pnpm" 会报「%1 不是有效的 Win32 应用程序」或「找不到指定的文件」。
$pnpmCmd = $null
foreach ($name in @("pnpm.cmd", "pnpm")) {
    $found = Get-Command $name -ErrorAction SilentlyContinue
    if ($found -and $found.Source -and (Test-Path -LiteralPath $found.Source)) {
        $pnpmCmd = $found.Source
        break
    }
}
if (-not $pnpmCmd) {
    Write-Host "启动失败: 未找到 pnpm。请先安装: npm i -g pnpm@11.20.0" -ForegroundColor Red
    exit 1
}

$apps = @(
    "apps\admin-web",
    "apps\user-workbench"
)

$processes = @()

try {
    foreach ($app in $apps) {
        $appPath = Join-Path $Root $app
        if (-not (Test-Path -LiteralPath $appPath)) {
            throw "目录不存在: $appPath"
        }

        $psi = New-Object System.Diagnostics.ProcessStartInfo
        $psi.FileName = "cmd.exe"
        $psi.Arguments = "/d /c `"$pnpmCmd`" dev"
        $psi.WorkingDirectory = $appPath
        $psi.UseShellExecute = $false

        $proc = [System.Diagnostics.Process]::Start($psi)
        if ($null -eq $proc) {
            throw "无法启动进程: $app"
        }
        $processes += $proc

        Write-Host "已启动: $app (pid=$($proc.Id))"
    }

    Write-Host ""
    Write-Host "两个前端服务已启动（admin-web / user-workbench）"
    Write-Host "日志会混在同一终端；按 Ctrl+C 停止所有服务"
    Write-Host ""

    while ($true) {
        $allStopped = $true
        foreach ($proc in $processes) {
            if (-not $proc.HasExited) {
                $allStopped = $false
                break
            }
        }
        if ($allStopped) { break }
        Start-Sleep -Milliseconds 500
    }

} catch {
    Write-Host "启动失败: $_" -ForegroundColor Red
    exit 1
} finally {
    foreach ($proc in $processes) {
        try {
            if (-not $proc.HasExited) {
                # 先杀 cmd，再尽量带走子进程树
                Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
            }
        } catch {}
    }
}
