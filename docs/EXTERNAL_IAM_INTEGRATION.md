# OpsHub 外部 IAM 接入指南（Integration Guide）

> 给接入方的对接说明：如何把 OpsHub（运控配置台）接入你们已有的 IAM / 权限体系，
> 以及下游业务服务如何通过 OpsHub 鉴权与拉取配置。
> 配套代码结构见 [`ARCHITECTURE.md`](ARCHITECTURE.md) §8「可替换适配点」。

---

## 1. 系统组成与端口

OpsHub 由三个可独立部署的 Go 服务组成，**对外统一入口是网关（8001）**。

| 模块 | 职责 | HTTP 端口 | 对外可见 |
|---|---|---|---|
| `gateway` | 统一入口：JWT 校验、限流、向 IAM 取 scope、转发配置/用户/策略、静态前端 | 8001 | ✅ |
| `iam` | 身份验证（登录/注册/登出/刷新）+ 授权决策（默认 casbin RBAC）+ scope 签发 | 8101 | ❌（仅网关内部） |
| `config-center` | 配置 CRUD / 分层浏览 / 历史审计 / 下游 pull，etcd 主存 + Redis L1 | 8201 | ❌（仅网关内部） |

依赖：MySQL（用户 + casbin 策略）、etcd（配置主存 + 审计）、Redis（配置缓存 + 登出黑名单，可选）。

**接入方只需对接网关 8001**，不需要知道 iam / config-center 的存在。

---

## 2. 三种接入视角

| 视角 | 适用 | 怎么做 |
|---|---|---|
| **A. 直接用内置 IAM** | 没有现成 IAM，想开箱即用 | 部署 3 服务，用 `/signup` `/login` 即可；首个注册用户自动成为管理员 |
| **B. 替换为公司自有 IAM/权限中心** | 公司已有 LDAP/AD、OIDC IdP、自研权限中心 | 实现 `GrantEngine` / `UserStore` 接口注入（见 §6），对外协议不变、下游零改动 |
| **C. 下游业务服务接入** | 你们的业务服务要读/同步 OpsHub 的配置 | 走网关：登录 → 授权 → 拉配置（见 §5） |

---

## 3. 对外稳定协议（契约）

以下协议是**稳定契约**：即使内部把 IAM 换成公司自己的实现，这些对外行为也不变。

### 3.1 身份令牌（JWT，Bearer）

- 登录/注册后签发 JWT（HS256，有效期 1h），请求时带 `Authorization: Bearer <token>`。
- Claims 含 `subject`（用户名）、`is_admin` 等；用户管理/策略管理接口按 JWT 判定管理员。

### 3.2 scope（授权列表）与签名决策

网关在转发配置请求前，调用 IAM `/authorize` 拿到「授权决策」，把决策随请求带给配置中心：

```
决策载体 Decision:
  scope|<subject>|<grantsJSON>|<unixTimestamp>
签名 Signature:
  base64(HMAC-SHA256(Decision, decisionSecret))

转发头:
  X-Auth-Decision    = Decision
  X-Auth-Signature   = Signature
  X-Auth-Subject     = subject
```

- `grantsJSON` 形如 `[{"obj":"config/pay/**","act":"read"}]`。
- 配置中心用**同一把 `decisionSecret`** 验签后，按 scope 做细粒度过滤/校验。
- 因此 `decisionSecret` 必须在 iam 与 config-center 之间**保持一致**（生产必须配置化，见 §7）。

### 3.3 授权对象（obj）与动作（act）

| 对象 | 含义 |
|---|---|
| `config/**` | 全部配置 |
| `config/pay/**` | 整个 `pay` 业务 |
| `config/pay/gateway/**` | `pay` 下 `gateway` 模块 |
| `config/pay/gateway/timeout_ms` | 某个具体配置项 |

动作：`read` / `write` / `delete` / `grant` / `*`（全部）。
通配语义：`*` 不跨 `/`，`**` 跨 `/`（doublestar globMatch）。

### 3.4 错误语义

| 状态码 | 场景 |
|---|---|
| 400 | 参数错误 / 登录失败（账号或密码错误） |
| 401 | 未带 token / token 无效或过期 / 已登出（黑名单） |
| 403 | 写/删无权限、非管理员操作管理接口、缺少鉴权上下文 |
| 404 | **读无权限**（不暴露配置是否存在）、资源不存在 |
| 409 | 配置已存在 / 策略已存在 |
| 429 | 网关限流 |
| 502/503 | 上游（IAM/配置中心）不可用 |

> 约定：**读无权限一律 404**（假装资源不存在），**写/删无权限 403**。

---

## 4. HTTP 接口清单（经网关 `http://<host>:8001`）

