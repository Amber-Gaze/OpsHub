# OpsHub
A practice project for learning Go.The project is divided into three submodules.  
1. Gateway: focuses on external gateway services, implementing JWT verification and rate limiting.   
2. Auth: handles authentication and authorization management.   
3. Config Center: manages production configurations.  

The three components collectively accomplish a function that enables real-time updates and deployment of online configurations.  

这是一个我自己用于学习Go语言的实践项目。该项目分为三个子模块：  
1. 网关：专注于外部网关服务，实现JWT验证和速率限制。  
2. 认证：处理身份验证与授权管理。  
3. 配置中心：管理生产环境配置。  

这三个组件共同实现实时更新和部署在线配置运控台。

## 文档

- **整体逻辑说明（推荐先读）**：[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
  - 三模块职责与端口、请求/鉴权链路、配置「业务-模块-列表」分层模型
  - 权限模型（scope：`config/pay/**` 等）、Redis 混存（etcd 主存 + Redis L1 缓存）方案
  - 接口清单与错误语义（读无权限 404、写无权限 403）
- **外部接入指南（对接公司 IAM / 下游业务接入）**：[docs/EXTERNAL_IAM_INTEGRATION.md](docs/EXTERNAL_IAM_INTEGRATION.md)
  - 对外稳定协议（JWT / scope / HMAC 签名 / X-Auth-* 头）、接口清单与 curl 示例
  - 下游服务拉取配置（pkg/configclient + demo）
  - 替换为公司自有 IAM 的 `GrantEngine` / `UserStore` 对接说明与代码骨架

## 特性

- **配置分层浏览**：`GET /configs/tree[/business[/module[/name]]]`
- **细粒度权限**：按业务/模块/具体项授权（`/policies/config-grant`），配置中心按 scope 过滤执行
- **Redis 混存**：etcd 主存储 + Redis L1 读缓存（cache-aside）；IAM 登出令牌黑名单
- **统一入口**：网关透传用户/策略管理到 IAM，后续接口自动复用

## 快速开始

依赖：Go 1.25+、Docker（Compose v2）。两种部署模式任选其一，**不要同时运行**（端口 8001/8101/8201 会冲突）。

### 模式 A：本地开发（依赖跑容器，服务跑宿主机）

```bash
make dev          # 一键：起 mysql/redis/etcd → 构建 → 启动三个服务 → 等就绪
make dev-down     # 停止三个本地服务（依赖容器保留）
```

### 模式 B：全容器化（依赖 + 三个服务都进容器）

```bash
make docker-up    # 构建镜像并启动 6 个容器（mysql/redis/etcd + gateway/iam/config）
make docker-down  # 停止全部容器（保留数据卷）
```

> 容器内使用 `build/conf/ops_hub.docker.yaml`（host 全为 compose 服务名），
> 本地使用 `build/conf/ops_hub.yaml`（host 为 localhost）。数据卷共享，切换模式配置不丢。

### 访问与账号

- 控制台：http://127.0.0.1:8001/ （登录页「没有账号？注册」）
- **首个注册用户自动成为管理员**；之后注册的为普通用户（可被管理员授权）
- 登录接口：`POST /login`，配置接口均需 `Authorization: Bearer <token>`
