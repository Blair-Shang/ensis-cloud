package ratelimit

import (
	"testing"
	"time"
)

func TestCoolDownAllow(t *testing.T) {
	c := NewCoolDown(40 * time.Millisecond)
	if !c.Allow("a") {
		t.Fatal("first allow")
	}
	if c.Allow("a") {
		t.Fatal("still cooling")
	}
	if !c.Allow("b") {
		t.Fatal("other key")
	}
	time.Sleep(50 * time.Millisecond)
	if !c.Allow("a") {
		t.Fatal("after ttl")
	}
}

func TestCoolDownPrune(t *testing.T) {
	c := NewCoolDown(time.Millisecond)
	c.mu.Lock()
	c.last["old"] = time.Now().Add(-time.Second)
	c.last["fresh"] = time.Now()
	c.pruneLocked(time.Now())
	if _, ok := c.last["old"]; ok {
		t.Fatal("old key should be pruned")
	}
	if _, ok := c.last["fresh"]; !ok {
		t.Fatal("fresh key should remain")
	}
	c.mu.Unlock()
}
