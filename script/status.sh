#!/usr/bin/env bash
# 查看 Go 服务状态（Linux）
# 用法：bash script/status.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PID_FILE="$ROOT/run/ensis-cloud.pid"
HEALTH="http://127.0.0.1:8090/niuma/cloud/healthz"

if [[ ! -f "$PID_FILE" ]]; then
  echo "stopped (no pid file)"
  exit 1
fi

pid="$(cat "$PID_FILE" 2>/dev/null || true)"
if [[ -z "${pid:-}" ]]; then
  echo "stopped (empty pid file)"
  exit 1
fi

if ! kill -0 "$pid" 2>/dev/null; then
  echo "stopped (stale pid=$pid)"
  exit 1
fi

echo "running pid=$pid"
if command -v curl >/dev/null 2>&1; then
  if code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 3 "$HEALTH")"; then
    echo "healthz $code $HEALTH"
    [[ "$code" == "200" ]]
  else
    echo "healthz unreachable $HEALTH"
    exit 1
  fi
fi
