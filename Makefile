# OpsHub 常用命令入口
# 用法: make build | make test | make vet | make run | make clean | make pack
#       make dev | make dev-down | make docker-up | make docker-down

SERVICES := opshub-gateway opshub-iam opshub-config

.PHONY: build pack test vet run stop dev dev-down docker-up docker-down clean tidy

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

## 本地启动全部服务（后台，需先 make build）
run:
	bash build/bin/start.sh

## 停止本地服务
stop:
	bash build/bin/stop.sh

## 一键本地开发：起依赖三件套(docker compose) + 构建 + 启动三个服务 + 等就绪
dev:
	bash scripts/dev.sh

## 停止本地开发的服务（依赖容器保留）
dev-down:
	bash build/bin/stop.sh

## 容器化部署：构建镜像并启动 mysql/redis/etcd + gateway/iam/config
docker-up:
	bash scripts/docker-up.sh

## 停止容器化部署（保留数据卷）
docker-down:
	bash scripts/docker-down.sh

## 清理构建产物
clean:
	rm -rf output cover.out *.tar.gz

## 整理 go.mod
tidy:
	go mod tidy
