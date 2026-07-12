package handler

import (
	"net/http"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/gin-gonic/gin"
)

type BriefArticle struct {
	ID        int64  `json:"id"`
	FeedID    int64  `json:"feed_id"`
	FeedTitle string `json:"feed_title"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Author    string `json:"author"`
	Summary   string `json:"summary"`
	Published string `json:"published"`
}

func GetDailyBrief(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	articles, err := db.ListRecentSummarizedArticles(c.Request.Context(), userID, 30)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	brief := make([]BriefArticle, 0, len(articles))
	for _, a := range articles {
		published := ""
		if a.PublishedAt != nil {
			published = a.PublishedAt.Format("2006-01-02 15:04")
		}
		brief = append(brief, BriefArticle{
			ID:        a.ID,
			FeedID:    a.FeedID,
			FeedTitle: a.FeedTitle,
			Title:     a.Title,
			URL:       a.URL,
			Author:    a.Author,
			Summary:   a.Summary,
			Published: published,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"date":     time.Now().Format("2006-01-02"),
		"articles": brief,
		"total":    len(brief),
	})
}