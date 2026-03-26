package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthHandler handles liveness and readiness probes.
type HealthHandler struct {
	pool  *pgxpool.Pool
	redis *redis.Client
	env   string
}

func NewHealthHandler(pool *pgxpool.Pool, redis *redis.Client, env string) *HealthHandler {
	return &HealthHandler{pool: pool, redis: redis, env: env}
}

// Health returns basic liveness.
// GET /api/health
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"service":     "web3-account-api",
		"environment": h.env,
	})
}

// Ready checks Postgres and Redis connectivity.
// GET /api/health/ready
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx := c.Request.Context()
	components := gin.H{}

	// Check Postgres
	if err := h.pool.Ping(ctx); err != nil {
		components["postgres"] = "error: " + err.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "components": components})
		return
	}
	components["postgres"] = "connected"

	// Check Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		components["redis"] = "error: " + err.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "components": components})
		return
	}
	components["redis"] = "connected"

	c.JSON(http.StatusOK, gin.H{"status": "ok", "components": components})
}
