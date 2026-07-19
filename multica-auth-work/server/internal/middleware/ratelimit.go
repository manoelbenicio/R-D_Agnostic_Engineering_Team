package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const localRateLimitMaxEntries = 10_000

type rateLimitIncrement func(context.Context, string, time.Duration) (int64, error)

type localRateLimitEntry struct {
	count     int
	expiresAt time.Time
}

// boundedLocalRateLimiter is the fail-closed fallback for missing or
// unavailable Redis. The hard key cap prevents spoofed client addresses from
// causing unbounded process memory growth. Once full, new unexpired keys are
// rejected until an entry expires.
type boundedLocalRateLimiter struct {
	mu         sync.Mutex
	entries    map[string]localRateLimitEntry
	limit      int
	window     time.Duration
	maxEntries int
	now        func() time.Time
}

func newBoundedLocalRateLimiter(limit int, window time.Duration, maxEntries int, now func() time.Time) *boundedLocalRateLimiter {
	return &boundedLocalRateLimiter{
		entries:    make(map[string]localRateLimitEntry),
		limit:      limit,
		window:     window,
		maxEntries: maxEntries,
		now:        now,
	}
}

func (l *boundedLocalRateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if entry, ok := l.entries[key]; ok {
		if !now.Before(entry.expiresAt) {
			entry = localRateLimitEntry{expiresAt: now.Add(l.window)}
		}
		entry.count++
		l.entries[key] = entry
		return entry.count <= l.limit
	}

	if len(l.entries) >= l.maxEntries {
		for entryKey, entry := range l.entries {
			if !now.Before(entry.expiresAt) {
				delete(l.entries, entryKey)
			}
		}
		if len(l.entries) >= l.maxEntries {
			return false
		}
	}

	l.entries[key] = localRateLimitEntry{count: 1, expiresAt: now.Add(l.window)}
	return l.limit >= 1
}

// rateLimitScript atomically increments the counter and sets the TTL on
// first access. Using a Lua script ensures INCR and EXPIRE cannot be
// split by a network failure — if INCR succeeds the TTL is guaranteed
// to be set, preventing a stuck key that acts as a permanent ban.
var rateLimitScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`)

// ParseTrustedProxies parses a comma-separated list of CIDRs into a
// slice of *net.IPNet. Invalid entries are warned and skipped.
// Returns nil if raw is empty (default: never trust X-Forwarded-For).
func ParseTrustedProxies(raw string) []*net.IPNet {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var nets []*net.IPNet
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, cidr, err := net.ParseCIDR(p)
		if err != nil {
			slog.Warn("ratelimit: invalid trusted proxy CIDR, skipping", "cidr", p, "error", err)
			continue
		}
		nets = append(nets, cidr)
	}
	return nets
}

// RateLimit returns a per-IP fixed-window rate limiter backed by Redis with a
// bounded in-process fallback. Missing or unavailable Redis never disables
// enforcement.
//
// trustedProxies controls X-Forwarded-For handling: when the direct
// connection (RemoteAddr) originates from a CIDR in the list, the
// rightmost non-trusted IP in the XFF chain is used as the client IP.
// When the list is empty (default), XFF is never consulted — only
// RemoteAddr is used. This matches the project's conservative trust
// model (see health_realtime.go).
func RateLimit(rdb *redis.Client, limit int, window time.Duration, trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	var increment rateLimitIncrement
	if rdb != nil {
		increment = func(ctx context.Context, key string, window time.Duration) (int64, error) {
			return rateLimitScript.Run(ctx, rdb, []string{key}, int(window.Seconds())).Int64()
		}
	}
	return rateLimitWithIncrement(increment, limit, window, trustedProxies, localRateLimitMaxEntries, time.Now)
}

func rateLimitWithIncrement(increment rateLimitIncrement, limit int, window time.Duration, trustedProxies []*net.IPNet, maxLocalEntries int, now func() time.Time) func(http.Handler) http.Handler {
	fallback := newBoundedLocalRateLimiter(limit, window, maxLocalEntries, now)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r, trustedProxies)
			key := rateLimitKey(r.URL.Path, ip)
			if increment == nil {
				if !fallback.allow(key) {
					writeRateLimitExceeded(w, window)
					return
				}
			} else {
				count, err := increment(r.Context(), key, window)
				if err != nil {
					slog.Warn("ratelimit: Redis unavailable; using bounded local fallback")
					if !fallback.allow(key) {
						writeRateLimitExceeded(w, window)
						return
					}
				} else if count > int64(limit) {
					writeRateLimitExceeded(w, window)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeRateLimitExceeded(w http.ResponseWriter, window time.Duration) {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests"})
}

// extractIP determines the client IP for rate limiting purposes.
// It only honors X-Forwarded-For when RemoteAddr is from a trusted proxy.
func extractIP(r *http.Request, trustedProxies []*net.IPNet) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	if len(trustedProxies) > 0 {
		remoteIP := net.ParseIP(remoteHost)
		if remoteIP != nil && isTrustedProxy(remoteIP, trustedProxies) {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				// Walk right-to-left: the rightmost non-trusted entry is
				// the last hop before the trusted proxy chain.
				parts := strings.Split(xff, ",")
				for i := len(parts) - 1; i >= 0; i-- {
					candidate := net.ParseIP(strings.TrimSpace(parts[i]))
					if candidate != nil && !isTrustedProxy(candidate, trustedProxies) {
						return candidate.String()
					}
				}
			}
		}
	}

	// Default: use RemoteAddr in canonical form.
	if ip := net.ParseIP(remoteHost); ip != nil {
		return ip.String()
	}
	return remoteHost
}

func isTrustedProxy(ip net.IP, cidrs []*net.IPNet) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func rateLimitKey(path, ip string) string {
	sanitized := strings.TrimPrefix(path, "/")
	sanitized = strings.ReplaceAll(sanitized, "/", ":")
	return fmt.Sprintf("mul:ratelimit:%s:%s", sanitized, ip)
}
