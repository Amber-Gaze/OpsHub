package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/config_center/domain"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/etcd"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/redis"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
)

var (
	ErrConfigExists   = errors.New("config already exists")
	ErrConfigNotFound = errors.New("config not found")
	defaultUpdatedBy  = "system"

	// Redis 缓存 key 前缀：etcd 为主存储，Redis 作为 L1 读缓存（cache-aside）。
	cacheKeyList = "opshub:config:list"
)

func cacheKeyItem(key string) string {
	return "opshub:config:item:" + key
}

type Service struct {
	// store 配置主存储（可替换实现：etcd / 内存 / Nacos 等，见 ConfigStore）
	store ConfigStore
	// audit 审计存储（可替换实现：etcd sibling / 自定义；nil 表示进程内存审计）
	audit AuditStore
	// mu/history 仅用于进程内存审计模式（未配置 audit 存储时）
	mu      sync.RWMutex
	history map[string][]domain.ConfigChange
	cache   *redis.Cache
}

func NewService() *Service {
	return &Service{
		store:   newMemoryConfigStore(),
		history: make(map[string][]domain.ConfigChange),
	}
}

// SetRedisCache 附加 Redis 读缓存（可选）。启用后读写走 cache-aside：
// 读先查缓存、未命中回源（etcd/内存）并回填；写后主动失效相关缓存。
func (s *Service) SetRedisCache(c *redis.Cache) *Service {
	s.cache = c
	return s
}

// SetAuditKV 附加审计存储（etcd 模式，前缀与配置存储互为 sibling，互不污染）。
// 未设置时审计历史保存在进程内存（仅单实例可见）。
func (s *Service) SetAuditKV(kv *etcd.ConfigKV) *Service {
	s.audit = &etcdAuditStore{kv: kv}
	return s
}

// SetAuditStore 注入自定义审计存储（对接公司变更记录/审计系统时使用）。
func (s *Service) SetAuditStore(a AuditStore) *Service {
	s.audit = a
	return s
}

// SetConfigStore 注入自定义配置主存储（对接公司配置存储如 Nacos/MySQL 时使用）。
func (s *Service) SetConfigStore(st ConfigStore) *Service {
	s.store = st
	return s
}

// recordChange 记录一条配置变更（携带当时的全局版本号 revision，供下游增量拉取）。
// 配置了 audit 存储时持久化（etcd sibling / 自定义实现，跨实例共享）；
// 否则写入进程内存 history（单实例可见；本方法自行加锁，调用方无需持锁）。
func (s *Service) recordChange(ch domain.ConfigChange, revision int64) {
	ch.Revision = revision
	if s.audit != nil {
		payload, err := json.Marshal(ch)
		if err != nil {
			return
		}
		// 用创建时间纳秒做追加键，保证同一 key 的多次变更不冲突
		auditKey := fmt.Sprintf("%s/%d", ch.Key, ch.CreatedAt.UnixNano())
		ctx, cancel := s.etcdCtx()
		defer cancel()
		_ = s.audit.Put(ctx, auditKey, payload)
		return
	}
	s.mu.Lock()
	s.history[ch.Key] = append(s.history[ch.Key], ch)
	s.mu.Unlock()
}

// bumpRevision 递增全局配置版本号并返回新值；失败时记录日志并返回 0（写操作本身不因此失败）。
func (s *Service) bumpRevision(ctx context.Context) int64 {
	if s.store == nil {
		return 0
	}
	rev, err := s.store.BumpRevision(ctx)
	if err != nil {
		logger.Errorf("config-center: bump revision: %v", err)
		return 0
	}
	return rev
}

// Revision 返回当前全局配置版本号（供下游增量拉取判断更新）。
func (s *Service) Revision(ctx context.Context) int64 {
	if s.store == nil {
		return 0
	}
	rev, err := s.store.Revision(ctx)
	if err != nil {
		return 0
	}
	return rev
}

// allChanges 返回全部配置变更记录（etcd audit 或进程内存），按时间升序。
func (s *Service) allChanges() []domain.ConfigChange {
	if s.audit != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		raw, err := s.audit.ListSub(ctx, "")
		if err != nil {
			return nil
		}
		changes := make([]domain.ConfigChange, 0, len(raw))
		for _, b := range raw {
			var ch domain.ConfigChange
			if json.Unmarshal(b, &ch) != nil {
				continue
			}
			changes = append(changes, ch)
		}
		sort.Slice(changes, func(i, j int) bool { return changes[i].CreatedAt.Before(changes[j].CreatedAt) })
		return changes
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	changes := make([]domain.ConfigChange, 0, len(s.history))
	for _, list := range s.history {
		changes = append(changes, list...)
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].CreatedAt.Before(changes[j].CreatedAt) })
	return changes
}

