# 构建 Cloud：Admin 前端 → 嵌入目录 → Go 单二进制（Windows）
# 用法：powershell -ExecutionPolicy Bypass -File script/build.ps1

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$RunDir = Join-Path $Root "run"
New-Item -ItemType Directory -Force -Path $RunDir | Out-Null

Write-Host "==> pnpm install"
pnpm install

Write-Host "==> admin build (embed -> server/internal/web/dist)"
pnpm exec vue-tsc --noEmit -p admin/tsconfig.json
pnpm exec vite build --config admin/vite.config.ts

$Exe = Join-Path $RunDir "ensis-cloud.exe"
Write-Host "==> go build -> $Exe"
Push-Location (Join-Path $Root "server")
try {
  go build -o $Exe ./cmd/server
} finally {
  Pop-Location
}

Write-Host "OK: $Exe"
Write-Host "Start: powershell -ExecutionPolicy Bypass -File script/start.ps1"
