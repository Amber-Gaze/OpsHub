#!/bin/sh

ROOT=$(cd $(dirname $0) && pwd)
cd $ROOT/bin

export PATH=$PATH:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

PROGRAM="OpsHub" 

exec &> $ROOT/log/.monitor.log
log() {
  echo "[$(date "+%Y-%m-%d %H:%M:%S")] $@"
}


find_pid() {
    local pname=$1
    pid=$(ps -eo pid,command | grep -v grep | grep "$pname" | awk -v target="$pname" '
        {
            pid = $1
            cmd = ""
            for (i = 2; i <= NF; i++) {
                cmd = cmd $i (i == NF ? "" : " ")
            }

            n = split(cmd, parts, " ")
            lastArg = parts[n]
            m = split(lastArg, pathParts, "/")
            binname = pathParts[m]

            if (binname == target) {
                printf("find PID=%s, BIN=%s, CMD=%s\n", pid, binname, cmd) > "/dev/stderr"
                print pid
            }
        }
    ')
    echo "$pid"
}

# 定义函数（用法：kill_all <信号量> <进程名>)
kill_all() {
    local signal="TERM"
    local pid=""

    case "$1" in
        -9|-KILL|-kill)
            signal="9"
            pid="$2"
            ;;
        -TERM|-term)
            signal="TERM"
            pid="$2"
            ;;
        -HUP|-hup)
            signal="HUP"
            pid="$2"
            ;;
        -*)
            log "Unknown signal: $1"
            return 1
            ;;
        *)
            pid="$1"
            ;;
    esac

    if [ -z "$pid" ]; then
        log "Error: process name not specified"
        return 1
    fi

    [ -n "$pid" ] && kill -"$signal" $pid
}

# 带参数命令启动程序无法查找并kill
stop_process() {
    local signal="$1"
    local pname="$2"
    local killed=0

    if command -v killall >/dev/null 2>&1; then
        log "Using system killall for $pname"
        killall -9 "$pname" 2>/dev/null && killed=1 || log "System killall: no $pname process found"
    else
        log "System killall not found, will use local killAll for $pname"
    fi

    local pid=$(find_pid "$pname")
    # 检查进程是否还存在
    if  [ -n "$pid" ]; then
        log "Processes of $pname, pid=$pid still running, using local kill_all"
        kill_all $signal "$pid"
    else
        [ $killed -eq 1 ] && log "Processes of $pname successfully killed by system killall"
    fi
}
log "### Monitor Start ###"

# 初始化计数器，每10分钟检查一次端口
counter=0
port_check_interval=10 
port=8002

check_port() {
    if command -v ss &>/dev/null; then
        if ss -ltn | grep -q ":$port"; then
            log "Port $port is alive (via ss)"
            return 0
        else
            log "Port $port is dead (via ss)"
        fi
    else
        log "ss not found, falling back to /dev/tcp check..."
        if (echo > /dev/tcp/127.0.0.1/$port) &>/dev/null; then
            log "Port $port is alive (via /dev/tcp)"
            return 0
        else
            log "Port $port is dead (via /dev/tcp)"
        fi
    fi

    # 如果走到这里，说明端口不可用，杀掉程序
    stop_process -9 "$PROGRAM"
}

while :
do
    # 每10分钟检查一次端口状态
    if [ $counter -eq 0 ]; then
        check_port
    fi

    pids=$(pidof -x $PROGRAM)
    if [ -z "$pids" ]; then
        log "Start $PROGRAM ..."

        export GIN_MODE=release
        ulimit -v unlimited
        nohup $ROOT/bin/$PROGRAM &

        log "Start $PROGRAM success: $(pidof -x $PROGRAM)"
    fi

    # 更新计数器
    counter=$((counter + 1))
    if [ $counter -ge $port_check_interval ]; then
        counter=0
    fi

    sleep 60
done

log "### Monitor Ended ###"

