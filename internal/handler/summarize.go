package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/XiaoleC05/SuperRead/internal/summarizer"
	"github.com/gin-gonic/gin"
)

// Summarize POST /api/summarize?count=N - generate AI summaries for the most recent N unsummarized articles
func Summarize(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	// Parse count from query (default 10)
	count := 10
	if countStr := c.Query("count"); countStr != "" {
		if n, err := strconv.Atoi(countStr); err == nil && n > 0 {
			count = n
		}
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

	// Fetch recent articles (limit scaled to account for already-summarized ones)
	fetchLimit := count * 5
	if fetchLimit < 50 {
		fetchLimit = 50
	}
	articles, err := db.ListArticles(c.Request.Context(), userID, nil, nil, nil, fetchLimit)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	// Filter articles without a summary, take first N
	var pending []model.Article
	for _, a := range articles {
		if a.Summary == "" {
			pending = append(pending, a)
			if len(pending) >= count {
				break
			}
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