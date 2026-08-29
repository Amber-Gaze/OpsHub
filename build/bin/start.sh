#!/usr/bin/env bash
# 启动全部 OpsHub 服务（gateway / iam / config-center）。
# 工作目录应为 output/bin；配置默认读取 ../conf/ops_hub.yaml。
set -euo pipefail

cd "$(dirname "$0")"

SERVICES="opshub-gateway opshub-iam opshub-config"
CONF="../conf/ops_hub.yaml"

mkdir -p ../logs

for p in $SERVICES; do
    if [ -x "./$p" ]; then
        if pgrep -f "\./$p -c" >/dev/null 2>&1 || pgrep -x "$p" >/dev/null 2>&1; then
            echo "[skip] $p already running"
            continue
        fi
        nohup "./$p" -c "$CONF" >"../logs/$p.stdout" 2>"../logs/$p.stderr" &
        echo "[start] $p pid=$!"
    else
        echo "[warn] $p binary not found (请先运行 bash scripts/build.sh)"
    fi
done

sleep 1
echo "started. 日志见 ../logs/"