#!/usr/bin/env bash
# 重启 Go 服务（Linux）
# 用法：bash script/restart.sh
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
bash "$DIR/stop.sh"
sleep 0.4
bash "$DIR/start.sh"
