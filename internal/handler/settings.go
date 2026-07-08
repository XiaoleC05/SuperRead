package handler

import (
	"net/http"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/gin-gonic/gin"
)

func GetSettings(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	settings, err := db.GetSettings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if settings == nil {
		settings = &model.UserSettings{
			UserID:          userID,
			APIKey:          "",
			APIBase:         "",
			Model:           "gpt-4o-mini",
			FetchIntervalMin: 30,
		}
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
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

	settings, err := db.UpdateSettings(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}
