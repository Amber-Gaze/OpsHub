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

	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/etcd"
)

var (
	ErrConfigExists   = errors.New("config already exists")
	ErrConfigNotFound = errors.New("config not found")
	defaultUpdatedBy  = "system"
)

type ConfigItem struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

type Service struct {
	mu      sync.RWMutex
	configs map[string]ConfigItem
	etcdKV  *etcd.ConfigKV
}

func NewService() *Service {
	return &Service{
		configs: make(map[string]ConfigItem),
	}
}

// NewServiceWithEtcd 使用 etcd 持久化配置；未配置 endpoints 时请使用 NewService()。
func NewServiceWithEtcd(kv *etcd.ConfigKV) *Service {
	return &Service{etcdKV: kv}
}

func (s *Service) etcdCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (s *Service) List() []ConfigItem {
	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		raw, err := s.etcdKV.List(ctx)
		if err != nil {
			return nil
		}
		items := make([]ConfigItem, 0, len(raw))
		for _, b := range raw {
			var cfg ConfigItem
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

	items := make([]ConfigItem, 0, len(s.configs))
	for _, cfg := range s.configs {
		items = append(items, cfg)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
	return items
}

func (s *Service) Get(key string) (ConfigItem, bool) {
	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		b, err := s.etcdKV.Get(ctx, key)
		if err != nil || len(b) == 0 {
			return ConfigItem{}, false
		}
		var cfg ConfigItem
		if json.Unmarshal(b, &cfg) != nil {
			return ConfigItem{}, false
		}
		return cfg, true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, ok := s.configs[key]
	return cfg, ok
}

func (s *Service) Create(key, value, operator string) (ConfigItem, error) {
	if key == "" {
		return ConfigItem{}, errors.New("key is required")
	}

	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		b, err := s.etcdKV.Get(ctx, key)
		if err != nil {
			return ConfigItem{}, fmt.Errorf("etcd get: %w", err)
		}
		if len(b) > 0 {
			return ConfigItem{}, ErrConfigExists
		}
		cfg := newConfigItem(key, value, operator, 1)
		payload, err := json.Marshal(cfg)
		if err != nil {
			return ConfigItem{}, err
		}
		if err := s.etcdKV.Put(ctx, key, payload); err != nil {
			return ConfigItem{}, fmt.Errorf("etcd put: %w", err)
		}
		return cfg, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configs[key]; exists {
		return ConfigItem{}, ErrConfigExists
	}

	cfg := newConfigItem(key, value, operator, 1)
	s.configs[key] = cfg
	return cfg, nil
}

func (s *Service) Update(key, value, operator string) (ConfigItem, error) {
	if key == "" {
		return ConfigItem{}, errors.New("key is required")
	}

	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		b, err := s.etcdKV.Get(ctx, key)
		if err != nil {
			return ConfigItem{}, fmt.Errorf("etcd get: %w", err)
		}
		if len(b) == 0 {
			return ConfigItem{}, ErrConfigNotFound
		}
		var cfg ConfigItem
		if err := json.Unmarshal(b, &cfg); err != nil {
			return ConfigItem{}, err
		}
		cfg.Value = value
		cfg.Version++
		cfg.UpdatedAt = time.Now().UTC()
		cfg.UpdatedBy = fallbackOperator(operator)
		payload, err := json.Marshal(cfg)
		if err != nil {
			return ConfigItem{}, err
		}
		if err := s.etcdKV.Put(ctx, key, payload); err != nil {
			return ConfigItem{}, fmt.Errorf("etcd put: %w", err)
		}
		return cfg, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, exists := s.configs[key]
	if !exists {
		return ConfigItem{}, ErrConfigNotFound
	}

	cfg.Value = value
	cfg.Version++
	cfg.UpdatedAt = time.Now().UTC()
	cfg.UpdatedBy = fallbackOperator(operator)
	s.configs[key] = cfg
	return cfg, nil
}

func (s *Service) Delete(key string) error {
	if key == "" {
		return errors.New("key is required")
	}

	if s.etcdKV != nil {
		ctx, cancel := s.etcdCtx()
		defer cancel()
		ok, err := s.etcdKV.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf("etcd delete: %w", err)
		}
		if !ok {
			return ErrConfigNotFound
		}
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configs[key]; !exists {
		return ErrConfigNotFound
	}
	delete(s.configs, key)
	return nil
}

func newConfigItem(key, value, operator string, version int) ConfigItem {
	return ConfigItem{
		Key:       key,
		Value:     value,
		Version:   version,
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: fallbackOperator(operator),
	}
}

func fallbackOperator(operator string) string {
	if strings.TrimSpace(operator) == "" {
		return defaultUpdatedBy
	}
	return operator
}
