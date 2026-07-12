package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/config"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware enforces authentication in production.
// In gateway mode, requires valid X-User-Id/X-Username/X-Role headers from the gateway
// and verifies HMAC-SHA256 signature to prevent header forgery.
// Test authentication (X-Test-User-Id) is only available when GIN_MODE=debug AND TEST_AUTH_ENABLED=true.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Cfg.OxeliaGatewayMode {
			userID := c.GetHeader("X-User-Id")
			username := c.GetHeader("X-Username")
			role := c.GetHeader("X-Role")

			if userID != "" && username != "" && role != "" {
				// Verify HMAC signature if secret is configured
				if config.Cfg.GatewayHMACSecret != "" {
					if !verifyGatewaySignature(c) {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid gateway signature"})
						c.Abort()
						return
					}
				}

				uid, err := strconv.ParseInt(userID, 10, 64)
				if err == nil {
					c.Set("user_id", uid)
					c.Set("username", username)
					c.Set("role", role)
					c.Next()
					return
				}
			}

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

// verifyGatewaySignature validates the HMAC-SHA256 signature from the gateway.
// Format: X-Gateway-Signature = HMAC-SHA256(timestamp + user_id + secret)
// Anti-replay: timestamp must be within 5 minutes.
func verifyGatewaySignature(c *gin.Context) bool {
	sig := c.GetHeader("X-Gateway-Signature")
	ts := c.GetHeader("X-Gateway-Timestamp")
	userID := c.GetHeader("X-User-Id")

	if sig == "" || ts == "" {
		return false
	}

	// Anti-replay: timestamp must be within 5 minutes
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || time.Now().Unix()-t > 300 {
		return false
	}

	expected := computeHMAC(ts + userID + config.Cfg.GatewayHMACSecret)
	return hmac.Equal([]byte(sig), []byte(expected))
}

// computeHMAC returns the hex-encoded HMAC-SHA256 of the given message.
func computeHMAC(message string) string {
	mac := hmac.New(sha256.New, []byte(config.Cfg.GatewayHMACSecret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
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