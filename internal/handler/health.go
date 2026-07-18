// Package handler contains the HTTP handlers.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHealthHandler constructs a HealthHandler.
func NewHealthHandler(db *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: rdb}
}

// Register mounts the health routes on the given router.
func (h *HealthHandler) Register(r gin.IRouter) {
	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)
}

// Healthz is a liveness probe: it reports that the process is up.
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz is a readiness probe: it confirms that Postgres and Redis are reachable.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{}
	status := http.StatusOK

	if err := h.db.Ping(ctx); err != nil {
		checks["postgres"] = "unreachable"
		status = http.StatusServiceUnavailable
	} else {
		checks["postgres"] = "ok"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "unreachable"
		status = http.StatusServiceUnavailable
	} else {
		checks["redis"] = "ok"
	}

	body := gin.H{"status": "ready", "checks": checks}
	if status != http.StatusOK {
		body["status"] = "degraded"
	}
	c.JSON(status, body)
}
