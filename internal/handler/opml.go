package handler

import (
	"encoding/xml"
	"io"
	"net/http"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/gin-gonic/gin"
)

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Body    struct {
		Outline []Outline `xml:"outline"`
	} `xml:"body"`
}

type Outline struct {
	XMLName xml.Name `xml:"outline"`
	Text    string   `xml:"text,attr"`
	Title   string   `xml:"title,attr"`
	Type    string   `xml:"type,attr"`
	XMLURL  string   `xml:"xmlUrl,attr"`
	HTMLURL string   `xml:"htmlUrl,attr"`
	Outline []Outline `xml:"outline"`
}

func ImportOPML(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
		return
	}

	var opml OPML
	if err := xml.Unmarshal(data, &opml); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid OPML format"})
		return
	}

	feeds := extractFeeds(opml.Body.Outline)

	imported := 0
	var errors []string

	for _, feed := range feeds {
		req := model.CreateFeedRequest{
			Title:   feed.Title,
			FeedURL: feed.FeedURL,
			SiteURL: feed.SiteURL,
		}

		_, err := db.CreateFeed(c.Request.Context(), userID, req)
		if err != nil {
			errors = append(errors, feed.FeedURL+": "+err.Error())
			continue
		}
		imported++
	}

	result := model.OPMLImportResponse{
		Imported: imported,
		Errors:   errors,
	}

	c.JSON(http.StatusOK, result)
}

type feedInfo struct {
	Title   string
	FeedURL string
	SiteURL string
}

func extractFeeds(outlines []Outline) []feedInfo {
	var feeds []feedInfo

	for _, outline := range outlines {
		if outline.XMLURL != "" {
			title := outline.Title
			if title == "" {
				title = outline.Text
			}
			feeds = append(feeds, feedInfo{
				Title:   title,
				FeedURL: outline.XMLURL,
				SiteURL: outline.HTMLURL,
			})
		}

		if len(outline.Outline) > 0 {
			feeds = append(feeds, extractFeeds(outline.Outline)...)
		}
	}

	return feeds
}
