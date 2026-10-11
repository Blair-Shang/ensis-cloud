# 启动内嵌 Admin 的 Go 服务（Windows）
# 用法：powershell -ExecutionPolicy Bypass -File script/start.ps1

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$RunDir = Join-Path $Root "run"
$ConfigDir = Join-Path $Root "config"
$LogDir = Join-Path $Root "logs"
$DataDir = Join-Path $Root "data"
$Exe = Join-Path $RunDir "ensis-cloud.exe"
$PidFile = Join-Path $RunDir "ensis-cloud.pid"
$BootstrapErr = Join-Path $LogDir "bootstrap.err"

if (-not (Test-Path $Exe)) {
  Write-Error "Binary not found: $Exe`nRun script/build.ps1 first."
}

if (Test-Path $PidFile) {
  $old = Get-Content $PidFile -ErrorAction SilentlyContinue
  if ($old -and (Get-Process -Id $old -ErrorAction SilentlyContinue)) {
    Write-Host "Already running (pid=$old). Use script/restart.ps1 or stop first."
    exit 0
  }
}

New-Item -ItemType Directory -Force -Path $RunDir, $ConfigDir, $LogDir, $DataDir | Out-Null

$example = Join-Path $ConfigDir "app.yaml.example"
$appYaml = Join-Path $ConfigDir "app.yaml"
if ((Test-Path $example) -and -not (Test-Path $appYaml)) {
  Copy-Item $example $appYaml
  Write-Host "Created $appYaml from example"
  Write-Host "Edit jwt.secret / db.dsn / staff.password before production use."
}

# 应用日志由 zap 写入 logs/ensis-cloud.log；此处仅捕获启动极早期 panic
$proc = Start-Process -FilePath $Exe `
  -WorkingDirectory $Root `
  -PassThru `
  -WindowStyle Hidden `
  -RedirectStandardError $BootstrapErr

Set-Content -Path $PidFile -Value $proc.Id -Encoding ascii
Write-Host "Started pid=$($proc.Id)"
Write-Host "Config: $ConfigDir"
Write-Host "Logs:   $(Join-Path $LogDir 'ensis-cloud.log')"
Write-Host "API:    http://127.0.0.1:8090/niuma/cloud/healthz"
Write-Host "Ops:    http://127.0.0.1:8090/niuma/ops/"
