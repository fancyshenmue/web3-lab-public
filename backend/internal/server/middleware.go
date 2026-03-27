package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
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

// jwtAuthMiddleware extracts the Bearer token's `sub` claim (Kratos identity UUID),
// resolves it to the app-level account_id via the identity lookup, and stores both
// in the gin context. Signature verification is handled upstream by Oathkeeper/gateway.
func jwtAuthMiddleware(accountService *services.AccountService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "missing or invalid Authorization header",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Decode JWT payload (second segment) without verification —
		// signature is validated upstream by Oathkeeper/API gateway.
		parts := strings.Split(tokenStr, ".")
		if len(parts) != 3 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "malformed JWT",
			})
			return
		}

		payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "invalid JWT payload encoding",
			})
			return
		}

		var claims struct {
			Sub string `json:"sub"`
		}
		if err := json.Unmarshal(payloadBytes, &claims); err != nil || claims.Sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "JWT missing sub claim",
			})
			return
		}

		// Parse the sub as a Kratos identity UUID
		kratosID, err := uuid.Parse(claims.Sub)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "invalid sub format",
			})
			return
		}

		// Resolve to app-level account
		ident, err := accountService.GetAccountIdentityByKratosID(c.Request.Context(), kratosID)
		if err != nil || ident == nil {
			logs.FromContext(c.Request.Context()).Warn("JWT auth: identity not found",
				zap.String("kratos_id", kratosID.String()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": "UNAUTHORIZED", "message": "account not found for token subject",
			})
			return
		}

		c.Set("account_id", ident.AccountID)
		c.Set("identity_id", ident.IdentityID)
		c.Set("kratos_identity_id", kratosID)
		c.Next()
	}
}
