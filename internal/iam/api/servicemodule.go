package api

import (
	"context"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
)

// ListServiceModules 返回某服务账号订阅的模块。
func (s *Service) ListServiceModules(ctx context.Context, username string) ([]*store.ServiceModule, error) {
	return s.serviceModuleStore().List(ctx, username)
}

// SetServiceModules 管理员覆盖式注册服务的模块订阅：
// 先撤销旧订阅（删订阅表 + 撤 read 授权），再对每个模块授 read 并写订阅表。
// 「注册即授读」，默认不授写/删，满足下游服务只读拉取。
func (s *Service) SetServiceModules(ctx context.Context, username string, paths []string) ([]*store.ServiceModule, error) {
	// 服务账号必须存在
	if _, err := s.userStore().Get(ctx, username); err != nil {
		return nil, ErrInvalidUser
	}

	// 撤销旧的订阅与授权
	old, err := s.serviceModuleStore().List(ctx, username)
	if err != nil {
		return nil, err
	}
	for _, o := range old {
		if obj, ok := modulePathToObj(o.Path); ok {
			_, _ = s.RemovePolicy(username, obj, "read")
		}
		_ = s.serviceModuleStore().DeleteByPath(ctx, username, o.Path)
	}

	// 逐个注册新模块：授 read + 写订阅表
	now := time.Now().Unix()
	out := make([]*store.ServiceModule, 0, len(paths))
	for _, p := range paths {
		p = normalizeModulePath(p)
		if p == "" {
			continue
		}
		obj, ok := modulePathToObj(p)
		if !ok {
			continue
		}
		if ok, err := s.AddPolicy(username, obj, "read"); err != nil || !ok {
			continue
		}
		b, m := splitModulePath(p)
		sm := &store.ServiceModule{
			Username:  username,
			Business:  b,
			Module:    m,
			Path:      p,
			CreatedAt: now,
		}
		if err := s.serviceModuleStore().Create(ctx, sm); err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, nil
}

// RemoveServiceModule 取消单个模块订阅（撤 read 授权 + 删订阅记录）。
func (s *Service) RemoveServiceModule(ctx context.Context, username, path string) error {
	p := normalizeModulePath(path)
	if p == "" {
		return nil
	}
	if obj, ok := modulePathToObj(p); ok {
		_, _ = s.RemovePolicy(username, obj, "read")
	}
	return s.serviceModuleStore().DeleteByPath(ctx, username, p)
}

// serviceModuleStore 返回订阅存储（默认全局 MySQL）。
func (s *Service) serviceModuleStore() store.ServiceModuleStore {
	if s.svcMods != nil {
		return s.svcMods
	}
	return store.Client().ServiceModules()
}

// SetServiceModuleStore 注入订阅存储（默认全局 MySQL）。
func (s *Service) SetServiceModuleStore(sms store.ServiceModuleStore) *Service {
	s.svcMods = sms
	return s
}

// normalizeModulePath 归一化模块路径：去首尾斜杠、过滤空段，最多保留 2 段（business 或 business/module）。
func normalizeModulePath(p string) string {
	p = strings.Trim(strings.TrimSpace(p), "/")
	if p == "" {
		return ""
	}
	clean := make([]string, 0, 2)
	for _, s := range strings.Split(p, "/") {
		s = strings.TrimSpace(s)
		if s != "" {
			clean = append(clean, s)
		}
		if len(clean) == 2 {
			break
		}
	}
	return strings.Join(clean, "/")
}

// splitModulePath 拆分模块路径为 (business, module)；单段视为整业务（module=""）。
func splitModulePath(p string) (business, module string) {
	segs := strings.Split(p, "/")
	business = segs[0]
	if len(segs) > 1 {
		module = segs[1]
	}
	return
}

// modulePathToObj 将模块路径转为 casbin 对象（config/business/module/** 或 config/business/**）。
func modulePathToObj(p string) (string, bool) {
	b, m := splitModulePath(p)
	if b == "" {
		return "", false
	}
	return casbinx.BuildConfigObj(b, m, ""), true
}
