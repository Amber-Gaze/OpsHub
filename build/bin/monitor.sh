#!/usr/bin/env bash
# 简易守护进程：周期检查三个服务进程与健康端口，挂掉即拉起。
# 工作目录应为 output/bin；配置默认读取 ../conf/ops_hub.yaml。
set -uo pipefail

ROOT=$(cd "$(dirname "$0")" && pwd)
cd "$ROOT"

mkdir -p ../logs
exec >> ../logs/.monitor.log 2>&1

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }

# 服务 -> 对外 HTTP 端口（用于健康检查）
declare -A PORTS=(
    [opshub-gateway]=8001
    [opshub-iam]=8004
    [opshub-config]=8007
)
CONF="../conf/ops_hub.yaml"
CHECK_INTERVAL="${CHECK_INTERVAL:-30}" # 秒

is_running() {
    pgrep -f "\./$1 -c" >/dev/null 2>&1 || pgrep -x "$1" >/dev/null 2>&1
}

port_open() {
    local port=$1
    if command -v nc >/dev/null 2>&1; then
        nc -z 127.0.0.1 "$port" >/dev/null 2>&1
    elif command -v ss >/dev/null 2>&1; then
        ss -ltn | grep -q ":$port"
    else
        (echo > /dev/tcp/127.0.0.1/$port) >/dev/null 2>&1
    fi
}

log "### Monitor Start (interval=${CHECK_INTERVAL}s) ###"

while :; do
    for p in "${!PORTS[@]}"; do
        port=${PORTS[$p]}
        if is_running "$p" && port_open "$port"; then
            continue
        fi
        if is_running "$p"; then
            log "process $p alive but port $port dead, kill & restart"
            pkill -f "\./$p -c" 2>/dev/null || true
            sleep 1
        else
            log "process $p down, restarting"
        fi
        if [ -x "./$p" ]; then
            nohup "./$p" -c "$CONF" >>"../logs/$p.stdout" 2>>"../logs/$p.stderr" &
            log "restarted $p pid=$!"
        else
            log "binary ./$p not found, skip"
        fi
    done
    sleep "$CHECK_INTERVAL"
done

