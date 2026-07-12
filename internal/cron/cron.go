package cron

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/config"
	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/fetcher"
	"github.com/XiaoleC05/SuperRead/internal/mailer"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/XiaoleC05/SuperRead/internal/summarizer"
	"github.com/robfig/cron/v3"
)

var scheduler *cron.Cron

func Start() {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	scheduler = cron.New(cron.WithLocation(loc))

	fetchInterval := config.Cfg.FetchCronInterval
	if fetchInterval == "" {
		fetchInterval = "@every 30m"
	}
	_, err := scheduler.AddFunc(fetchInterval, func() {
		log.Println("Cron: feed fetch started")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		fetcher.FetchAllFeeds(ctx)
		log.Println("Cron: feed fetch completed")
	})
	if err != nil {
		log.Printf("Failed to add fetch cron job: %v", err)
	}

	briefingCron := config.Cfg.BriefingCronTime
	if briefingCron == "" {
		briefingCron = "0 8 * * *"
	}
	_, err = scheduler.AddFunc(briefingCron, func() {
		log.Println("Cron: daily briefing started")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		generateAndSendBriefings(ctx, loc)
		log.Println("Cron: daily briefing completed")
	})
	if err != nil {
		log.Printf("Failed to add briefing cron job: %v", err)
	}

	scheduler.Start()
	log.Println("Cron scheduler started (Beijing time)")
}

func generateAndSendBriefings(ctx context.Context, loc *time.Location) {
	// Query all users with API key + email
	rows, err := db.Pool.Query(ctx,
		`SELECT user_id, api_key, api_base, model, email FROM superread.user_settings WHERE api_key != ''`)
	if err != nil {
		log.Printf("Briefing: failed to query users: %v", err)
		return
	}

	var users []model.UserSettings
	for rows.Next() {
		var s model.UserSettings
		if err := rows.Scan(&s.UserID, &s.APIKey, &s.APIBase, &s.Model, &s.Email); err != nil {
			log.Printf("Briefing: failed to scan user: %v", err)
			continue
		}
		users = append(users, s)
	}
	rows.Close()

	if len(users) == 0 {
		return
	}

	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	since := start.Add(-24 * time.Hour) // 24h window for unsummarized articles

	var wg sync.WaitGroup
	for _, settings := range users {
		wg.Add(1)
		go func(s model.UserSettings) {
			defer wg.Done()
			processUserBriefing(ctx, s, loc, start, end, since)
		}(settings)
	}
	wg.Wait()
}

func processUserBriefing(ctx context.Context, settings model.UserSettings, loc *time.Location, start, end, since time.Time) {
	s := summarizer.New()

	// Get unsummarized articles from the time window
	articles, err := db.ListUnsummarizedArticles(ctx, settings.UserID, since)
	if err != nil {
		log.Printf("Briefing: failed to list unsummarized articles for user %d: %v", settings.UserID, err)
		return
	}

	if len(articles) > 0 {
		// Semaphore: max 3 concurrent summarizations
		sem := make(chan struct{}, 3)
		var sumWg sync.WaitGroup

		for i := range articles {
			sumWg.Add(1)
			go func(article *model.Article) {
				defer sumWg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				// Per-article timeout: 60s
				sumCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
				defer cancel()

				summary, err := s.Summarize(sumCtx, &settings, article)
				if err != nil {
					log.Printf("Briefing: summarize article %d failed: %v", article.ID, err)
					return
				}
				if summary != "" {
					if err := db.UpdateArticleSummary(ctx, article.ID, summary); err != nil {
						log.Printf("Briefing: update article %d summary failed: %v", article.ID, err)
					}
				}
			}(&articles[i])
		}
		sumWg.Wait()
	}

	// Get summarized articles for today's briefing
	summarized, err := db.ListSummarizedArticles(ctx, settings.UserID, start, end)
	if err != nil {
		log.Printf("Briefing: failed to list summarized articles for user %d: %v", settings.UserID, err)
		return
	}

	if len(summarized) == 0 {
		return
	}

	// Send email if SMTP configured
	if config.Cfg.SMTPHost != "" {
		// Use user email, fallback to DefaultToEmail
		to := settings.Email
		if to == "" {
			to = config.Cfg.DefaultToEmail
		}
		if to == "" {
			return
		}

		briefingArticles := make([]mailer.BriefingArticle, 0, len(summarized))
		for _, a := range summarized {
			briefingArticles = append(briefingArticles, mailer.BriefingArticle{
				Title:     a.Title,
				FeedTitle: a.FeedTitle,
				Summary:   a.Summary,
				URL:       a.URL,
			})
		}

		htmlBody := renderBriefingHTML(start.Format("2006-01-02"), briefingArticles)
		subject := fmt.Sprintf("SuperRead Daily Brief - %s", start.Format("2006-01-02"))
		if err := mailer.SendBriefing(to, subject, htmlBody); err != nil {
			log.Printf("Briefing: failed to send email to %s: %v", to, err)
		} else {
			log.Printf("Briefing: sent %d articles to %s", len(summarized), to)
		}
	}

	log.Printf("Briefing: processed user %d, %d summarized articles", settings.UserID, len(summarized))
}

func renderBriefingHTML(date string, articles []mailer.BriefingArticle) string {
	html := fmt.Sprintf(`<html><body style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
<h2>SuperRead Daily Brief - %s</h2>`, date)

	for _, a := range articles {
		html += fmt.Sprintf(`<div style="margin-bottom: 20px; padding: 15px; border: 1px solid #ddd; border-radius: 8px;">
<h3 style="margin: 0 0 5px 0;"><a href="%s" style="text-decoration: none; color: #333;">%s</a></h3>
<p style="color: #888; font-size: 12px; margin: 0 0 10px 0;">%s</p>
<p style="color: #555; font-size: 14px; line-height: 1.6;">%s</p>
</div>`, a.URL, a.Title, a.FeedTitle, a.Summary)
	}

	html += "</body></html>"
	return html
}

func Stop() {
	if scheduler != nil {
		scheduler.Stop()
		log.Println("Cron scheduler stopped")
	}
}