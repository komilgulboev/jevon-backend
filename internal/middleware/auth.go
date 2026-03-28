package middleware

import (
	"net/http"
	"strings"

	"jevon/internal/auth"

	"github.com/gin-gonic/gin"
)

const ClaimsKey = "claims"

type UserClaims = auth.Claims

func RequireAuth(authSvc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "missing or malformed token"})
			return
		}
		claims, err := authSvc.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool)
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		v, _ := c.Get(ClaimsKey)
		claims, ok := v.(*auth.Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		if !allowed[claims.RoleName] {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": "access denied", "your_role": claims.RoleName})
			return
		}
		c.Next()
	}
}

// GetClaims extracts claims from gin context — use in handlers
func GetClaims(c *gin.Context) *auth.Claims {
	v, _ := c.Get(ClaimsKey)
	claims, _ := v.(*auth.Claims)
	return claims
}
