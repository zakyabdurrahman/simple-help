// Package middleware holds HTTP middleware (auth, role checks).
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"simplehelp/src/auth"
)

const (
	// CtxUserID is the gin context key for the authenticated user's ID.
	CtxUserID = "user_id"
	// CtxRole is the gin context key for the authenticated user's role.
	CtxRole = "role"
)

// AuthRequired validates a Bearer JWT and loads user ID + role into the
// request context. Aborts with 401 if missing/invalid.
func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		claims, err := auth.Parse(secret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}

// RequireRole aborts unless the authenticated user has one of the given roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		r, _ := role.(string)
		for _, want := range roles {
			if r == want {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "forbidden",
		})
	}
}

// UserIDFrom extracts the authenticated user's ID from the context.
func UserIDFrom(c *gin.Context) int64 {
	v, _ := c.Get(CtxUserID)
	id, _ := v.(int64)
	return id
}

// RoleFrom extracts the authenticated user's role from the context.
func RoleFrom(c *gin.Context) string {
	v, _ := c.Get(CtxRole)
	r, _ := v.(string)
	return r
}