### 4.1 认证

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| POST | `/signup` | 注册（**首个用户自动成为管理员**） | `{"user","password","email","phone"?}` |
| POST | `/login` | 登录 | `{"user","password"}` |
| POST | `/logout` | 登出（令牌进 Redis 黑名单，可选） | `{"token"}` |
| POST | `/refresh` | 刷新令牌 | `{"token"}` |
| POST | `/scope` | 查询当前令牌用户的配置授权 | `{"token"}` |

登录响应：`{"token","token_type":"Bearer","expires_at"}`。

### 4.2 配置（需 `Authorization: Bearer`，经网关自动附带 scope 决策）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/configs/tree[/business[/module[/name]]]` | 分层浏览（读权限过滤） |
| GET/POST | `/configs`、`/configs/{key}` | 扁平 CRUD（单段 key） |
| PUT/DELETE | `/configs/tree/{business}/{module}/{name}` | 改/删具体项 |
| GET | `/configs/pull` | 下游拉取快照：层级过滤（`business`/`module`/`name`、`path`、`key`）+ 增量（`since=<rev>` 返回变更项与 `removed`） |
| GET | `/configs/history/{key}` | 历史 + 当前值（before/after，前端做 diff） |

### 4.3 用户 / 策略管理（管理员）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/users/` | 用户列表（管理员） |
| GET/PUT/DELETE | `/users/{name}` | 查询 / 改信息 / 删除（本人或管理员） |
| PUT | `/users/{name}/change-passwd` | 改密（本人或管理员） |
| GET | `/users/{name}/grants` | 用户的配置授权（管理员） |
| POST | `/policies/config-grant` | 授权：`{"sub","business","module"?,"item"?,"act"}` |
| POST | `/policies/config-revoke` | 撤销授权（同上） |
| GET | `/policies/` | 策略 + 角色关系（管理员） |
| POST | `/policies/roles` / `/roles/delete` | 角色绑定/解绑 |

### 4.4 服务凭证（AccessKey）与模块订阅（管理员）

下游服务**不用账号密码**，而是用「服务凭证」认证（AccessKeyID + AccessKeySecret 自签 JWT），
再通过「模块订阅」声明可拉取的业务/模块（注册即授 read，**默认只读**）：

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/accesskeys` | 创建服务凭证：`{"username","description"?,"expires"?}`，返回 `access_key_id` + `access_key_secret`（仅此一次） |
| GET | `/accesskeys` | 凭证列表（本人；管理员 `?username=` 查任意） |
| DELETE | `/accesskeys/{keyID}` | 吊销凭证（本人或管理员） |
| PUT | `/services/{name}/modules` | **注册/覆盖模块订阅**：`{"modules":["pay/gateway","common/ratelimit"]}` → 批量授 read |
| GET | `/services/{name}/modules` | 查看某服务订阅的模块 |
| DELETE | `/services/{name}/modules` | 取消单个订阅：body `{"path":"pay/gateway"}` |

> 约定：**注册由管理员完成**（新服务需人工确认开启）；未注册（未授权）的模块，服务拉不到内容。

### 4.5 curl 示例

```bash
B=http://127.0.0.1:8001
# 1) 登录
TOKEN=$(curl -s -X POST $B/login -H 'Content-Type: application/json' \
  -d '{"user":"admin","password":"Admin@12345"}' | sed -nE 's/.*"token":"([^"]+)".*/\1/p')
# 2) 我的权限
curl -s -X POST $B/scope -H 'Content-Type: application/json' -d "{\"token\":\"$TOKEN\"}"
# 3) 拉配置树（网关自动完成 authorize + 附加 scope 决策头）
curl -s -H "Authorization: Bearer $TOKEN" $B/configs/tree
# 4) 下游拉取快照
curl -s -H "Authorization: Bearer $TOKEN" "$B/configs/pull?business=pay"
# 5) 授权普通用户只读 pay 业务（管理员）
curl -s -X POST $B/policies/config-grant -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"sub":"alice","business":"pay","module":"","item":"","act":"read"}'
# 6) 给服务账号 svc-pay 注册模块订阅 + 建凭证（管理员）
curl -s -X PUT $B/services/svc-pay/modules -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"modules":["pay/gateway","common/ratelimit"]}'
curl -s -X POST $B/accesskeys -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"username":"svc-pay","description":"pay 服务"}'
```

---

## 5. 下游业务服务接入（视角 C）

业务服务要「拉取配置并感知变更」时，推荐用内置客户端 `pkg/configclient`：

```go
import (
    "context"
    "github.com/Amber-Gaze/OpsHub/pkg/configclient"
)

