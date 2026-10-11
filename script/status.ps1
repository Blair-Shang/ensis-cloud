# 查看 Go 服务状态（Windows）
# 用法：powershell -ExecutionPolicy Bypass -File script/status.ps1

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$PidFile = Join-Path $Root "run\ensis-cloud.pid"
$Health = "http://127.0.0.1:8090/niuma/cloud/healthz"

if (-not (Test-Path $PidFile)) {
  Write-Host "stopped (no pid file)"
  exit 1
}

$procId = Get-Content $PidFile -ErrorAction SilentlyContinue
if (-not $procId) {
  Write-Host "stopped (empty pid file)"
  exit 1
}

$p = Get-Process -Id $procId -ErrorAction SilentlyContinue
if (-not $p) {
  Write-Host "stopped (stale pid=$procId)"
  exit 1
}

Write-Host "running pid=$procId"
try {
  $resp = Invoke-WebRequest -Uri $Health -UseBasicParsing -TimeoutSec 3
  Write-Host "healthz $($resp.StatusCode) $Health"
} catch {
  Write-Host "healthz unreachable $Health"
  exit 1
}
