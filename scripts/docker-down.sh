#!/usr/bin/env bash
# 停止 OpsHub 容器化部署（依赖三件套 + 三个应用服务），保留数据卷。
# 用法: bash scripts/docker-down.sh
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

docker compose -f docker-compose.yml -f docker-compose.app.yml down
echo "stopped. (数据卷已保留；彻底删除加 -v)"
