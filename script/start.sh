#!/usr/bin/env bash
# 启动内嵌 Admin 的 Go 服务（Linux）
# 用法：bash script/start.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

RUN="$ROOT/run"
CONFIG="$ROOT/config"
LOGS="$ROOT/logs"
DATA="$ROOT/data"
BIN="$RUN/ensis-cloud"
PID_FILE="$RUN/ensis-cloud.pid"
BOOTSTRAP_ERR="$LOGS/bootstrap.err"

if [[ ! -x "$BIN" && ! -f "$BIN" ]]; then
  echo "Binary not found: $BIN" >&2
  echo "Run: bash script/build.sh" >&2
  exit 1
fi
chmod +x "$BIN" 2>/dev/null || true

if [[ -f "$PID_FILE" ]]; then
  old="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [[ -n "${old:-}" ]] && kill -0 "$old" 2>/dev/null; then
    echo "Already running (pid=$old). Use script/restart.sh or stop first."
    exit 0
  fi
fi

mkdir -p "$RUN" "$CONFIG" "$LOGS" "$DATA"

if [[ -f "$CONFIG/app.yaml.example" && ! -f "$CONFIG/app.yaml" ]]; then
  cp "$CONFIG/app.yaml.example" "$CONFIG/app.yaml"
  echo "Created $CONFIG/app.yaml from example"
  echo "Edit jwt.secret / db.dsn / staff.password before production use."
fi

# 应用日志由 zap 写入 logs/ensis-cloud.log；stdout 丢弃，stderr 仅作启动兜底
nohup "$BIN" >/dev/null 2>>"$BOOTSTRAP_ERR" &
echo $! >"$PID_FILE"
echo "Started pid=$(cat "$PID_FILE")"
echo "Config: $CONFIG"
echo "Logs:   $LOGS/ensis-cloud.log"
echo "API:    http://127.0.0.1:8090/niuma/cloud/healthz"
echo "Ops:    http://127.0.0.1:8090/niuma/ops/"
