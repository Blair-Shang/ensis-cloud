#!/usr/bin/env bash
# 构建 Cloud：Admin 前端 → 嵌入目录 → Go 单二进制（Linux）
# 用法：bash script/build.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

mkdir -p "$ROOT/run"

echo "==> pnpm install"
pnpm install

echo "==> admin build (embed -> server/internal/web/dist)"
pnpm exec vue-tsc --noEmit -p admin/tsconfig.json
pnpm exec vite build --config admin/vite.config.ts

BIN="$ROOT/run/ensis-cloud"
echo "==> go build -> $BIN"
(
  cd "$ROOT/server"
  go build -o "$BIN" ./cmd/server
)

echo "OK: $BIN"
echo "Start: bash script/start.sh"
