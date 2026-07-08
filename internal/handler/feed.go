package handler

import (
	"net/http"
	"strconv"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if feed == nil || feed.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "feed not found"})
		return
	}

	f := fetcher.New()
	added, err := f.FetchFeed(c.Request.Context(), feed)

	fetchErr := ""
	if err != nil {
		fetchErr = err.Error()
	}

	if err := db.UpdateFeedFetchTime(c.Request.Context(), feed.ID, fetchErr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"added": added})
}
