// Package middleware provides Gin middleware built on Zap.
package middleware

import (
	"net/http"
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
