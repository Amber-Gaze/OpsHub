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

## 特性

- **配置分层浏览**：`GET /configs/tree[/business[/module[/name]]]`
- **细粒度权限**：按业务/模块/具体项授权（`/policies/config-grant`），配置中心按 scope 过滤执行
- **Redis 混存**：etcd 主存储 + Redis L1 读缓存（cache-aside）；IAM 登出令牌黑名单
- **统一入口**：网关透传用户/策略管理到 IAM，后续接口自动复用
