package casbinx

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/casbin/casbin/v2/util"
)

// Grant 一条配置授权：对象(Obj) + 动作(Act)。
// Obj 规范（与 NormalizeConfigObject 的产物一致，支持 doublestar 通配）：
//   - config/**                         全部配置
//   - config/pay/**                     整个业务（pay）
//   - config/pay/gateway/**             某业务下某模块
//   - config/pay/gateway/timeout_ms     某个具体配置项
//   - config/default/debug              单段 key（业务级配置）
//
// Act 取值：read | write | delete | grant | *（* 表示全部动作）。
type Grant struct {
	Obj string `json:"obj"`
	Act string `json:"act"`
}

// NormalizeConfigObject 将逻辑配置 key 归一化为 casbin 对象串。
// 单段 key（如 debug）→ config/default/debug；多段（如 pay/gateway/timeout）→ config/pay/gateway/timeout。
func NormalizeConfigObject(key string) string {
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key == "" {
		return "config/*"
	}
	if !strings.Contains(key, "/") {
		return "config/default/" + key
	}
	return "config/" + key
}

// GrantsMatch 判断 grants 中是否存在能覆盖 (obj, act) 的授权。
// obj 应为 NormalizeConfigObject 的产物；匹配语义与 casbin 模型中的 globMatch 一致。
func GrantsMatch(grants []Grant, obj, act string) bool {
	for _, g := range grants {
		if g.Act != "*" && g.Act != act {
			continue
		}
		if ok, err := util.GlobMatch(obj, g.Obj); err == nil && ok {
			return true
		}
	}
	return false
}

// CanRead 判断是否有权读取某个配置 key（obj 覆盖 + read/* 动作）。
func CanRead(grants []Grant, key string) bool {
	return GrantsMatch(grants, NormalizeConfigObject(key), "read")
}

// CanWrite 判断是否有权写入（修改）某个配置 key。
func CanWrite(grants []Grant, key string) bool {
	return GrantsMatch(grants, NormalizeConfigObject(key), "write")
}

// CanDelete 判断是否有权删除某个配置 key。
func CanDelete(grants []Grant, key string) bool {
	return GrantsMatch(grants, NormalizeConfigObject(key), "delete")
}

// AdminGrant 管理员可读可写可删全部配置。
var AdminGrant = []Grant{{Obj: "config/**", Act: "*"}}

// EncodeGrants 将授权列表编码为可放入 HTTP 头的 base64(JSON)。
func EncodeGrants(grants []Grant) string {
	if len(grants) == 0 {
		return ""
	}
	b, _ := json.Marshal(grants)
	return base64.StdEncoding.EncodeToString(b)
}

// DecodeGrants 解析 EncodeGrants 的产物；解析失败返回 nil。
func DecodeGrants(encoded string) []Grant {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil
	}
	var grants []Grant
	if err := json.Unmarshal(b, &grants); err != nil {
		return nil
	}
	return grants
}

// BuildConfigObj 由业务/模块/项生成授权对象串。
// item 非空 → 精确项；module 非空 → config/<b>/<m>/**；否则 → config/<b>/**。
func BuildConfigObj(business, module, item string) string {
	business = strings.Trim(strings.TrimSpace(business), "/")
	if business == "" {
		return "config/**"
	}
	if item != "" {
		return "config/" + business + "/" + module + "/" + item
	}
	if module != "" {
		return "config/" + business + "/" + module + "/**"
	}
	return "config/" + business + "/**"
}

// scopePayloadPrefix 标识 scope 签名载体格式：scope|<subject>|<grantsJSON>|<unix>。
const scopePayloadPrefix = "scope"

// BuildScopePayload 构建 scope 签名载体并返回 payload 与内嵌的 grantsJSON。
// IAM 签发该 payload 并以 HMAC 签名；配置中心收到后验签并解析出 subject+grants。
func BuildScopePayload(subject string, grants []Grant) (payload, grantsJSON string) {
	grantsJSON = string(mustJSON(grants))
	payload = fmt.Sprintf("%s|%s|%s|%d", scopePayloadPrefix, subject, grantsJSON, time.Now().UTC().Unix())
	return payload, grantsJSON
}

// ParseScopePayload 从 payload 中还原 (subject, grants)。
func ParseScopePayload(payload string) (subject string, grants []Grant, err error) {
	parts := strings.SplitN(payload, "|", 4)
	if len(parts) != 4 || parts[0] != scopePayloadPrefix {
		return "", nil, errors.New("invalid scope payload")
	}
	var g []Grant
	if err := json.Unmarshal([]byte(parts[2]), &g); err != nil {
		return "", nil, err
	}
	return parts[1], g, nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
