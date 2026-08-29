# OpsHub 常用命令入口
# 用法: make build | make test | make vet | make run | make clean | make pack

SERVICES := opshub-gateway opshub-iam opshub-config

.PHONY: build pack test vet run clean tidy

## 构建全部服务到 output/bin
build:
	bash scripts/build.sh

## 构建并打 tar.gz 包
pack:
	bash scripts/build.sh pack

## 单元测试 + 覆盖率
test:
	bash scripts/test.sh

## 竞态检测测试
test-race:
	bash scripts/test.sh -race

## 静态检查
vet:
	go vet ./...

## 本地启动全部服务（后台）
run:
	bash build/bin/start.sh

## 停止本地服务
stop:
	bash build/bin/stop.sh

## 清理构建产物
clean:
	rm -rf output cover.out *.tar.gz

## 整理 go.mod
tidy:
	go mod tidy
