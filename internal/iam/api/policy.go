package api

import (
	"github.com/casbin/casbin/v2"
)

// ListPolicies 返回所有 p 策略（仅展示用）。
func (s *Service) ListPolicies() [][]string {
	if s.enf == nil {
		return nil
	}
	return s.enf.GetPolicy()
}

// ListGroupingPolicies 返回所有 g（用户-角色）关系。
func (s *Service) ListGroupingPolicies() [][]string {
	if s.enf == nil {
		return nil
	}
	return s.enf.GetGroupingPolicy()
}

// AddPolicy 添加 p 策略：sub 可为用户名或角色名；obj 支持通配如 config:pay:*；act 为 read|write|delete|grant|*。
func (s *Service) AddPolicy(sub, obj, act string) (bool, error) {
	if s.enf == nil {
		return false, ErrCasbinDisabled
	}
	return s.enf.AddPolicy(sub, obj, act)
}

// RemovePolicy 删除一条 p 策略。
func (s *Service) RemovePolicy(sub, obj, act string) (bool, error) {
	if s.enf == nil {
		return false, ErrCasbinDisabled
	}
	return s.enf.RemovePolicy(sub, obj, act)
}

// AddRoleForUser 将用户加入角色（g 规则）。
func (s *Service) AddRoleForUser(user, role string) (bool, error) {
	if s.enf == nil {
		return false, ErrCasbinDisabled
	}
	return s.enf.AddGroupingPolicy(user, role)
}

// RemoveRoleForUser 移除用户与角色绑定。
func (s *Service) RemoveRoleForUser(user, role string) (bool, error) {
	if s.enf == nil {
		return false, ErrCasbinDisabled
	}
	return s.enf.RemoveGroupingPolicy(user, role)
}

// EnforceConfig 对配置资源做 Casbin 校验（不含 IsAdmin 短路）。
func (s *Service) EnforceConfig(sub, obj, act string) (bool, error) {
	if s.enf == nil {
		return false, nil
	}
	return s.enf.Enforce(sub, obj, act)
}

// Enforcer 返回底层执行器（测试或高级用法）。
func (s *Service) Enforcer() *casbin.SyncedEnforcer {
	return s.enf
}
