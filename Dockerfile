# syntax=docker/dockerfile:1
# OpsHub 单服务镜像（多阶段构建）。
# 通过构建参数 TARGET 选择打包哪个服务：
#   docker build --build-arg TARGET=opshub-gateway -t opshub-gateway .
#   docker build --build-arg TARGET=opshub-iam     -t opshub-iam .
#   docker build --build-arg TARGET=opshub-config  -t opshub-config .
# 容器化整体部署请用 docker-compose.app.yml（自动按服务传 TARGET）。

# ---- 构建阶段 ----
FROM golang:1.25-alpine AS builder
WORKDIR /src

# 先只拷贝依赖清单，最大化利用构建缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGET=opshub-gateway
RUN CGO_ENABLED=0 GOFLAGS="-trimpath" \
    go build -o /out/opshub-server ./cmd/$TARGET

# ---- 运行阶段 ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S opshub && adduser -S -G opshub -H -h /app opshub

WORKDIR /app
COPY --from=builder /out/opshub-server /app/bin/opshub-server
# 容器内配置：host 全用 compose 服务名（mysql/redis/etcd/iam/config）
COPY build/conf/ops_hub.docker.yaml /app/conf/ops_hub.yaml
# 运控台前端（gateway 静态服务使用）
COPY web /app/web

RUN mkdir -p /app/logs && chown -R opshub:opshub /app
USER opshub

# gateway 8001 / iam 8004 / config 8007（按 TARGET 只有一个生效）
EXPOSE 8001 8004 8007
ENTRYPOINT ["/app/bin/opshub-server", "-c", "/app/conf/ops_hub.yaml"]
