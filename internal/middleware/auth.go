package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/auth"
	"github.com/faizan/ats/internal/domain"
)

const (
	ctxUserID = "auth.userID"
	ctxRole   = "auth.role"
)

// Auth validates the Bearer access token and stores the user id + role in the
// Gin context for downstream handlers.
func Auth(tm *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := tm.ParseAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxRole, claims.Role)
		c.Next()
	}
}

// RequireRole allows the request only if the caller's role is in roles.
// It must run after Auth.
func RequireRole(roles ...domain.Role) gin.HandlerFunc {
	allowed := make(map[domain.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, ok := c.Get(ctxRole)
		if !ok || !allowed[role.(domain.Role)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// UserID returns the authenticated user's id (0 if unauthenticated).
func UserID(c *gin.Context) int64 {
	v, _ := c.Get(ctxUserID)
	id, _ := v.(int64)
	return id
}

// CurrentRole returns the authenticated user's role ("" if unauthenticated).
func CurrentRole(c *gin.Context) domain.Role {
	v, _ := c.Get(ctxRole)
	r, _ := v.(domain.Role)
	return r
}
