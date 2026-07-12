package cron

import (
	"context"
	"fmt"
	"log"
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

	// Feed fetch (configurable interval, default 30m)
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

	// Daily briefing (configurable, default 8:00 Beijing time)
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
	// Get all users with API key configured
	rows, err := db.Pool.Query(ctx,
		`SELECT user_id, api_key, api_base, model FROM superread.user_settings WHERE api_key != ''`)
	if err != nil {
		log.Printf("Briefing: failed to query users: %v", err)
		return
	}
	defer rows.Close()

	s := summarizer.New()
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	for rows.Next() {
		var settings model.UserSettings
		if err := rows.Scan(&settings.UserID, &settings.APIKey, &settings.APIBase, &settings.Model); err != nil {
			log.Printf("Briefing: failed to scan user: %v", err)
			continue
		}

		// Get recent articles and summarize unsummarized ones
		articles, err := db.ListArticles(ctx, settings.UserID, nil, nil, nil, 50)
		if err != nil {
			log.Printf("Briefing: failed to list articles for user %d: %v", settings.UserID, err)
			continue
		}

		for i := range articles {
			if articles[i].Summary != "" {
				continue
			}
			summary, err := s.Summarize(ctx, &settings, &articles[i])
			if err != nil {
				log.Printf("Briefing: summarize article %d failed: %v", articles[i].ID, err)
				continue
			}
			if summary != "" {
				if err := db.UpdateArticleSummary(ctx, articles[i].ID, summary); err != nil {
					log.Printf("Briefing: update article %d summary failed: %v", articles[i].ID, err)
				}
			}
		}

		// Get summarized articles for today's briefing
		summarized, err := db.ListSummarizedArticles(ctx, settings.UserID, start, end)
		if err != nil {
			log.Printf("Briefing: failed to list summarized articles for user %d: %v", settings.UserID, err)
			continue
		}

		if len(summarized) == 0 {
			continue
		}

		// Send email if SMTP configured
		if config.Cfg.SMTPHost != "" && config.Cfg.DefaultToEmail != "" {
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
			if err := mailer.SendBriefing(config.Cfg.DefaultToEmail, subject, htmlBody); err != nil {
				log.Printf("Briefing: failed to send email for user %d: %v", settings.UserID, err)
			} else {
				log.Printf("Briefing: sent %d articles to %s", len(summarized), config.Cfg.DefaultToEmail)
			}
		}

		log.Printf("Briefing: processed user %d, %d summarized articles", settings.UserID, len(summarized))
	}
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