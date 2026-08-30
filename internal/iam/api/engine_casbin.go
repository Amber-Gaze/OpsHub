package api

import (
	"strings"

	"github.com/casbin/casbin/v2"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"
)

// casbinGrantEngine 是 GrantEngine 的默认实现：基于 casbin（策略存 MySQL，经 gorm-adapter 同步）。
// enf 为 nil 时退化为「仅管理员可通过配置鉴权」（与历史行为一致）。
type casbinGrantEngine struct {
	enf *casbin.SyncedEnforcer
}

// ImplicitGrants 由 casbin 策略（含角色继承）推导出 config/ 对象授权列表，并去重。
func (e *casbinGrantEngine) ImplicitGrants(subject string) ([]casbinx.Grant, error) {
	if e.enf == nil {
		return nil, nil
	}
	perms, err := e.enf.GetImplicitPermissionsForUser(subject)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	grants := make([]casbinx.Grant, 0, len(perms))
	for _, p := range perms {
		if len(p) < 3 || !strings.HasPrefix(p[1], "config/") {
			continue
		}
		key := p[1] + "|" + p[2]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		grants = append(grants, casbinx.Grant{Obj: p[1], Act: p[2]})
	}
	return grants, nil
}

func (e *casbinGrantEngine) Enforce(sub, obj, act string) (bool, error) {
	if e.enf == nil {
		return false, nil
	}
	return e.enf.Enforce(sub, obj, act)
}

func (e *casbinGrantEngine) ListPolicies() ([][]string, error) {
	if e.enf == nil {
		return nil, ErrCasbinDisabled
	}
	return e.enf.GetPolicy()
}

func (e *casbinGrantEngine) ListGroupingPolicies() ([][]string, error) {
	if e.enf == nil {
		return nil, ErrCasbinDisabled
	}
	return e.enf.GetGroupingPolicy()
}

func (e *casbinGrantEngine) AddPolicy(sub, obj, act string) (bool, error) {
	if e.enf == nil {
		return false, ErrCasbinDisabled
	}
	return e.enf.AddPolicy(sub, obj, act)
}

func (e *casbinGrantEngine) RemovePolicy(sub, obj, act string) (bool, error) {
	if e.enf == nil {
		return false, ErrCasbinDisabled
	}
	return e.enf.RemovePolicy(sub, obj, act)
}

func (e *casbinGrantEngine) AddRoleForUser(user, role string) (bool, error) {
	if e.enf == nil {
		return false, ErrCasbinDisabled
	}
	return e.enf.AddGroupingPolicy(user, role)
}

func (e *casbinGrantEngine) RemoveRoleForUser(user, role string) (bool, error) {
	if e.enf == nil {
		return false, ErrCasbinDisabled
	}
	return e.enf.RemoveGroupingPolicy(user, role)
}
