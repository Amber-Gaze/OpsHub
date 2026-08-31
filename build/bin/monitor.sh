#!/usr/bin/env bash
# 简易守护进程：周期检查三个服务进程与健康端口，挂掉即拉起。
# 兼容两种调用形态，统一以 output/ 为运行目录（与 start.sh 一致）。
set -uo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
if [ -x "$SCRIPT_DIR/opshub-gateway" ]; then
    RUN_DIR="$(dirname "$SCRIPT_DIR")"
else
    ROOT=$(cd "$SCRIPT_DIR/../.." && pwd)
    RUN_DIR="$ROOT/output"
fi

BIN_DIR="$RUN_DIR/bin"
CONF="$RUN_DIR/conf/ops_hub.yaml"
LOG_DIR="$RUN_DIR/logs"

mkdir -p "$LOG_DIR"
exec >>"$LOG_DIR/.monitor.log" 2>&1

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }

# 服务 -> 对外 HTTP 端口（用于健康检查）
declare -A PORTS=(
    [opshub-gateway]=8001
    [opshub-iam]=8101
    [opshub-config]=8201
)
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

log "### Monitor Start (dir=$RUN_DIR interval=${CHECK_INTERVAL}s) ###"

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
        if [ -x "$BIN_DIR/$p" ]; then
            (cd "$RUN_DIR" && nohup "./bin/$p" -c "$CONF" >>"$LOG_DIR/$p.stdout" 2>>"$LOG_DIR/$p.stderr" & echo $! >"$RUN_DIR/.pid.$p")
            log "restarted $p pid=$(cat "$RUN_DIR/.pid.$p" 2>/dev/null)"
        else
            log "binary $BIN_DIR/$p not found, skip"
        fi
    done
    sleep "$CHECK_INTERVAL"
done

