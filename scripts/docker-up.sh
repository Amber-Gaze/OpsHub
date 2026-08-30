#!/usr/bin/env bash
# OpsHub 容器化一键部署：起依赖三件套 + 构建并启动三个应用服务。
# 用法: bash scripts/docker-up.sh
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

echo "[1/2] 构建镜像并启动依赖 + 应用服务 ..."
docker compose -f docker-compose.yml -f docker-compose.app.yml up -d --build

echo "[2/2] 等待就绪 ..."
B=http://127.0.0.1
g=--; i=--; c=--
for _ in $(seq 1 60); do
    curl -sf -o /dev/null "$B:8001/healthz" && g=ok || g=fail
    curl -sf -o /dev/null "$B:8004/readyz"  && i=ok || i=fail
    curl -sf -o /dev/null "$B:8007/readyz"  && c=ok || c=fail
    [ "$g" = ok ] && [ "$i" = ok ] && [ "$c" = ok ] && break
    sleep 2
done

echo
echo "gateway :8001 $g | iam :8004 $i | config-center :8007 $c"
echo
echo "访问控制台: http://127.0.0.1:8001/  (首个注册用户自动成为管理员)"
echo "容器日志:   docker compose -f docker-compose.yml -f docker-compose.app.yml logs -f"
