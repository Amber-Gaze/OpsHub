#!/usr/bin/env bash
# OpsHub 一键本地开发：起依赖三件套 → 构建 → 启动三个服务 → 等待就绪。
# 用法:
#   make dev            等价于 bash scripts/dev.sh
#   make dev-down       停止本地服务（依赖容器保留）
# 说明: 依赖容器（mysql/redis/etcd）用 docker-compose.yml 起；服务本体在宿主机本地部署。
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

echo "[1/4] 启动依赖容器 (mysql/redis/etcd) ..."
docker compose up -d

wait_healthy() {
    local name=$1
    printf "    waiting %s healthy" "$name"
    for _ in $(seq 1 60); do
        local st
        st=$(docker inspect --format '{{.State.Health.Status}}' "$name" 2>/dev/null || echo none)
        if [ "$st" = "healthy" ]; then
            echo " ok"
            return 0
        fi
        printf '.'
        sleep 2
    done
    echo " timeout"
    return 1
}
wait_healthy opshub-mysql
wait_healthy opshub-redis
wait_healthy opshub-etcd

echo "[2/4] 构建三个服务 (output/bin) ..."
bash scripts/build.sh

echo "[3/4] 启动服务 (后台) ..."
bash build/bin/start.sh

echo "[4/4] 等待服务就绪 ..."
B=http://127.0.0.1
g=--; i=--; c=--
for _ in $(seq 1 30); do
    curl -sf -o /dev/null "$B:8001/healthz" && g=ok || g=fail
    curl -sf -o /dev/null "$B:8101/readyz"  && i=ok || i=fail
    curl -sf -o /dev/null "$B:8201/readyz"  && c=ok || c=fail
    [ "$g" = ok ] && [ "$i" = ok ] && [ "$c" = ok ] && break
    sleep 1
done

echo
echo "gateway :8001 $g | iam :8101 $i | config-center :8201 $c"
echo
echo "访问控制台: http://127.0.0.1:8001/"
echo "  首个注册用户自动成为管理员；日志见 output/logs/；停止用 make dev-down"
