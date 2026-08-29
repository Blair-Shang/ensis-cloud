# 重启 Go 服务（Windows）
# 用法：powershell -ExecutionPolicy Bypass -File script/restart.ps1

$ErrorActionPreference = "Stop"
$ScriptDir = $PSScriptRoot

& (Join-Path $ScriptDir "stop.ps1")
Start-Sleep -Milliseconds 400
& (Join-Path $ScriptDir "start.ps1")
