package handler

import (
	"errors"
	"net/http"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type statsResponse struct {
	APIKeyConfigured bool   `json:"api_key_configured"`
	APIKeyMasked     string `json:"api_key_masked"`
	LastUpdated      string `json:"last_updated"`
	RequestCount     int    `json:"request_count"`
	TokenTotal       int    `json:"token_total"`
}

// Stats GET /api/stats 鈥?杩斿洖鐢ㄦ埛 API Key 閰嶇疆鐘舵€佷笌鐢ㄩ噺缁熻
func Stats(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	var apiKey string
	var updatedAt *string

	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT api_key, updated_at::text FROM superread.user_settings WHERE user_id = $1`,
		userID,
	).Scan(&apiKey, &updatedAt)

	resp := statsResponse{}

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "鏌ヨ澶辫触"})
			return
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.APIKeyConfigured = apiKey != ""
	resp.APIKeyMasked = maskAPIKey(apiKey)
	if updatedAt != nil {
		resp.LastUpdated = *updatedAt
	}

	c.JSON(http.StatusOK, resp)
}