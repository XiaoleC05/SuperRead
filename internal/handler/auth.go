package handler

import (
	"net/http"
	"os"
	"strconv"

	"github.com/XiaoleC05/SuperRead/internal/config"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware enforces authentication in production.
// In gateway mode, requires valid X-User-Id/X-Username/X-Role headers from the gateway.
// Test authentication (X-Test-User-Id) is only available when GIN_MODE=debug AND TEST_AUTH_ENABLED=true.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Cfg.OxeliaGatewayMode {
			userID := c.GetHeader("X-User-Id")
			username := c.GetHeader("X-Username")
			role := c.GetHeader("X-Role")

			if userID != "" && username != "" && role != "" {
				uid, err := strconv.ParseInt(userID, 10, 64)
				if err == nil {
					c.Set("user_id", uid)
					c.Set("username", username)
					c.Set("role", role)
					c.Next()
					return
				}
			}

			// Gateway mode with invalid/missing headers: reject immediately
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Non-gateway mode: test auth only if GIN_MODE=debug AND TEST_AUTH_ENABLED=true
		if os.Getenv("GIN_MODE") == "debug" && os.Getenv("TEST_AUTH_ENABLED") == "true" {
			testUserID := c.GetHeader("X-Test-User-Id")
			if testUserID != "" {
				uid, err := strconv.ParseInt(testUserID, 10, 64)
				if err == nil {
					c.Set("user_id", uid)
					c.Set("username", "test-user")
					c.Set("role", "user")
					c.Next()
					return
				}
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		c.Abort()
	}
}

// GetUserID extracts the authenticated user ID from the context.
func GetUserID(c *gin.Context) (int64, bool) {
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return 0, false
	}
	return uid.(int64), true
}
