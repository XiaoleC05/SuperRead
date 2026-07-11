package handler

import (
	"net/http"
	"strings"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/llm"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/gin-gonic/gin"
)

type settingsDTO struct {
	UserID           int64  `json:"user_id"`
	APIKey           string `json:"api_key"`
	APIBase          string `json:"api_base"`
	Model            string `json:"model"`
	FetchIntervalMin int    `json:"fetch_interval_min"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "sk-...xxxx"
	}
	return "sk-..." + key[len(key)-4:]
}

func toSettingsDTO(s *model.UserSettings) settingsDTO {
	return settingsDTO{
		UserID:           s.UserID,
		APIKey:           maskAPIKey(s.APIKey),
		APIBase:          s.APIBase,
		Model:            s.Model,
		FetchIntervalMin: s.FetchIntervalMin,
		UpdatedAt:        s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func GetSettings(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	settings, err := db.GetSettings(c.Request.Context(), userID)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if settings == nil {
		settings = &model.UserSettings{
			UserID:           userID,
			APIKey:           "",
			APIBase:          "",
			Model:            "gpt-4o-mini",
			FetchIntervalMin: 30,
		}
	}

	c.JSON(http.StatusOK, gin.H{"settings": toSettingsDTO(settings)})
}

func UpdateSettings(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	var req model.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.APIBase != nil && strings.TrimSpace(*req.APIBase) != "" {
		if err := llm.ValidateAPIBase(*req.APIBase); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 API 地址"})
			return
		}
	}

	settings, err := db.UpdateSettings(c.Request.Context(), userID, req)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": toSettingsDTO(settings)})
}
