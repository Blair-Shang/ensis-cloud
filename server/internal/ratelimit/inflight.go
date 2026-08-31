package ratelimit

import "sync"

// Inflight 限制同一 key 同时进行的数量。
type Inflight struct {
	mu  sync.Mutex
	cur map[string]int
	max int
}

// NewInflight 创建并发闸；max<=0 时按 2。
func NewInflight(max int) *Inflight {
	if max <= 0 {
		max = 2
	}
	return &Inflight{cur: make(map[string]int), max: max}
}

// Acquire 占用一个名额；已满返回 false。
func (g *Inflight) Acquire(key string) bool {
	if g == nil || key == "" {
		return true
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cur[key] >= g.max {
		return false
	}
	g.cur[key]++
	return true
}

// Release 归还一个名额。
func (g *Inflight) Release(key string) {
	if g == nil || key == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	n := g.cur[key] - 1
	if n <= 0 {
		delete(g.cur, key)
		return
	}
	g.cur[key] = n
}
