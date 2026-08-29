package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache 封装 go-redis 客户端，提供 []byte 读写、TTL 与按前缀删除，
// 供配置中心 L1 缓存、IAM 令牌黑名单等场景复用。
type Cache struct {
	cli *redis.Client
	ttl time.Duration
}

// Options 创建缓存所需的连接参数。
type Options struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      time.Duration
}

// NewCache 建立并探活 Redis 连接；连接失败返回 error。
func NewCache(opts Options) (*Cache, error) {
	host := opts.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := opts.Port
	if port <= 0 {
		port = 6379
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	cli := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: opts.Password,
		DB:       opts.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := cli.Ping(ctx).Err(); err != nil {
		_ = cli.Close()
		return nil, err
	}
	return &Cache{cli: cli, ttl: ttl}, nil
}

// Close 关闭底层连接。
func (c *Cache) Close() error {
	if c == nil || c.cli == nil {
		return nil
	}
	return c.cli.Close()
}

// TTL 返回默认过期时间。
func (c *Cache) TTL() time.Duration {
	return c.ttl
}

// Get 读取缓存值；miss 或出错时返回 ok=false。
func (c *Cache) Get(ctx context.Context, key string) (b []byte, ok bool) {
	b, err := c.cli.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return b, true
}

// Set 写入缓存值，使用默认 TTL。
func (c *Cache) Set(ctx context.Context, key string, value []byte) error {
	return c.cli.Set(ctx, key, value, c.ttl).Err()
}

// SetWithTTL 写入缓存值并指定过期时间。
func (c *Cache) SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.cli.Set(ctx, key, value, ttl).Err()
}

// SetString 以字符串形式写入（令牌黑名单等文本场景）。
func (c *Cache) SetString(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.cli.Set(ctx, key, value, ttl).Err()
}

// GetString 读取字符串值；miss 或出错时返回 ok=false。
func (c *Cache) GetString(ctx context.Context, key string) (string, bool) {
	s, err := c.cli.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return s, true
}

// Delete 删除一个或多个 key。
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.cli.Del(ctx, keys...).Err()
}

// DelPrefix 删除指定前缀下的所有 key（SCAN + DEL，避免 KEYS 阻塞）。
func (c *Cache) DelPrefix(ctx context.Context, prefix string) error {
	var cursor uint64
	for {
		keys, next, err := c.cli.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.cli.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return nil
}
