package api

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
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
}

func NewService() *Service {
	return &Service{
		configs: make(map[string]ConfigItem),
	}
}

func (s *Service) List() []ConfigItem {
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg, ok := s.configs[key]
	return cfg, ok
}

func (s *Service) Create(key, value, operator string) (ConfigItem, error) {
	if key == "" {
		return ConfigItem{}, errors.New("key is required")
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
