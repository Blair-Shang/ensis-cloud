package ratelimit

import "testing"

func TestInflightCap(t *testing.T) {
	g := NewInflight(2)
	if !g.Acquire("u") || !g.Acquire("u") {
		t.Fatal("first two")
	}
	if g.Acquire("u") {
		t.Fatal("third should fail")
	}
	g.Release("u")
	if !g.Acquire("u") {
		t.Fatal("after release")
	}
	g.Release("u")
	g.Release("u")
	if !g.Acquire("u") {
		t.Fatal("after drain")
	}
}
