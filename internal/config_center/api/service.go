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
	mu      sync.RWMutex
	configs map[string]domain.ConfigItem
	history map[string][]domain.ConfigChange
	etcdKV  *etcd.ConfigKV
	auditKV *etcd.ConfigKV // 审计历史（etcd 模式，sibling prefix，多实例共享）
	cache   *redis.Cache
}

func NewService() *Service {
	return &Service{
		configs: make(map[string]domain.ConfigItem),
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
	s.auditKV = kv
	return s
}

// recordChange 记录一条配置变更。etcd 模式下持久化到 auditKV（跨实例共享）；
// 内存模式下直接追加到进程内 history —— 调用方（Create/Update/Delete 的内存分支）
// 必须已持有 s.mu，避免与 Create 等持有的锁重入死锁（sync.RWMutex 不可重入）。
func (s *Service) recordChange(ch domain.ConfigChange) {
	if s.auditKV != nil {
		payload, err := json.Marshal(ch)
		if err != nil {
			return
		}
		// 用创建时间纳秒做追加键，保证同一 key 的多次变更不冲突
		auditKey := fmt.Sprintf("%s/%d", ch.Key, ch.CreatedAt.UnixNano())
		ctx, cancel := s.etcdCtx()
		defer cancel()
		_ = s.auditKV.Put(ctx, auditKey, payload)
		return
	}
	// 内存模式：调用方已持有 s.mu，直接追加（不加锁）
	s.history[ch.Key] = append(s.history[ch.Key], ch)
}

// History 返回某个配置 key 的全部变更记录（按时间升序）。
func (s *Service) History(key string) []domain.ConfigChange {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	if s.auditKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		raw, err := s.auditKV.ListSub(ctx, key+"/")
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
	return &Service{etcdKV: kv}
}

func (s *Service) etcdCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// Ready 用于 readiness：启用 etcd 时必须能连上集群；纯内存模式始终就绪。
func (s *Service) Ready() error {
	if s.etcdKV == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.etcdKV.Ping(ctx)
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

// loadList 从主存储（etcd 或内存）读取全部配置并按 key 排序。
func (s *Service) loadList() []domain.ConfigItem {
	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		raw, err := s.etcdKV.List(ctx)
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

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.ConfigItem, 0, len(s.configs))
	for _, cfg := range s.configs {
		items = append(items, cfg)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
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
	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		b, err := s.etcdKV.Get(ctx, key)
		if err != nil || len(b) == 0 {
			return domain.ConfigItem{}, false
		}
		var cfg domain.ConfigItem
		if json.Unmarshal(b, &cfg) != nil {
			return domain.ConfigItem{}, false
		}
		return cfg, true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, ok := s.configs[key]
	return cfg, ok
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

	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		b, err := s.etcdKV.Get(ctx, key)
		if err != nil {
			return domain.ConfigItem{}, fmt.Errorf("etcd get: %w", err)
		}
		if len(b) > 0 {
			return domain.ConfigItem{}, ErrConfigExists
		}
		cfg := newConfigItem(key, value, operator, 1)
		payload, err := json.Marshal(cfg)
		if err != nil {
			return domain.ConfigItem{}, err
		}
		if err := s.etcdKV.Put(ctx, key, payload); err != nil {
			return domain.ConfigItem{}, fmt.Errorf("etcd put: %w", err)
		}
		s.invalidateCache(key)
		s.recordChange(newChange(key, cfg.Version, "create", "", cfg.Value, operator))
		return cfg, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configs[key]; exists {
		return domain.ConfigItem{}, ErrConfigExists
	}

	cfg := newConfigItem(key, value, operator, 1)
	s.configs[key] = cfg
	s.invalidateCache(key)
	s.recordChange(newChange(key, cfg.Version, "create", "", cfg.Value, operator))
	return cfg, nil
}

func (s *Service) Update(key, value, operator string) (domain.ConfigItem, error) {
	if err := domain.ValidateKey(key); err != nil {
		return domain.ConfigItem{}, err
	}

	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		b, err := s.etcdKV.Get(ctx, key)
		if err != nil {
			return domain.ConfigItem{}, fmt.Errorf("etcd get: %w", err)
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
		if err := s.etcdKV.Put(ctx, key, payload); err != nil {
			return domain.ConfigItem{}, fmt.Errorf("etcd put: %w", err)
		}
		s.invalidateCache(key)
		s.recordChange(newChange(key, cfg.Version, "update", oldValue, value, operator))
		return cfg, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, exists := s.configs[key]
	if !exists {
		return domain.ConfigItem{}, ErrConfigNotFound
	}

	oldValue := cfg.Value
	cfg.Value = value
	cfg.Version++
	cfg.UpdatedAt = time.Now().UTC()
	cfg.UpdatedBy = fallbackOperator(operator)
	s.configs[key] = cfg
	s.invalidateCache(key)
	s.recordChange(newChange(key, cfg.Version, "update", oldValue, value, operator))
	return cfg, nil
}

func (s *Service) Delete(key string) error {
	if err := domain.ValidateKey(key); err != nil {
		return err
	}

	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		// 先读取被删配置的版本与值，用于审计记录
		deletedVersion, deletedValue := 0, ""
		if b, err := s.etcdKV.Get(ctx, key); err == nil && len(b) > 0 {
			var cfg domain.ConfigItem
			if json.Unmarshal(b, &cfg) == nil {
				deletedVersion = cfg.Version
				deletedValue = cfg.Value
			}
		}
		ok, err := s.etcdKV.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf("etcd delete: %w", err)
		}
		if !ok {
			return ErrConfigNotFound
		}
		s.invalidateCache(key)
		s.recordChange(newChange(key, deletedVersion+1, "delete", deletedValue, "", ""))
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, exists := s.configs[key]
	if !exists {
		return ErrConfigNotFound
	}
	delete(s.configs, key)
	s.invalidateCache(key)
	s.recordChange(newChange(key, cfg.Version+1, "delete", cfg.Value, "", cfg.UpdatedBy))
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
