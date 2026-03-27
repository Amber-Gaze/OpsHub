package etcd

import (
	"context"
	"fmt"
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
