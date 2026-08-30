package api

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/etcd"
)

// revisionKey 全局配置版本号的保留 key。配置 key 为 business/module/name，
// 不会以 "@" 开头，因此与合法配置不冲突（List 时会过滤掉保留 key）。
const revisionKey = "@revision"

// ConfigStore 是「配置主存储」的抽象（适配点）。
//
// 现状默认实现为 etcd（etcdConfigStore，生产/多实例）与进程内存（memoryConfigStore，单实例/测试/教学）。
// 后续对接公司配置存储（Nacos / Apollo / MySQL 等）时，实现本接口并注入即可，业务逻辑（Service）零改动。
type ConfigStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	List(ctx context.Context) (map[string][]byte, error)
	Put(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) (bool, error)
	Ping(ctx context.Context) error
	// Revision 返回全局配置版本号（单调递增，用于下游增量拉取判断更新）。
	Revision(ctx context.Context) (int64, error)
	// BumpRevision 原子递增全局版本号并返回新值。
	BumpRevision(ctx context.Context) (int64, error)
}

// AuditStore 是「配置变更审计」的抽象（适配点）：只需追加式写入 + 按 key 前缀查询（历史/差异）。
// 现状默认实现为 etcd sibling（etcdAuditStore，跨实例共享）；未配置时为进程内存审计。
type AuditStore interface {
	Put(ctx context.Context, key string, value []byte) error
	ListSub(ctx context.Context, subPrefix string) (map[string][]byte, error)
}

// ---- etcd 实现 ----

type etcdConfigStore struct {
	kv *etcd.ConfigKV
}

func (s *etcdConfigStore) Get(ctx context.Context, key string) ([]byte, error) {
	return s.kv.Get(ctx, key)
}

func (s *etcdConfigStore) List(ctx context.Context) (map[string][]byte, error) {
	raw, err := s.kv.List(ctx)
	if err != nil {
		return nil, err
	}
	return filterReservedKeys(raw), nil
}

func (s *etcdConfigStore) Put(ctx context.Context, key string, value []byte) error {
	return s.kv.Put(ctx, key, value)
}

func (s *etcdConfigStore) Delete(ctx context.Context, key string) (bool, error) {
	return s.kv.Delete(ctx, key)
}

func (s *etcdConfigStore) Ping(ctx context.Context) error {
	return s.kv.Ping(ctx)
}

func (s *etcdConfigStore) Revision(ctx context.Context) (int64, error) {
	b, err := s.kv.Get(ctx, revisionKey)
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return 0, nil
	}
	return strconv.ParseInt(string(b), 10, 64)
}

func (s *etcdConfigStore) BumpRevision(ctx context.Context) (int64, error) {
	return s.kv.Incr(ctx, revisionKey)
}

type etcdAuditStore struct {
	kv *etcd.ConfigKV
}

func (s *etcdAuditStore) Put(ctx context.Context, key string, value []byte) error {
	return s.kv.Put(ctx, key, value)
}

func (s *etcdAuditStore) ListSub(ctx context.Context, subPrefix string) (map[string][]byte, error) {
	return s.kv.ListSub(ctx, subPrefix)
}

// ---- 内存实现（单实例/测试/教学；进程重启即失） ----

type memoryConfigStore struct {
	mu  sync.RWMutex
	m   map[string][]byte
	rev atomic.Int64
}

func newMemoryConfigStore() *memoryConfigStore {
	return &memoryConfigStore{m: make(map[string][]byte)}
}

func filterReservedKeys(m map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(m))
	for k, v := range m {
		if strings.HasPrefix(k, "@") {
			continue // 保留 key（如全局版本号）不视为配置项
		}
		out[k] = v
	}
	return out
}

func (s *memoryConfigStore) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if b, ok := s.m[key]; ok {
		return append([]byte(nil), b...), nil
	}
	return nil, nil
}

func (s *memoryConfigStore) List(ctx context.Context) (map[string][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return filterReservedKeys(s.m), nil
}

func (s *memoryConfigStore) Put(ctx context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = append([]byte(nil), value...)
	return nil
}

func (s *memoryConfigStore) Delete(ctx context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[key]; !ok {
		return false, nil
	}
	delete(s.m, key)
	return true, nil
}

func (s *memoryConfigStore) Ping(ctx context.Context) error {
	return nil
}

func (s *memoryConfigStore) Revision(ctx context.Context) (int64, error) {
	return s.rev.Load(), nil
}

func (s *memoryConfigStore) BumpRevision(ctx context.Context) (int64, error) {
	return s.rev.Add(1), nil
}
