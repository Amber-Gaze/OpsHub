#!/usr/bin/env bash 

ROOT=$(cd $(dirname $0)/.. && pwd)
cd $ROOT

# 罗列含有测试文件的包目录
DIRS=$(find "." -type f -name '*_test.go' -exec dirname {} \;)

# 逐目录执行单元测试
for DIR in $DIRS; do
	go test -gcflags=-l -coverprofile=cover.out $DIR/...
done
