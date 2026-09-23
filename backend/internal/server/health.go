package server

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"vps-billing/internal/database"
	backendRedis "vps-billing/internal/redis"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redisClient,
	}
}

type ReadyResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	JSON(w, r, http.StatusOK, map[string]string{
		"status": "alive",
	})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)
	isReady := true

	// Check PostgreSQL
	if h.db != nil {
		if err := database.Ping(r.Context(), h.db); err != nil {
			checks["postgres"] = "unhealthy: " + err.Error()
			isReady = false
		} else {
			checks["postgres"] = "healthy"
		}
	} else {
		checks["postgres"] = "not_configured"
		isReady = false
	}

	// Check Redis
	if h.redis != nil {
		if err := backendRedis.Ping(r.Context(), h.redis); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			isReady = false
		} else {
			checks["redis"] = "healthy"
		}
	} else {
		checks["redis"] = "not_configured"
		isReady = false
	}

	resp := ReadyResponse{
		Status:    "ready",
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	}

	if !isReady {
		resp.Status = "unready"
		JSON(w, r, http.StatusServiceUnavailable, resp)
		return
	}

	JSON(w, r, http.StatusOK, resp)
}
