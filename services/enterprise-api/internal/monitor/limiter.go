package monitor

import (
	"sync"
	"time"
)

// Limiter is a per-key token bucket. A non-positive rate allows every request.
type Limiter struct {
	rps     float64
	burst   float64
	now     func() time.Time
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens  float64
	updated time.Time
}

// NewLimiter returns a process-local limiter.
func NewLimiter(rps, burst int) *Limiter {
	if burst < rps {
		burst = rps
	}
	return &Limiter{
		rps:     float64(rps),
		burst:   float64(burst),
		now:     time.Now,
		buckets: map[string]*bucket{},
	}
}

// Allow reports whether the key may proceed.
func (l *Limiter) Allow(key string) bool {
	if l == nil || l.rps <= 0 {
		return true
	}
	if key == "" {
		key = "unknown"
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.buckets) > 4096 {
		for visitor, item := range l.buckets {
			if now.Sub(item.updated) > 2*time.Minute {
				delete(l.buckets, visitor)
			}
		}
	}
	item := l.buckets[key]
	if item == nil {
		item = &bucket{tokens: l.burst, updated: now}
		l.buckets[key] = item
	}
	elapsed := now.Sub(item.updated).Seconds()
	item.tokens += elapsed * l.rps
	if item.tokens > l.burst {
		item.tokens = l.burst
	}
	item.updated = now
	if item.tokens < 1 {
		return false
	}
	item.tokens--
	return true
}
