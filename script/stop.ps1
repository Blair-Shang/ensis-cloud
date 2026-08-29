# 停止 Go 服务（Windows）
# 用法：powershell -ExecutionPolicy Bypass -File script/stop.ps1

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$PidFile = Join-Path $Root "run\niuma-cloud.pid"

if (-not (Test-Path $PidFile)) {
  Write-Host "Not running (no pid file)."
  exit 0
}

$procId = Get-Content $PidFile -ErrorAction SilentlyContinue
if (-not $procId) {
  Remove-Item $PidFile -Force -ErrorAction SilentlyContinue
  Write-Host "Not running."
  exit 0
}

$p = Get-Process -Id $procId -ErrorAction SilentlyContinue
if ($p) {
  Stop-Process -Id $procId
  for ($i = 0; $i -lt 8; $i++) {
    if (-not (Get-Process -Id $procId -ErrorAction SilentlyContinue)) {
      break
    }
    Start-Sleep -Milliseconds 400
  }
  if (Get-Process -Id $procId -ErrorAction SilentlyContinue) {
    Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
  }
  Write-Host "Stopped pid=$procId"
} else {
  Write-Host "Process $procId not found."
}

Remove-Item $PidFile -Force -ErrorAction SilentlyContinue
