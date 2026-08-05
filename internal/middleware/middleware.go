// Package middleware provides Gin middleware built on Zap.
package middleware

import (
	"net/http"
	"string"
	"time"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger logs each request with method, path, status, client IP, and latency.
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", time.Since(start)),
		}

		switch {
		case len(c.Errors) > 0:
			log.Error("request", append(fields, zap.String("errors", c.Errors.String()))...)
		case c.Writer.Status() >= http.StatusInternalServerError:
			log.Error("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

// Recovery recovers from panics, logs them via Zap, and returns a 500.
// CORS allows a browser-based frontend hosted on a different origin to call
// this API. allowedOrigins is a comma-separated list from ATS_CORS_ORIGINS
// (e.g. "https://your-frontend.vercel.app,http://localhost:5500"). If empty,
// it defaults to "*" (any origin) — fine for local/dev, tighten it in prod.
func CORS(allowedOrigins string) gin.HandlerFunc {
	list := map[string]bool{}
	for _, o := range strings.Split(allowedOrigins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			list[o] = true
		}
	}
	allowAll := len(list) == 0

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (allowAll || list[origin]) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err any) {
		log.Error("panic recovered",
			zap.Any("error", err),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	})
}
