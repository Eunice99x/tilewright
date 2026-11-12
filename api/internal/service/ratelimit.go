package service

import (
	"sync"
	"time"
)

const (
	loginEmailLimit = 10
	loginIPLimit    = 30
	loginWindow     = 15 * time.Minute
	signupIPLimit   = 5

	emailAddressLimit = 5
	emailIPLimit      = 20
	signupWindow      = 24 * time.Hour
)

type attempts struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	counts    map[string]*attemptCount
	lastSweep time.Time
}

type attemptCount struct {
	n     int
	reset time.Time
}

func newAttempts(limit int, window time.Duration) *attempts {
	return &attempts{limit: limit, window: window, counts: map[string]*attemptCount{}}
}

func (a *attempts) Hit(key string, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if now.Sub(a.lastSweep) > time.Minute {
		for k, c := range a.counts {
			if !now.Before(c.reset) {
				delete(a.counts, k)
			}
		}
		a.lastSweep = now
	}
	c, ok := a.counts[key]
	if !ok || !now.Before(c.reset) {
		a.counts[key] = &attemptCount{n: 1, reset: now.Add(a.window)}
		return true
	}
	if c.n >= a.limit {
		return false
	}
	c.n++
	return true
}

func (a *attempts) Blocked(key string, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	c, ok := a.counts[key]
	return ok && now.Before(c.reset) && c.n >= a.limit
}
