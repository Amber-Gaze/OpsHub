package rate

import (
	"sync"

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

	l := &RateLimiter{
		rl: ratelimit.New(rps),
	}
	limiterMap[key] = l
	return l
}

func Allow(key string, rps int) bool {
	limiter := GetLimiter(key, rps)
	return limiter.rl.Take().IsZero()
}
