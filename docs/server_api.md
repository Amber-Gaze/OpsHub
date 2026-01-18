#### 用户认证模块（/api/v1/auth）
| 方法 | 路径 | 描述 |
|------|------|------|
| `POST` | `/api/v1/auth/login` | 登录，返回 JWT Token |
| `GET` | `/api/v1/auth/me` | 获取当前用户信息（需携带 Token） |
| `POST` | `/api/v1/auth/logout` | 注销（可选，JWT 无状态） |

#### 用户管理模块（/api/v1/users）→ 仅管理员可访问
| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/api/v1/users` | 获取所有用户列表 |
| `GET` | `/api/v1/users/{id}` | 获取指定用户详情 |
| `POST` | `/api/v1/users` | 创建新用户（需指定 role） |
| `PUT` | `/api/v1/users/{id}` | 更新用户信息（密码、角色等） |
| `DELETE` | `/api/v1/users/{id}` | 删除用户 |


#### 项目管理模块（/api/v1/projects）
| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/api/v1/permissions` | 获取所有权限策略（Casbin 规则） |
| `POST` | `/api/v1/permissions` | 添加新权限规则 |
| `DELETE` | `/api/v1/permissions` | 删除权限规则 |


#### 配置管理模块（/api/v1/configs）
| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/api/v1/configs/groups` | 获取所有配置分组（支持分页） |
| `GET` | `/api/v1/configs/groups/{id}` | 获取指定分组详情 |
| `POST` | `/api/v1/configs/groups` | 创建新分组（支持 parent_id） |
| `PUT` | `/api/v1/configs/groups/{id}` | 更新分组信息 |
| `DELETE` | `/api/v1/configs/groups/{id}` | 删除分组（级联删除子项） |

##### 子模块：配置项（Item）
| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/api/v1/configs/items` | 获取所有配置项（支持 filter by group） |
| `GET` | `/api/v1/configs/items/{id}` | 获取指定配置项 |
| `POST` | `/api/v1/configs/items` | 创建新配置项（指定 group_id, key_name, value） |
| `PUT` | `/api/v1/configs/items/{id}` | 更新配置项（自动创建历史版本） |
| `DELETE` | `/api/v1/configs/items/{id}` | 删除配置项 |

#### 子模块：配置历史（History）
| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/api/v1/configs/items/{id}/history` | 获取指定配置项的历史版本列表 |
| `GET` | `/api/v1/configs/items/{id}/history/{version}` | 获取指定版本的配置内容 |
| `POST` | `/api/v1/configs/items/{id}/rollback?version=5` | 回退到指定版本 |


#### 服务发现模块（/api/v1/services）
| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/api/v1/services` | 获取所有服务实例列表 |
| `GET` | `/api/v1/services/{service_name}` | 获取指定服务的所有实例 |
| `GET` | `/api/v1/services/{service_name}/{instance_id}` | 获取指定实例详情 |
| `POST` | `/api/v1/services/register` | 服务注册（客户端上报） |
| `POST` | `/api/v1/services/heartbeat` | 心跳续约（客户端定时调用） |


#### 审计日志模块（/api/v1/audit）→ 仅管理员或审计员可访问
| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/api/v1/audit/logs` | 获取审计日志列表（支持时间范围、操作类型过滤） |
| `GET` | `/api/v1/audit/logs/{id}` | 获取单条审计记录详情 |


#### 7. 可视化数据接口（/api/v1/dashboard）
| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/api/v1/dashboard/service-status` | 返回服务健康状态统计（用于饼图） |
| `GET` | `/api/v1/dashboard/config-trend` | 返回配置变更趋势（用于折线图） |
| `GET` | `/api/v1/dashboard/top-services` | 返回最活跃的服务列表（用于柱状图） |

