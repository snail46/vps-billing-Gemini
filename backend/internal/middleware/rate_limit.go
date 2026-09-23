package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RedisRateLimiter uses Redis fixed window or token bucket
type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if r.client == nil {
		return true, nil
	}

	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, time.Now().Unix()/(int64(window.Seconds())))
	count, err := r.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return true, nil // Fail open on Redis error so legitimate traffic is not blocked
	}

	if count == 1 {
		_ = r.client.Expire(ctx, redisKey, window*2).Err()
	}

	return count <= int64(limit), nil
}

// MemoryRateLimiter for testing and single-node development
type MemoryRateLimiter struct {
	mu      sync.Mutex
	counts  map[string]int
	expires map[string]time.Time
}

func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{
		counts:  make(map[string]int),
		expires: make(map[string]time.Time),
	}
}

func (m *MemoryRateLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	exp, exists := m.expires[key]
	if !exists || now.After(exp) {
		m.counts[key] = 1
		m.expires[key] = now.Add(window)
		return true, nil
	}

	m.counts[key]++
	return m.counts[key] <= limit, nil
}

func RateLimit(limiter RateLimiter, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := r.RemoteAddr
			key := fmt.Sprintf("%s:%s", ip, r.URL.Path)

			allowed, err := limiter.Allow(r.Context(), key, limit, window)
			if err != nil || !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				httpError(w, r, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "errors.too_many_requests", "rate limit exceeded, please retry later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
