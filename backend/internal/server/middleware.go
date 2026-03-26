package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/web3-lab/backend/internal/services"
	"github.com/web3-lab/backend/pkg/logs"
)

// requestIDMiddleware injects or generates X-Request-ID.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}

// zapMiddleware logs each request with zap.
func zapMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Create per-request logger
		reqLogger := logs.Logger.With(
			zap.String("request_id", c.GetString("request_id")),
			zap.String("client_ip", c.ClientIP()),
		)
		ctx := logs.WithContext(c.Request.Context(), reqLogger)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		reqLogger.Info("request",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Duration("latency", time.Since(start)),
		)
	}
}

// recoveryMiddleware catches panics and logs with zap.
func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logs.FromContext(c.Request.Context()).Error("panic recovered",
					zap.Any("error", r),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{"code": "INTERNAL_ERROR", "message": "internal server error"},
				})
			}
		}()
		c.Next()
	}
}

// corsMiddleware handles CORS with configurable allowed origins and dynamic AppClients origins.
func corsMiddleware(staticAllowedOrigins []string, clientService *services.AppClientService) gin.HandlerFunc {
	staticOrigins := make(map[string]bool, len(staticAllowedOrigins))
	for _, o := range staticAllowedOrigins {
		staticOrigins[o] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		isAllowed := staticOrigins[origin] || len(staticAllowedOrigins) == 0
		if !isAllowed && origin != "" {
			// Check dynamic cache
			cachedAllowed, _ := clientService.IsCORSAllowed(c.Request.Context(), origin)
			isAllowed = cachedAllowed
		}

		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Session-Token, Accept, Origin")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
