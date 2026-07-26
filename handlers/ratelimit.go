package handlers

import (
	"strings"
	"sync"
	"time"
)

// slidingWindowLimiter counts recent events per key and rejects once a key
// exceeds max events within window. It's process-local (no shared store),
// which is sufficient for Avenue's single-instance deployment model and
// avoids adding an infra dependency just to slow down login/reset abuse.
type slidingWindowLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	events map[string][]time.Time
}

func newSlidingWindowLimiter(max int, window time.Duration) *slidingWindowLimiter {
	return &slidingWindowLimiter{
		max:    max,
		window: window,
		events: make(map[string][]time.Time),
	}
}

// allow records an attempt for key and reports whether it's still within
// the limit. Expired timestamps are pruned on access so the map can't grow
// without bound for a key that stops being used.
func (l *slidingWindowLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	kept := l.events[key][:0]
	for _, t := range l.events[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.max {
		l.events[key] = kept
		return false
	}

	l.events[key] = append(kept, now)
	return true
}

// reset clears recorded attempts for key, e.g. after a successful login so a
// legitimate user isn't penalized by earlier failed attempts.
func (l *slidingWindowLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.events, key)
}

// Separate IP and per-account limiters: the IP limiter slows down a single
// attacker, while the per-account limiter stops credential-stuffing spread
// across many source IPs from ever locking onto one account.
var (
	loginIPLimiter    = newSlidingWindowLimiter(20, time.Minute)
	loginEmailLimiter = newSlidingWindowLimiter(5, 15*time.Minute)

	forgotPasswordIPLimiter    = newSlidingWindowLimiter(5, 15*time.Minute)
	forgotPasswordEmailLimiter = newSlidingWindowLimiter(3, time.Hour)
)

func normalizeEmailKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
