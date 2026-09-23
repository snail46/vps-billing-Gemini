package server

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	startTime          = time.Now().UTC()
	totalRequestsCount uint64
)

// MetricsHandler exports Prometheus-formatted operational metrics.
type MetricsHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewMetricsHandler(db *pgxpool.Pool, redisClient *redis.Client) *MetricsHandler {
	return &MetricsHandler{
		db:    db,
		redis: redisClient,
	}
}

func TrackRequestMetric(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&totalRequestsCount, 1)
		next.ServeHTTP(w, r)
	})
}

func (h *MetricsHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(startTime).Seconds()
	totalReqs := atomic.LoadUint64(&totalRequestsCount)

	dbStatus := 0
	if h.db != nil {
		if err := h.db.Ping(r.Context()); err == nil {
			dbStatus = 1
		}
	}

	redisStatus := 0
	if h.redis != nil {
		if err := h.redis.Ping(r.Context()).Err(); err == nil {
			redisStatus = 1
		}
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = fmt.Fprintf(w, `# HELP vps_billing_info System information
# TYPE vps_billing_info gauge
vps_billing_info{version="1.0.0",env="production"} 1

# HELP vps_billing_uptime_seconds Total runtime of the server in seconds
# TYPE vps_billing_uptime_seconds gauge
vps_billing_uptime_seconds %.2f

# HELP vps_billing_requests_total Total number of HTTP requests handled
# TYPE vps_billing_requests_total counter
vps_billing_requests_total %d

# HELP vps_billing_database_healthy PostgreSQL database connection health (1 = ok, 0 = down)
# TYPE vps_billing_database_healthy gauge
vps_billing_database_healthy %d

# HELP vps_billing_redis_healthy Redis connection health (1 = ok, 0 = down)
# TYPE vps_billing_redis_healthy gauge
vps_billing_redis_healthy %d
`, uptime, totalReqs, dbStatus, redisStatus)
}
