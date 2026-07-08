package cron

import (
	"context"
	"log"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/fetcher"
	"github.com/robfig/cron/v3"
)

var scheduler *cron.Cron

func Start() {
	scheduler = cron.New()

	// Fetch feeds every 30 minutes (default)
	_, err := scheduler.AddFunc("@every 30m", func() {
		log.Println("Cron: Starting scheduled feed fetch")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		fetcher.FetchAllFeeds(ctx)
		log.Println("Cron: Scheduled feed fetch completed")
	})

	if err != nil {
		log.Printf("Failed to add cron job: %v", err)
		return
	}

	scheduler.Start()
	log.Println("Cron scheduler started")
}

func Stop() {
	if scheduler != nil {
		scheduler.Stop()
		log.Println("Cron scheduler stopped")
	}
}