ctx := context.Background()
// auth 指向 IAM(8101)，config 指向配置中心(8201)；也可把 config 指向网关(8001)走统一入口
cli := configclient.New("http://127.0.0.1:8101", "http://127.0.0.1:8201")
if err := cli.Login(ctx, "svc-pay", "Svc@2026Pay"); err != nil { /* 登录失败 */ }

items, err := cli.Pull(ctx) // 拉取全部可读配置快照（首次自动 authorize 并携带 X-Auth-* 头）
if err != nil { /* ... */ }
kv := configclient.Snapshot(items) // key -> value，业务代码直接取用

// 按层级/精确拉取：只拉关心的范围，不拉全部
payItems, _ := cli.PullByBusiness(ctx, "pay")            // pay/**
gwItems, _ := cli.PullPath(ctx, "pay/gateway")          // pay/gateway/**
one, _    := cli.PullKey(ctx, "pay/gateway/timeout_ms") // 精确一项

// 增量判断更新：记录上次 revision，之后只拉变更项 + 被删 key
res, err := cli.PullSince(ctx, rev) // rev 为上次收到的全局版本号
if err != nil { /* ... */ }
if res.HasChanged(rev) {
    // 有变化：应用 res.Items（新增/更新），删除 res.Removed，更新 rev = res.Revision
    rev = res.Revision
}
```

完整可运行示例见 `examples/config-consumer`：

```bash
cd examples/config-consumer
go run main.go \
  -auth http://127.0.0.1:8101 -config http://127.0.0.1:8201 \
  -user svc-pay -pass 'Svc@2026Pay' -interval 5s
```

接入要点：
- **服务用凭证认证，不用账号密码**：`LoginWithAccessKey(ctx, accessKeyID, accessKeySecret)` 本地自签 JWT，服务端按 `kid` 验签识别服务账号。
- **模块订阅**：管理员通过 `/services/{name}/modules` 注册服务可拉取的模块（注册即授 read、默认只读）；未注册模块拉不到。
- **按需拉取**：`PullSubscribed(ctx, []string{"pay/gateway","common/ratelimit"})` 只拉订阅模块；或用 `PullPath`/`PullKey` 精确拉。
- **增量更新**：`PullSince(rev)` 基于全局版本号返回变更项与被删 key，`HasChanged` 判断是否有更新，无需每次都全量拉取。
- **本地落盘**：`configclient.WriteTo(items, dir)` 按业务/模块分组写 JSON 到 `<dir>/<business>/<module>.json`；默认数据目录与 `bin/` 平级（`bin/../data`）。
- 完整示例：`examples/config-consumer`（`-access-key-id/-access-key-secret` + `-modules` + `-data-dir`）。
- `configcenter` 直连时 iam/config-center 需内网可达；把 `config` 指向网关 8001 可只暴露一个入口。

---

## 6. 替换为公司自有 IAM（视角 B）

OpsHub 把「鉴权决策」和「用户源」收敛成两个接口，**接口稳定、默认实现可换**。

### 6.1 授权决策：`GrantEngine`（`internal/iam/api/grant_engine.go`）

```go
type GrantEngine interface {
    // 返回用户经角色继承后的全部配置授权（仅 config/ 对象），用于签发 scope
    ImplicitGrants(subject string) ([]casbinx.Grant, error)
    // 精确校验 sub 对 obj 的 act 权限（写/删的最终裁决）
    Enforce(sub, obj, act string) (bool, error)
    // 策略/角色管理（管理端 UI 使用，可由适配层转写）
    ListPolicies() ([][]string, error)
    ListGroupingPolicies() ([][]string, error)
    AddPolicy(sub, obj, act string) (bool, error)
    RemovePolicy(sub, obj, act string) (bool, error)
    AddRoleForUser(user, role string) (bool, error)
    RemoveRoleForUser(user, role string) (bool, error)
}
```

**对接公司权限中心（如 OPA / OpenFGA / 自研）的骨架**（示例文件 `engine_remote.go`）：

```go
// 公司权限中心适配器：转发决策，翻译成 []casbinx.Grant
type remoteGrantEngine struct{ baseURL string }

func (e *remoteGrantEngine) ImplicitGrants(subject string) ([]casbinx.Grant, error) {
    // 1) 调公司权限中心: GET {baseURL}/grants?subject=xxx
    // 2) 把返回的权限点翻译成 casbinx.Grant{Obj, Act}
    //    例如权限点 "pay:gateway:read" → {Obj: "config/pay/gateway/**", Act: "read"}
    return nil, nil
}

