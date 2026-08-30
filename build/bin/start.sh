#!/usr/bin/env bash
# 启动全部 OpsHub 服务（gateway / iam / config-center）。
# 兼容两种调用形态，统一以 output/ 为运行目录（日志统一写 output/logs/）：
#   - 源码树:    bash build/bin/start.sh   （二进制在 output/bin/，build.sh 生成）
#   - 部署目录:  cd output/bin && ./start.sh
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
if [ -x "$SCRIPT_DIR/opshub-gateway" ]; then
    # 部署形态：脚本与二进制同目录（output/bin/）
    RUN_DIR="$(dirname "$SCRIPT_DIR")"
else
    # 源码树形态：脚本在 build/bin/，二进制在 output/bin/
    ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
    RUN_DIR="$ROOT/output"
fi

BIN_DIR="$RUN_DIR/bin"
CONF="$RUN_DIR/conf/ops_hub.yaml"
LOG_DIR="$RUN_DIR/logs"

SERVICES="opshub-gateway opshub-iam opshub-config"
mkdir -p "$LOG_DIR"
cd "$RUN_DIR"

for p in $SERVICES; do
    if [ ! -x "$BIN_DIR/$p" ]; then
        echo "[warn] $p binary not found ($BIN_DIR/$p，请先 bash scripts/build.sh)"
        continue
    fi
    if pgrep -f "\./$p -c" >/dev/null 2>&1 || pgrep -x "$p" >/dev/null 2>&1; then
        echo "[skip] $p already running"
        continue
    fi
    nohup "./bin/$p" -c "$CONF" >>"$LOG_DIR/$p.stdout" 2>>"$LOG_DIR/$p.stderr" &
    echo "[start] $p pid=$!"
done

sleep 1
echo "started. 日志见 $LOG_DIR/ (应用日志 ./logs 同步在该目录)"