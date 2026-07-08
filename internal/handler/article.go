package handler

import (
	"net/http"
	"strconv"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/gin-gonic/gin"
)

func ListArticles(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	var feedID *int64
	if fidStr := c.Query("feed_id"); fidStr != "" {
		fid, err := strconv.ParseInt(fidStr, 10, 64)
		if err == nil {
			feedID = &fid
		}
	}

	var starred *bool
	if starredStr := c.Query("starred"); starredStr != "" {
		s := starredStr == "true"
		starred = &s
	}

	var tag *string
	if tagStr := c.Query("tag"); tagStr != "" {
		tag = &tagStr
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	articles, err := db.ListArticles(c.Request.Context(), userID, feedID, starred, tag, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if articles == nil {
		articles = []model.Article{}
	}

	c.JSON(http.StatusOK, gin.H{"articles": articles})
}

func UpdateArticle(c *gin.Context) {
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

	var req model.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	article, err := db.UpdateArticle(c.Request.Context(), id, userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"article": article})
}
