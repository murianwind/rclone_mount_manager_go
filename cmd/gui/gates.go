package main

import (
	"sync"
	"time"
)

// dialogGateMaxHold is how long a dialog key can stay "open" without a
// close notification before the gate lets a new one through anyway — a
// safety net so a missed close callback can never silence a kind of alert
// for the rest of the session.
const dialogGateMaxHold = 30 * time.Minute

// keyedGate lets at most one holder per key through at a time. Used to
// keep the same alert from stacking up (a permanently failing auto-mount
// is retried every few seconds, and used to open a fresh dialog each
// time), and to avoid queueing a new UI task while the previous one is
// still pending.
type keyedGate struct {
	mu      sync.Mutex
	held    map[string]time.Time
	maxHold time.Duration // 0 = a held key never expires on its own
}

func (g *keyedGate) tryAcquire(key string) bool {
	return g.tryAcquireAt(key, time.Now())
}

func (g *keyedGate) tryAcquireAt(key string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if since, held := g.held[key]; held {
		if g.maxHold <= 0 || now.Sub(since) < g.maxHold {
			return false
		}
	}
	if g.held == nil {
		g.held = make(map[string]time.Time)
	}
	g.held[key] = now
	return true
}

func (g *keyedGate) release(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.held, key)
}
