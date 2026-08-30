package api

import "github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"

// GrantEngine 是「配置授权决策」的抽象（鉴权适配点）。
//
// 现状默认实现为 casbin（casbinGrantEngine，策略存 MySQL，经 gorm-adapter 同步）。
// 公司如已有自研/第三方权限中心（OPA、OpenFGA、authzed 等），只需实现本接口注入即可，
// 无需改动 Service / Handler / 对外协议（/authorize、scope 签名）与下游（gateway、config-center）。
// 典型做法：engine_remote.go 里转发到公司权限服务，把返回的权限点翻译成 []casbinx.Grant。
type GrantEngine interface {
	// ImplicitGrants 返回用户经角色继承后的全部配置授权（仅 config/ 对象），用于签发 scope。
	ImplicitGrants(subject string) ([]casbinx.Grant, error)
	// Enforce 精确校验 sub 对 obj 的 act 权限（写/删操作的最终裁决）。
	Enforce(sub, obj, act string) (bool, error)

	// 策略/角色管理（管理端 UI 使用；对接外部权限中心时可由适配层转写）。
	ListPolicies() ([][]string, error)
	ListGroupingPolicies() ([][]string, error)
	AddPolicy(sub, obj, act string) (bool, error)
	RemovePolicy(sub, obj, act string) (bool, error)
	AddRoleForUser(user, role string) (bool, error)
	RemoveRoleForUser(user, role string) (bool, error)
}
