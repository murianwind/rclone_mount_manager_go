package main

import (
	"sync"
	"time"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

const (
	// autoMountBackoffBase equals the network monitor's poll interval, so
	// the first failure changes nothing — only a *repeated* failure slows
	// the retries down.
	autoMountBackoffBase = 10 * time.Second
	autoMountBackoffMax  = 10 * time.Minute
	// autoMountStableRun is how long a mount must have stayed up before a
	// later failure counts as a brand-new incident rather than part of an
	// ongoing failure streak.
	autoMountStableRun = 30 * time.Second
)

// autoMountBackoffDelay is how long to wait before the next automatic
// attempt after `failures` consecutive failures: base, 2x, 4x, ... capped
// at autoMountBackoffMax. Without it, a mount that can never succeed (bad
// credentials, a folder that already exists, ...) respawns an rclone
// process, rewrites the log, and rebuilds the tray menu every 10 seconds
// for as long as the app runs — and floods the 1000-line log, pushing out
// the history you'd actually want to read.
func autoMountBackoffDelay(failures int) time.Duration {
	if failures <= 0 {
		return 0
	}
	d := autoMountBackoffBase
	for i := 1; i < failures; i++ {
		d *= 2
		if d >= autoMountBackoffMax {
			return autoMountBackoffMax
		}
	}
	return d
}

type backoffState struct {
	failures int
	next     time.Time
}

// backoffTracker remembers, per mount ID, how many automatic attempts in a
// row failed and when the next one is allowed. The zero value is ready to
// use.
type backoffTracker struct {
	mu    sync.Mutex
	state map[string]backoffState
}

// shouldSkip reports whether an automatic attempt for id should wait.
func (t *backoffTracker) shouldSkip(id string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.state[id]
	return ok && now.Before(s.next)
}

// recordExit records how an automatic attempt ended. A clean exit — or a
// failure after the mount had stayed up for autoMountStableRun — starts
// over; only consecutive quick failures lengthen the wait.
func (t *backoffTracker) recordExit(id string, failed bool, ranFor time.Duration, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !failed || ranFor >= autoMountStableRun {
		delete(t.state, id)
	}
	if !failed {
		return
	}
	if t.state == nil {
		t.state = make(map[string]backoffState)
	}
	s := t.state[id]
	s.failures++
	s.next = now.Add(autoMountBackoffDelay(s.failures))
	t.state[id] = s
}

func (t *backoffTracker) reset(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.state, id)
}

// resetAll clears every mount's state — used when the network comes back,
// so a mount that was only failing because we were offline retries
// immediately instead of waiting out a backoff it never deserved.
func (t *backoffTracker) resetAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = nil
}

// mountsDueForAutoMount picks which auto-mount entries may be attempted
// right now: AutoMount is on and the mount isn't waiting out a backoff.
func mountsDueForAutoMount(mounts []engine.Mount, t *backoffTracker, now time.Time) []engine.Mount {
	due := make([]engine.Mount, 0, len(mounts))
	for _, m := range mounts {
		if m.AutoMount && !t.shouldSkip(m.ID, now) {
			due = append(due, m)
		}
	}
	return due
}
