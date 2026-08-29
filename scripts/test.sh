#!/usr/bin/env bash
# OpsHub 测试脚本：一次运行全量单元测试并合并覆盖率。
# 用法:
#   bash scripts/test.sh          默认测试 + 覆盖率汇总
#   bash scripts/test.sh -race    启用竞态检测
#   bash scripts/test.sh -v       详细输出
#   bash scripts/test.sh -vet     额外运行 go vet ./...
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

RACE=""
VERBOSE=""
RUN_VET=0

for arg in "$@"; do
    case "$arg" in
        -race) RACE="-race" ;;
        -v) VERBOSE="-v" ;;
        -vet) RUN_VET=1 ;;
        *) echo "unknown arg: $arg (支持 -race / -v / -vet)"; exit 1 ;;
    esac
done

if [ "$RUN_VET" -eq 1 ]; then
    echo "==> go vet ./..."
    go vet ./...
fi

echo "==> go test $RACE $VERBOSE -coverprofile=cover.out ./..."
go test $RACE $VERBOSE -coverprofile=cover.out ./...

echo "==> 覆盖率汇总"
go tool cover -func=cover.out | tail -1
