package etcd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ConfigKV 在 etcd 中以 prefix + logicalKey 存 JSON 配置值。
type ConfigKV struct {
	cli    *clientv3.Client
	prefix string
}

func NewConfigKV(endpoints []string, prefix string) (*ConfigKV, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("etcd: no endpoints")
	}
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "/opshub/config"
	}
	p = strings.TrimRight(p, "/") + "/"

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &ConfigKV{cli: cli, prefix: p}, nil
}

func (k *ConfigKV) Close() error {
	if k == nil || k.cli == nil {
		return nil
	}
	return k.cli.Close()
}

func (k *ConfigKV) fullKey(logicalKey string) string {
	return k.prefix + logicalKey
}

func (k *ConfigKV) Get(ctx context.Context, logicalKey string) ([]byte, error) {
	resp, err := k.cli.Get(ctx, k.fullKey(logicalKey))
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	return resp.Kvs[0].Value, nil
}

func (k *ConfigKV) Put(ctx context.Context, logicalKey string, value []byte) error {
	_, err := k.cli.Put(ctx, k.fullKey(logicalKey), string(value))
	return err
}

func (k *ConfigKV) Delete(ctx context.Context, logicalKey string) (deleted bool, err error) {
	resp, err := k.cli.Delete(ctx, k.fullKey(logicalKey))
	if err != nil {
		return false, err
	}
	return resp.Deleted > 0, nil
}

func (k *ConfigKV) List(ctx context.Context) (map[string][]byte, error) {
	resp, err := k.cli.Get(ctx, k.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		key := strings.TrimPrefix(string(kv.Key), k.prefix)
		if key == "" {
			continue
		}
		out[key] = kv.Value
	}
	return out, nil
}

// ListSub 列出 prefix+subPrefix 前缀下的全部 KV（不要求 subPrefix 已存在）。
// 用于按逻辑 key 前缀查询，例如审计历史 auditKV.ListSub(ctx, "pay/gateway/timeout/")。
func (k *ConfigKV) ListSub(ctx context.Context, subPrefix string) (map[string][]byte, error) {
	resp, err := k.cli.Get(ctx, k.fullKey(subPrefix), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		key := strings.TrimPrefix(string(kv.Key), k.prefix)
		if key == "" {
			continue
		}
		out[key] = kv.Value
	}
	return out, nil
}

// Ping 探测与 etcd 集群的连通性（用于 readiness）。
func (k *ConfigKV) Ping(ctx context.Context) error {
	if k == nil || k.cli == nil {
		return fmt.Errorf("etcd: no client")
	}
	eps := k.cli.Endpoints()
	if len(eps) == 0 {
		return fmt.Errorf("etcd: no endpoints")
	}
	_, err := k.cli.Status(ctx, eps[0])
	return err
}

// Incr 原子递增 logicalKey 下的整数并返回新值（不存在则初始化为 1）。
// 用 etcd 事务 CAS 保证并发安全，用于全局配置版本号等场景。
func (k *ConfigKV) Incr(ctx context.Context, logicalKey string) (int64, error) {
	full := k.fullKey(logicalKey)
	for {
		resp, err := k.cli.Get(ctx, full)
		if err != nil {
			return 0, err
		}
		cur := int64(0)
		if len(resp.Kvs) > 0 {
			cur, _ = strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
		}
		next := cur + 1

		var cmp clientv3.Cmp
		if len(resp.Kvs) > 0 {
			cmp = clientv3.Compare(clientv3.Value(full), "=", string(resp.Kvs[0].Value))
		} else {
			cmp = clientv3.Compare(clientv3.CreateRevision(full), "=", 0)
		}
		txnResp, err := k.cli.Txn(ctx).
			If(cmp).
			Then(clientv3.OpPut(full, strconv.FormatInt(next, 10))).
			Commit()
		if err != nil {
			return 0, err
		}
		if txnResp.Succeeded {
			return next, nil
		}
		// 其他写入者抢先了，重试
	}
}
