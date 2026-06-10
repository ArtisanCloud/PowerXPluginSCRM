package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminSignatureGuard enforces that admin APIs carry a Bearer token.
// This is an explicit guard in front of JWTAuth middleware to satisfy zero-trust requirements.
func AdminSignatureGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Keep local dev mode workable.
		if os.Getenv("POWERX_DEV_MODE") == "1" {
			c.Next()
			return
		}
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			c.Next()
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing authorization",
			},
		})
		c.Abort()
	}
}
