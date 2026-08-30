#!/usr/bin/env bash
# OpsHub 编译脚本：构建 gateway / iam / config-center 三个服务二进制。
# 用法:
#   bash scripts/build.sh            仅构建
#   bash scripts/build.sh pack       构建并打 tar.gz 包
# 环境变量:
#   VERSION   覆盖版本号（默认取 CHANGELOG.md 最新版本）
#   GOOS/GOARCH 交叉编译（如 GOOS=linux GOARCH=amd64 bash scripts/build.sh）
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

SYSTEM=$(uname -s)
case "$SYSTEM" in
    Linux|Darwin) ;;
    *) echo "build only support Linux/Darwin, got: $SYSTEM"; exit 1 ;;
esac

PROJECT=$(basename "$ROOT")

# 版本号：环境变量优先，否则取 CHANGELOG.md 最新 "## [x.y.z]"
if [[ -z "${VERSION:-}" ]]; then
    VERSION=$(grep -E '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -1 | sed -E 's/^## \[([^]]+)\].*/\1/')
    VERSION=${VERSION:-0.0.0}
fi
GITHASH=$(git rev-parse --short=8 HEAD 2>/dev/null || echo "nogit")
DATESTR=$(date '+%Y%m%d%H%M%S')

# 需要构建的服务（对应 cmd 下的目录）
SERVICES="opshub-gateway opshub-iam opshub-config"

# 跨平台 md5 输出（Linux 用 md5sum，macOS 用 md5）
md5_cmd() {
    local file=$1
    if command -v md5sum >/dev/null 2>&1; then
        md5sum "$file"
    elif command -v md5 >/dev/null 2>&1; then
        echo "$(md5 -q "$file")  $file"
    else
        echo "no-md5-tool  $file"
    fi
}

build() {
    rm -rf output
    mkdir -p output/{data,conf,bin,logs}

    # 部署脚本 + 生效配置（跳过空/死文件）
    cp build/bin/* output/bin/
    cp build/conf/ops_hub.yaml output/conf/
    # 运控台前端页面（网关以 output/web 提供）
    if [ -d web ]; then
        cp -r web output/web
    fi

    export CGO_ENABLED=0
    export GOFLAGS="-trimpath"

    for svc in $SERVICES; do
        echo "building $svc  version=$VERSION git=$GITHASH date=$DATESTR"
        local_flags="-X 'main.version=$svc $VERSION ($GITHASH $DATESTR)'"
        go build -ldflags "$local_flags" -o "$ROOT/output/bin/$svc" "./cmd/$svc"
    done

    # 生成二进制校验文件
    : > output/md5sum.file
    for f in output/bin/*; do
        if [ -f "$f" ] && [ -x "$f" ]; then
            md5_cmd "$f" >> output/md5sum.file
        fi
    done
    echo "build done: output/bin -> $(ls output/bin | tr '\n' ' ')"
}

pack() {
    local pkg="$PROJECT-$VERSION-$GITHASH-$DATESTR"
    cp -rf output "$pkg"
    tar czf "$pkg.tar.gz" "$pkg"
    rm -rf "$pkg"
    echo "pack done: $pkg.tar.gz"
}

main() {
    build
    if [[ "${1:-}" == "pack" ]]; then
        pack
    fi
}

main "$@"