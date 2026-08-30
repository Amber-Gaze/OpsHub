#!/usr/bin/env bash
# 停止全部 OpsHub 服务（优雅 TERM，超时后强杀）。
set -uo pipefail

SERVICES="opshub-gateway opshub-iam opshub-config"

for p in $SERVICES; do
    # macOS 进程名截断为 15 字符（opshub-gateway/iam/config 均 <15），pgrep -x 可精确匹配
    pids=$(pgrep -f "\./$p -c" 2>/dev/null || true)
    [ -z "$pids" ] && pids=$(pgrep -x "$p" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        echo "[stop] $p pid=$pids"
        kill $pids 2>/dev/null || true
        sleep 1
        for pid in $pids; do
            if kill -0 "$pid" 2>/dev/null; then
                echo "[kill] $p pid=$pid (force)"
                kill -9 "$pid" 2>/dev/null || true
            fi
        done
    else
        echo "[skip] $p not running"
    fi
done

echo "stopped."