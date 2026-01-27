package etcd

import "context"

type ConfigMeta struct{}

func NewConfigMeta() *ConfigMeta {
	return &ConfigMeta{}
}

func (c *ConfigMeta) Put(ctx context.Context, key, value string) error {
	return nil
}

func (c *ConfigMeta) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}
