package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/fetcher"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/gin-gonic/gin"
)

func ListFeeds(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	feeds, err := db.ListFeeds(c.Request.Context(), userID)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if feeds == nil {
		feeds = []model.Feed{}
	}

	c.JSON(http.StatusOK, gin.H{"feeds": feeds})
}

func CreateFeed(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	var req model.CreateFeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	feed, err := db.CreateFeed(c.Request.Context(), userID, req)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"feed": feed})
}

func DeleteFeed(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := db.DeleteFeed(c.Request.Context(), id, userID); err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func FetchFeed(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	feed, err := db.GetFeed(c.Request.Context(), id)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if feed == nil || feed.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "feed not found"})
		return
	}

	// Skip if fetched within 24 hours
	if feed.LastFetchedAt != nil && time.Since(*feed.LastFetchedAt) < 24*time.Hour {
		c.JSON(http.StatusOK, gin.H{
			"added":           0,
			"skipped":          true,
			"message":          "fetched within 24h, skipping",
			"last_fetched_at": feed.LastFetchedAt,
		})
		return
	}

	f := fetcher.New()
	added, err := f.FetchFeed(c.Request.Context(), feed)

	fetchErr := ""
	if err != nil {
		fetchErr = err.Error()
	}

	if err := db.UpdateFeedFetchTime(c.Request.Context(), feed.ID, fetchErr); err != nil {
		respondInternalError(c, err)
		return
	}

	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"added": added})
}