// ChangesSince 返回全局版本号 > since 的各 key 最后一次变更（用于下游增量拉取）。
// 调用方据此得到：变更过的 key（含删除）列表。
func (s *Service) ChangesSince(since int64) map[string]domain.ConfigChange {
	last := map[string]domain.ConfigChange{}
	for _, ch := range s.allChanges() {
		if ch.Revision <= since {
			continue
		}
		if prev, ok := last[ch.Key]; !ok || ch.CreatedAt.After(prev.CreatedAt) {
			last[ch.Key] = ch
		}
	}
	return last
}

// History 返回某个配置 key 的全部变更记录（按时间升序）。
func (s *Service) History(key string) []domain.ConfigChange {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	if s.audit != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		raw, err := s.audit.ListSub(ctx, key+"/")
		if err != nil {
			return nil
		}
		changes := make([]domain.ConfigChange, 0, len(raw))
		for _, b := range raw {
			var ch domain.ConfigChange
			if json.Unmarshal(b, &ch) != nil {
				continue
			}
			changes = append(changes, ch)
		}
		sort.Slice(changes, func(i, j int) bool { return changes[i].CreatedAt.Before(changes[j].CreatedAt) })
		return changes
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	hist := s.history[key]
	out := make([]domain.ConfigChange, len(hist))
	copy(out, hist)
	return out
}

// NewServiceWithEtcd 使用 etcd 持久化配置；未配置 endpoints 时请使用 NewService()。
func NewServiceWithEtcd(kv *etcd.ConfigKV) *Service {
	return &Service{store: &etcdConfigStore{kv: kv}}
}

func (s *Service) etcdCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// Ready 用于 readiness：主存储必须可访问；纯内存模式始终就绪。
func (s *Service) Ready() error {
	if s.store == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.store.Ping(ctx)
}

func (s *Service) List() []domain.ConfigItem {
	// 缓存命中直接返回
	if s.cache != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		if b, ok := s.cache.Get(ctx, cacheKeyList); ok {
			var items []domain.ConfigItem
			if json.Unmarshal(b, &items) == nil {
				return items
			}
		}
	}
	items := s.loadList()
	if s.cache != nil {
		if b, err := json.Marshal(items); err == nil {
			ctx, cancel := s.etcdCtx()
			defer cancel()
			_ = s.cache.Set(ctx, cacheKeyList, b)
		}
	}
	return items
}

// loadList 从主存储读取全部配置并按 key 排序（etcd / 内存 / 自定义实现由 ConfigStore 决定）。
func (s *Service) loadList() []domain.ConfigItem {
	ctx, cancel := s.etcdCtx()
	defer cancel()
	raw, err := s.store.List(ctx)
	if err != nil {
		return nil
	}
	items := make([]domain.ConfigItem, 0, len(raw))
	for _, b := range raw {
		var cfg domain.ConfigItem
		if json.Unmarshal(b, &cfg) != nil {
			continue
		}
		items = append(items, cfg)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	return items
}

func (s *Service) Get(key string) (domain.ConfigItem, bool) {
	// 缓存命中直接返回
	if s.cache != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		if b, ok := s.cache.Get(ctx, cacheKeyItem(key)); ok {
			var cfg domain.ConfigItem
			if json.Unmarshal(b, &cfg) == nil {
				return cfg, true
			}
		}
	}
	cfg, ok := s.loadGet(key)
	if ok && s.cache != nil {
		if b, err := json.Marshal(cfg); err == nil {
			ctx, cancel := s.etcdCtx()
			defer cancel()
			_ = s.cache.Set(ctx, cacheKeyItem(key), b)
		}
	}
	return cfg, ok
}

// loadGet 从主存储读取单个配置。
func (s *Service) loadGet(key string) (domain.ConfigItem, bool) {
	ctx, cancel := s.etcdCtx()
	defer cancel()
	b, err := s.store.Get(ctx, key)
	if err != nil || len(b) == 0 {
		return domain.ConfigItem{}, false
	}
	var cfg domain.ConfigItem
	if json.Unmarshal(b, &cfg) != nil {
		return domain.ConfigItem{}, false
	}
	return cfg, true
}

// invalidateCache 写操作成功后失效受影响的缓存（列表 + 单项）。
func (s *Service) invalidateCache(key string) {
	if s.cache == nil {
		return
	}
	ctx, cancel := s.etcdCtx()
	defer cancel()
	_ = s.cache.Delete(ctx, cacheKeyList, cacheKeyItem(key))
}

func (s *Service) Create(key, value, operator string) (domain.ConfigItem, error) {
	if err := domain.ValidateKey(key); err != nil {
		return domain.ConfigItem{}, err
	}

	ctx, cancel := s.etcdCtx()
	defer cancel()
	b, err := s.store.Get(ctx, key)
	if err != nil {
		return domain.ConfigItem{}, fmt.Errorf("config store get: %w", err)
	}
	if len(b) > 0 {
		return domain.ConfigItem{}, ErrConfigExists
	}

	cfg := newConfigItem(key, value, operator, 1)
	payload, err := json.Marshal(cfg)
	if err != nil {
		return domain.ConfigItem{}, err
	}
	if err := s.store.Put(ctx, key, payload); err != nil {
		return domain.ConfigItem{}, fmt.Errorf("config store put: %w", err)
	}
	s.invalidateCache(key)
	s.recordChange(newChange(key, cfg.Version, "create", "", cfg.Value, operator), s.bumpRevision(ctx))
	return cfg, nil
}

