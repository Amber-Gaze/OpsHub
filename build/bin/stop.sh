#!/usr/bin/env bash
# 停止全部 OpsHub 服务（优雅 TERM，超时后强杀）。
set -uo pipefail

cd "$(dirname "$0")"

SERVICES="opshub-gateway opshub-iam opshub-config"

for p in $SERVICES; do
    pids=$(pgrep -f "\./$p -c" 2>/dev/null || true)
    if [ -z "$pids" ]; then
        pids=$(pgrep -x "$p" 2>/dev/null || true)
    fi
    if [ -n "$pids" ]; then
        echo "[stop] $p pid=$pids"
        kill $pids 2>/dev/null || kill -9 $pids 2>/dev/null || true
    else
        echo "[skip] $p not running"
    fi
done

echo "stopped."