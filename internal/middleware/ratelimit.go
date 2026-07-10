package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// IPRateLimiter is a simple fixed-window limiter keyed by client IP, meant
// to blunt scripted abuse of the public submit endpoint without imposing
// CAPTCHA-level friction on real visitors.
type IPRateLimiter struct {
	mu       sync.Mutex
	windows  map[string]*window
	limit    int
	interval time.Duration
}

type window struct {
	count   int
	resetAt time.Time
}

func NewIPRateLimiter(limit int, interval time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		windows:  make(map[string]*window),
		limit:    limit,
		interval: interval,
	}
}

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	w, ok := l.windows[ip]
	if !ok || now.After(w.resetAt) {
		w = &window{count: 0, resetAt: now.Add(l.interval)}
		l.windows[ip] = w
	}
	if w.count >= l.limit {
		return false
	}
	w.count++
	return true
}

// Middleware rejects requests once an IP exceeds the configured rate,
// responding 429 rather than silently dropping the request.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.Allow(ip) {
			http.Error(w, "too many requests, please try again later", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// Caddy appends the real client IP as the first entry.
		if idx := strings.IndexByte(fwd, ','); idx >= 0 {
			return fwd[:idx]
		}
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
