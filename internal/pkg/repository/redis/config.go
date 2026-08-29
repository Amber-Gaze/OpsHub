package redis

import (
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
)

// NewCacheFromConfig 根据全局配置构建 Redis 缓存。
// 未配置或显式禁用时返回 (nil, nil)；连接失败返回 error。
func NewCacheFromConfig() (*Cache, error) {
	conf := options.GetRedisConf()
	if conf == nil || !conf.IsEnabled() {
		return nil, nil
	}
	return NewCache(Options{
		Host:     conf.Host,
		Port:     conf.Port,
		Password: conf.Password,
		DB:       conf.DB,
		TTL:      5 * time.Minute,
	})
}
