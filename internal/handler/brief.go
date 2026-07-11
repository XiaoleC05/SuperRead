package handler

import (
	"net/http"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/gin-gonic/gin"
)

func GetDailyBrief(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	articles, err := db.ListArticlesByDateRange(c.Request.Context(), userID, start, end)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	brief := make([]map[string]interface{}, 0, len(articles))
	for _, a := range articles {
		item := map[string]interface{}{
			"id":        a.ID,
			"feed_id":   a.FeedID,
			"title":     a.Title,
			"url":       a.URL,
			"author":    a.Author,
			"summary":   a.Summary,
			"published": a.PublishedAt,
		}
		brief = append(brief, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"date":     start.Format("2006-01-02"),
		"articles": brief,
		"total":    len(brief),
	})
}
