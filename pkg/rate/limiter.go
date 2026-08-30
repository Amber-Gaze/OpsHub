package rate

import (
	"sync"
	"time"

	"go.uber.org/ratelimit"
)

var (
	limiterMap = make(map[string]*RateLimiter)
	mutex      = &sync.RWMutex{}
)

type RateLimiter struct {
	rl ratelimit.Limiter
}

func GetLimiter(key string, rps int) *RateLimiter {
	mutex.RLock()
	if l, exists := limiterMap[key]; exists {
		mutex.RUnlock()
		return l
	}
	mutex.RUnlock()

	mutex.Lock()
	defer mutex.Unlock()
	if l, exists := limiterMap[key]; exists {
		return l
	}

	var lim ratelimit.Limiter
	if rps <= 0 {
		// rps<=0 视为不限流；ratelimit.New(0) 会除零 panic
		lim = ratelimit.NewUnlimited()
	} else {
		lim = ratelimit.New(rps)
	}
	l := &RateLimiter{rl: lim}
	limiterMap[key] = l
	return l
}

func Allow(key string, rps int) bool {
	// go.uber.org/ratelimit v0.3.1 的 Take() 返回「下一次允许执行的时间点 time.Time」：
	// 时间点在当前之前（含）→ 立即放行；在未来 → 应休眠到该时刻（被限流）。
	limiter := GetLimiter(key, rps)
	return !limiter.rl.Take().After(time.Now())
}
