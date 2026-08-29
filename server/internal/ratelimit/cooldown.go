package ratelimit

import (
	"sync"
	"time"
)

const pruneThreshold = 4096

// CoolDown 进程内按 key 冷却；条目超过阈值时淘汰已过期 key，避免无限涨内存。
type CoolDown struct {
	mu   sync.Mutex
	ttl  time.Duration
	last map[string]time.Time
}

func NewCoolDown(ttl time.Duration) *CoolDown {
	return &CoolDown{ttl: ttl, last: make(map[string]time.Time)}
}

// Allow 返回 true 表示本次允许（并记录时间）；false 表示仍在冷却。
func (c *CoolDown) Allow(key string) bool {
	if c.ttl <= 0 {
		return true
	}
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.last) > pruneThreshold {
		c.pruneLocked(now)
	}
	if t, ok := c.last[key]; ok && now.Sub(t) < c.ttl {
		return false
	}
	c.last[key] = now
	return true
}

func (c *CoolDown) pruneLocked(now time.Time) {
	for k, t := range c.last {
		if now.Sub(t) >= c.ttl {
			delete(c.last, k)
		}
	}
}
