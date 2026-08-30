package api

// 策略管理接口：全部委托给授权引擎（GrantEngine）。
// 底层为 casbin 时行为与历史一致；对接外部权限中心时由适配层转写。

// ListPolicies 返回所有 p 策略（仅展示用）。
func (s *Service) ListPolicies() [][]string {
	if s.engine == nil {
		return nil
	}
	pol, err := s.engine.ListPolicies()
	if err != nil {
		return nil
	}
	return pol
}

// ListGroupingPolicies 返回所有 g（用户-角色）关系。
func (s *Service) ListGroupingPolicies() [][]string {
	if s.engine == nil {
		return nil
	}
	pol, err := s.engine.ListGroupingPolicies()
	if err != nil {
		return nil
	}
	return pol
}

// AddPolicy 添加 p 策略：sub 可为用户名或角色名；obj 支持通配如 config/pay/**；act 为 read|write|delete|grant|*。
func (s *Service) AddPolicy(sub, obj, act string) (bool, error) {
	if s.engine == nil {
		return false, ErrCasbinDisabled
	}
	return s.engine.AddPolicy(sub, obj, act)
}

// RemovePolicy 删除一条 p 策略。
func (s *Service) RemovePolicy(sub, obj, act string) (bool, error) {
	if s.engine == nil {
		return false, ErrCasbinDisabled
	}
	return s.engine.RemovePolicy(sub, obj, act)
}

// AddRoleForUser 将用户加入角色（g 规则）。
func (s *Service) AddRoleForUser(user, role string) (bool, error) {
	if s.engine == nil {
		return false, ErrCasbinDisabled
	}
	return s.engine.AddRoleForUser(user, role)
}

// RemoveRoleForUser 移除用户与角色绑定。
func (s *Service) RemoveRoleForUser(user, role string) (bool, error) {
	if s.engine == nil {
		return false, ErrCasbinDisabled
	}
	return s.engine.RemoveRoleForUser(user, role)
}

// EnforceConfig 对配置资源做授权校验（不含 IsAdmin 短路）。
func (s *Service) EnforceConfig(sub, obj, act string) (bool, error) {
	if s.engine == nil {
		return false, nil
	}
	return s.engine.Enforce(sub, obj, act)
}