func (s *Service) Update(key, value, operator string) (domain.ConfigItem, error) {
	if err := domain.ValidateKey(key); err != nil {
		return domain.ConfigItem{}, err
	}

	ctx, cancel := s.etcdCtx()
	defer cancel()
	b, err := s.store.Get(ctx, key)
	if err != nil {
		return domain.ConfigItem{}, fmt.Errorf("config store get: %w", err)
	}
	if len(b) == 0 {
		return domain.ConfigItem{}, ErrConfigNotFound
	}
	var cfg domain.ConfigItem
	if err := json.Unmarshal(b, &cfg); err != nil {
		return domain.ConfigItem{}, err
	}
	oldValue := cfg.Value
	cfg.Value = value
	cfg.Version++
	cfg.UpdatedAt = time.Now().UTC()
	cfg.UpdatedBy = fallbackOperator(operator)
	payload, err := json.Marshal(cfg)
	if err != nil {
		return domain.ConfigItem{}, err
	}
	if err := s.store.Put(ctx, key, payload); err != nil {
		return domain.ConfigItem{}, fmt.Errorf("config store put: %w", err)
	}
	s.invalidateCache(key)
	s.recordChange(newChange(key, cfg.Version, "update", oldValue, value, operator), s.bumpRevision(ctx))
	return cfg, nil
}

func (s *Service) Delete(key string) error {
	if err := domain.ValidateKey(key); err != nil {
		return err
	}

	ctx, cancel := s.etcdCtx()
	defer cancel()
	// 先读取被删配置的版本/值/操作人，用于审计记录
	deletedVersion, deletedValue, deletedOperator := 0, "", ""
	if b, err := s.store.Get(ctx, key); err == nil && len(b) > 0 {
		var cfg domain.ConfigItem
		if json.Unmarshal(b, &cfg) == nil {
			deletedVersion = cfg.Version
			deletedValue = cfg.Value
			deletedOperator = cfg.UpdatedBy
		}
	}
	ok, err := s.store.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("config store delete: %w", err)
	}
	if !ok {
		return ErrConfigNotFound
	}
	s.invalidateCache(key)
	s.recordChange(newChange(key, deletedVersion+1, "delete", deletedValue, "", deletedOperator), s.bumpRevision(ctx))
	return nil
}

// Tree 返回完整的「业务 → 模块 → 具体项」层级树，供控制台左侧导航与首屏展示。
func (s *Service) Tree() []domain.BusinessNode {
	return domain.BuildTree(s.List())
}

// GetBusiness 返回单个业务下的模块分组（业务子树）；业务不存在时返回 false。
func (s *Service) GetBusiness(business string) ([]domain.ModuleNode, bool) {
	business = strings.TrimSpace(business)
	if business == "" {
		return nil, false
	}
	return domain.GroupByBusiness(s.List(), business)
}

// GetModule 返回 business/module 下的全部配置项；模块不存在时返回 false。
func (s *Service) GetModule(business, module string) ([]domain.ConfigItem, bool) {
	business = strings.TrimSpace(business)
	module = strings.TrimSpace(module)
	if business == "" || module == "" {
		return nil, false
	}
	var items []domain.ConfigItem
	for _, it := range s.List() {
		b, m := domain.SplitKey(it.Key)
		if b == business && m == module {
			items = append(items, it)
		}
	}
	if len(items) == 0 {
		return nil, false
	}
	return items, true
}

// GetItem 返回 business/module/name 对应的具体配置项。
func (s *Service) GetItem(business, module, name string) (domain.ConfigItem, bool) {
	business = strings.TrimSpace(business)
	module = strings.TrimSpace(module)
	name = strings.TrimSpace(name)
	if business == "" || module == "" || name == "" {
		return domain.ConfigItem{}, false
	}
	return s.Get(strings.Join([]string{business, module, name}, "/"))
}

func newConfigItem(key, value, operator string, version int) domain.ConfigItem {
	return domain.ConfigItem{
		Key:       key,
		Value:     value,
		Version:   version,
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: fallbackOperator(operator),
	}
}

func newChange(key string, version int, action, before, after, operator string) domain.ConfigChange {
	return domain.ConfigChange{
		Key:       key,
		Version:   version,
		Action:    action,
		Before:    before,
		After:     after,
		Operator:  fallbackOperator(operator),
		CreatedAt: time.Now().UTC(),
	}
}

func fallbackOperator(operator string) string {
	if strings.TrimSpace(operator) == "" {
		return defaultUpdatedBy
	}
	return operator
}
