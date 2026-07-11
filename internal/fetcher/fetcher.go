package fetcher

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/XiaoleC05/SuperRead/internal/summarizer"
	"github.com/mmcdole/gofeed"
)

const maxRSSBodyBytes = 10 * 1024 * 1024

type Fetcher struct {
	parser     *gofeed.Parser
	summarizer *summarizer.Summarizer
}

func New() *Fetcher {
	return &Fetcher{
		parser:     gofeed.NewParser(),
		summarizer: summarizer.New(),
	}
}

func (f *Fetcher) FetchFeed(ctx context.Context, feed *model.Feed) (int, error) {
	fp := gofeed.NewParser()
	if fp.Client == nil {
		fp.Client = &http.Client{Timeout: 30 * time.Second}
	} else {
		fp.Client.Timeout = 30 * time.Second
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.FeedURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := fp.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	parsed, err := fp.Parse(io.LimitReader(resp.Body, maxRSSBodyBytes))
	if err != nil {
		return 0, err
	}

	settings, _ := db.GetSettings(ctx, feed.UserID)

	added := 0
	for _, item := range parsed.Items {
		author := ""
		if item.Author != nil {
			author = item.Author.Name
		}

		article := &model.Article{
			FeedID:      feed.ID,
			Title:       item.Title,
			URL:         item.Link,
			Author:      author,
			GUID:        item.GUID,
			ContentText: extractContent(item),
		}

		if item.PublishedParsed != nil {
			article.PublishedAt = item.PublishedParsed
		}

		articleID, err := db.CreateArticle(ctx, article)
		if err != nil {
			log.Printf("Failed to create article: %v", err)
			continue
		}
		if articleID == 0 {
			continue
		}

		added++
		article.ID = articleID

		if settings != nil && settings.APIKey != "" && settings.APIBase != "" {
			summary, sumErr := f.summarizer.Summarize(ctx, settings, article)
			if sumErr != nil {
				log.Printf("Failed to summarize article %d: %v", articleID, sumErr)
			} else if summary != "" {
				if err := db.UpdateArticleSummary(ctx, articleID, summary); err != nil {
					log.Printf("Failed to save summary for article %d: %v", articleID, err)
				}
			}
		}
	}

	return added, nil
}

func extractContent(item *gofeed.Item) string {
	if item.Content != "" {
		return item.Content
	}
	if item.Description != "" {
		return item.Description
	}
	return ""
}

func FetchAllFeeds(ctx context.Context) {
	feeds, err := db.ListAllFeeds(ctx)
	if err != nil {
		log.Printf("Failed to list feeds: %v", err)
		return
	}

	fetcher := New()
	for _, feed := range feeds {
		added, err := fetcher.FetchFeed(ctx, &feed)
		fetchErr := ""
		if err != nil {
			fetchErr = err.Error()
			log.Printf("Failed to fetch feed %s: %v", feed.FeedURL, err)
		} else {
			log.Printf("Fetched feed %s: %d new articles", feed.FeedURL, added)
		}

		if err := db.UpdateFeedFetchTime(ctx, feed.ID, fetchErr); err != nil {
			log.Printf("Failed to update feed fetch time: %v", err)
		}
	}
}
