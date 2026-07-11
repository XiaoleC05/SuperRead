package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func respondInternalError(c *gin.Context, err error) {
	log.Printf("internal error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "内部错误"})
}