func (e *remoteGrantEngine) Enforce(sub, obj, act string) (bool, error) {
    // 调公司权限中心做一次决策: POST {baseURL}/enforce {sub,obj,act}
    return false, nil
}
// 其余方法（策略管理）按需实现，或用"本地只读 + 转发管理到公司控制台"的方式留空返回。
```

注入（`cmd/opshub-iam/main.go`）：

```go
svc := api.NewServiceWithEngine(&remoteGrantEngine{baseURL: "http://perm-center:9000"})
```

### 6.2 用户/身份源：`store.UserStore`（`internal/pkg/store/user.go`）

```go
type UserStore interface {
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, username string) error
    Get(ctx context.Context, username string) (*User, error)
    List(ctx context.Context) ([]*User, error)
}
```

对接 LDAP / 公司用户源时实现该接口（登录可改为查 LDAP 校验密码，`Get` 返回统一 `store.User`），然后：

```go
svc.SetUserStore(&ldapUserStore{...})   // 覆盖默认 MySQL
```

### 6.3 （可选）配置存储 / 审计存储替换

| 接口 | 位置 | 注入 |
|---|---|---|
| `ConfigStore`（Get/List/Put/Delete/Ping） | `internal/config_center/api/store.go` | `NewServiceWithEtcd` / `svc.SetConfigStore(...)` |
| `AuditStore`（Put/ListSub） | 同上 | `svc.SetAuditKV` / `svc.SetAuditStore(...)` |

对接 Nacos / MySQL / 公司变更记录系统时实现对应接口即可，业务逻辑零改动。

### 6.4 关键约束

- **对外协议是契约**：只要 `/authorize`、`/login`、scope 签名协议不变，网关、配置中心、前端、下游**全部零改动**。
- 公司权限中心替换后，建议**保留一个 `config/**` 的管理员角色**映射，保证控制台能完成初始化。

---

## 7. 密钥与安全约定

| 密钥 | 用途 | 现状 | 生产要求 |
|---|---|---|---|
| `decisionSecret`（`authutil.DefaultDecisionSecret`） | scope 决策签名：iam 签发 / config-center 验签 | 默认值写死 | **必须改为可配置**且 iam 与 config-center 使用同一把（k8s Secret 挂载） |
| JWT `CustomSecret`（`pkg/jwt`） | 登录令牌签名 | 默认值 | 生产改为配置/Secret 注入 |

其他约定：
- 网关到 iam/config-center 是**内网转发**；若跨网络部署，建议 mTLS / VPN 或在网关层做访问控制。
- 登出黑名单依赖 Redis（可选）；未配置时登出仅靠令牌自然过期。
- 首个注册用户自动成为管理员（本地/教学便捷）；生产建议改为 bootstrap 配置：
  `auth.bootstrap_admin_username` + `bootstrap_admin_password_cipher`（密文，见 yaml 注释）。

---

## 8. 接入 Checklist

- [ ] 端口与网络：对外只暴露 8001；8101/8201 仅内网可达
- [ ] 依赖就绪：MySQL / etcd / Redis（compose 一键起，见 README「快速开始」）
- [ ] `decisionSecret` 统一配置到 iam 与 config-center（生产必做）
- [ ] 初始化：`/signup` 注册首个管理员，或配置 bootstrap admin
- [ ] 为下游业务服务创建专用账号并授权对应 `business/module` 的 `read`
- [ ] 验证链路：login → scope → tree → pull（见 §4.4）
- [ ] （公司替换场景）实现并注入 `GrantEngine` / `UserStore`，回归上述验证

---

## 9. FAQ

**Q1: 用户权限改了，为什么已登录用户的 scope 还是旧的？**
scope 决策是登录/请求时实时签发的；普通用户读权限在 IAM `/authorize` 时实时取 `ImplicitGrants`，因此权限变更在下次请求即生效（除非命中网关/配置中心的缓存）。令牌本身（JWT）只标识身份，不携带授权列表。

**Q2: 多副本部署时 `decisionSecret` 不一致会怎样？**
配置中心验签失败 → 所有配置请求 403 `invalid authorization signature`。务必用共享 Secret。

**Q3: 401 / 403 / 404 怎么区分？**
401=没登录或令牌无效；403=登录了但**写/删无权限**或操作管理接口非管理员；404=读无权限（不暴露资源存在）或资源真不存在。

**Q4: 公司不想用 casbin，能完全去掉吗？**
能。casbin 只在 `casbinGrantEngine` 内部，`GrantEngine` 换实现即可；策略管理接口（`/policies/*`）在远程引擎里转写或返回空即可，配置鉴权链路不受影响。

**Q5: 配置怎么下发到业务服务？**
当前是下游主动 `pull`（轮询 + diff）。后续可扩展：配置中心基于 etcd Watch 提供 SSE/WebSocket 订阅，或由 K8s Controller 同步为 ConfigMap（见 ARCHITECTURE.md §9）。
