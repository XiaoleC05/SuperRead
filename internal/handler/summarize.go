package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/XiaoleC05/SuperRead/internal/summarizer"
	"github.com/gin-gonic/gin"
)

// Summarize POST /api/summarize - generate AI summaries for today's unsummarized articles
func Summarize(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	// Load user settings for LLM API credentials
	settings, err := db.GetSettings(c.Request.Context(), userID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if settings == nil || settings.APIKey == "" || settings.APIBase == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key not configured"})
		return
	}

	// Get today's articles for the user
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	articles, err := db.ListArticlesByDateRange(c.Request.Context(), userID, start, end)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	// Filter articles without a summary
	var pending []model.Article
	for _, a := range articles {
		if a.Summary == "" {
			pending = append(pending, a)
		}
	}

	if len(pending) == 0 {
		c.JSON(http.StatusOK, gin.H{"summarized": 0, "failed": 0})
		return
	}

	// Summarize each article
	s := summarizer.New()
	summarized := 0
	failed := 0

	for i := range pending {
		article := &pending[i]
		summary, err := s.Summarize(c.Request.Context(), settings, article)
		if err != nil {
			log.Printf("summarize article %d failed: %v", article.ID, err)
			failed++
			continue
		}
		if summary == "" {
			failed++
			continue
		}
		if err := db.UpdateArticleSummary(c.Request.Context(), article.ID, summary); err != nil {
			log.Printf("update article %d summary failed: %v", article.ID, err)
			failed++
			continue
		}
		summarized++
	}

	c.JSON(http.StatusOK, gin.H{
		"summarized": summarized,
		"failed":     failed,
	})
}