package fetcher

import (
	"context"
	"log"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/mmcdole/gofeed"
)

type Fetcher struct {
	parser *gofeed.Parser
}

func New() *Fetcher {
	return &Fetcher{
		parser: gofeed.NewParser(),
	}
}

func (f *Fetcher) FetchFeed(ctx context.Context, feed *model.Feed) (int, error) {
	fp := gofeed.NewParser()
	fp.Client.Timeout = 30 * time.Second

	parsed, err := fp.ParseURL(feed.FeedURL)
	if err != nil {
		return 0, err
	}

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

		if err := db.CreateArticle(ctx, article); err != nil {
			log.Printf("Failed to create article: %v", err)
			continue
		}
		added++
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
