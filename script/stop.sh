#!/usr/bin/env bash
# 停止 Go 服务（Linux）
# 用法：bash script/stop.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PID_FILE="$ROOT/run/ensis-cloud.pid"

if [[ ! -f "$PID_FILE" ]]; then
  echo "Not running (no pid file)."
  exit 0
fi

pid="$(cat "$PID_FILE" 2>/dev/null || true)"
if [[ -z "${pid:-}" ]]; then
  rm -f "$PID_FILE"
  echo "Not running."
  exit 0
fi

if kill -0 "$pid" 2>/dev/null; then
  kill "$pid" 2>/dev/null || true
  for _ in 1 2 3 4 5 6 7 8; do
    if ! kill -0 "$pid" 2>/dev/null; then
      break
    fi
    sleep 0.4
  done
  if kill -0 "$pid" 2>/dev/null; then
    kill -9 "$pid" 2>/dev/null || true
  fi
  echo "Stopped pid=$pid"
else
  echo "Process $pid not found."
fi

rm -f "$PID_FILE"
