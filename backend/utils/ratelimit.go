package utils

import (
	"sync"
	"time"
)

// TokenBucket is a simple in-memory per-key rate limiter.
type TokenBucket struct {
	mu       sync.Mutex
	visitors map[string]*bucket
	rate     float64 // tokens per second
	burst    float64
	ttl      time.Duration
}

type bucket struct {
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

// NewTokenBucket creates a limiter that refills at rate tokens/sec up to burst.
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	if rate <= 0 {
		rate = 1
	}
	if burst < 1 {
		burst = 1
	}
	tb := &TokenBucket{
		visitors: make(map[string]*bucket),
		rate:     rate,
		burst:    float64(burst),
		ttl:      5 * time.Minute,
	}
	go tb.cleanupLoop()
	return tb
}

// Allow reports whether key may proceed (consumes 1 token).
func (tb *TokenBucket) Allow(key string) bool {
	if key == "" {
		key = "unknown"
	}
	now := time.Now()

	tb.mu.Lock()
	defer tb.mu.Unlock()

	b, ok := tb.visitors[key]
	if !ok {
		tb.visitors[key] = &bucket{
			tokens:   tb.burst - 1,
			last:     now,
			lastSeen: now,
		}
		return true
	}

	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * tb.rate
	if b.tokens > tb.burst {
		b.tokens = tb.burst
	}
	b.last = now
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (tb *TokenBucket) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		tb.mu.Lock()
		cutoff := time.Now().Add(-tb.ttl)
		for k, b := range tb.visitors {
			if b.lastSeen.Before(cutoff) {
				delete(tb.visitors, k)
			}
		}
		tb.mu.Unlock()
	}
}

// ClientIP extracts a best-effort client IP from request headers / RemoteAddr.
func ClientIP(remoteAddr, xff, xri string) string {
	if xff != "" {
		// first hop in X-Forwarded-For
		if i := indexByte(xff, ','); i >= 0 {
			return trimSpace(xff[:i])
		}
		return trimSpace(xff)
	}
	if xri != "" {
		return trimSpace(xri)
	}
	// strip port from host:port
	host := remoteAddr
	if i := lastIndexByte(host, ':'); i >= 0 {
		// handle [ipv6]:port
		if host[0] == '[' {
			if j := indexByte(host, ']'); j >= 0 {
				return host[1:j]
			}
		}
		return host[:i]
	}
	return host
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
