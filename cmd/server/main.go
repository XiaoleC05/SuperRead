package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/config"
	"github.com/XiaoleC05/SuperRead/internal/cron"
	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/fetcher"
	"github.com/XiaoleC05/SuperRead/internal/handler"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func corsOrigins() []string {
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		return strings.Split(v, ",")
	}
	return []string{"http://localhost:5173"}
}

func main() {
	// Load configuration
	config.Load()

	// Initialize database
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	runMigrations()

	// Set up router
	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-User-Id", "X-Username", "X-Role"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Public routes
	r.GET("/api/health", handler.Health)

	// Protected routes
	api := r.Group("/api")
	api.Use(handler.AuthMiddleware())
	{
		// Feeds
		api.GET("/feeds", handler.ListFeeds)
		api.POST("/feeds", handler.CreateFeed)
		api.DELETE("/feeds/:id", handler.DeleteFeed)
		api.POST("/feeds/:id/fetch", handler.FetchFeed)
		api.POST("/feeds/import", handler.ImportOPML)

		// Articles
		api.GET("/articles", handler.ListArticles)
		api.PATCH("/articles/:id", handler.UpdateArticle)

		// Brief
		api.GET("/daily-brief", handler.GetDailyBrief)
		api.GET("/daily-brief/list", handler.ListDailyBriefs)
		api.POST("/daily-brief/generate", handler.GenerateDailyBrief)
		api.POST("/daily-brief/send", handler.SendBriefingToEmail)

		// Settings
		api.GET("/settings", handler.GetSettings)
		api.PUT("/settings", handler.UpdateSettings)

		api.GET("/stats", handler.Stats)

		// Summarize
		api.POST("/summarize", handler.Summarize)
	}

	// Start cron scheduler
	cron.Start()
	defer cron.Stop()

	// Initial fetch on startup
	go func() {
		time.Sleep(5 * time.Second)
		log.Println("Starting initial feed fetch")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		fetcher.FetchAllFeeds(ctx)
		log.Println("Initial feed fetch completed")
	}()

	// Start server
	addr := ":" + config.Cfg.Port
	log.Printf("SuperRead server starting on %s", addr)

	// Graceful shutdown
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func runMigrations() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Read migration file
	migrationSQL := `
		CREATE SCHEMA IF NOT EXISTS superread;

		CREATE TABLE IF NOT EXISTS superread.feeds (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			feed_url TEXT NOT NULL,
			site_url TEXT DEFAULT '',
			last_fetched_at TIMESTAMPTZ,
			fetch_error TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, feed_url)
		);

		CREATE TABLE IF NOT EXISTS superread.articles (
			id BIGSERIAL PRIMARY KEY,
			feed_id BIGINT NOT NULL REFERENCES superread.feeds(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			author TEXT DEFAULT '',
			published_at TIMESTAMPTZ,
			content_text TEXT DEFAULT '',
			summary TEXT DEFAULT '',
			is_read BOOLEAN DEFAULT FALSE,
			is_starred BOOLEAN DEFAULT FALSE,
			tag TEXT DEFAULT '',
			guid TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(feed_id, guid)
		);

		CREATE TABLE IF NOT EXISTS superread.user_settings (
			user_id BIGINT PRIMARY KEY,
			api_key TEXT DEFAULT '',
			api_base TEXT DEFAULT '',
			model TEXT DEFAULT 'gpt-4o-mini',
			fetch_interval_min INT DEFAULT 30,
			email TEXT DEFAULT '',
			briefing_range TEXT DEFAULT '24h',
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		-- Ensure columns exist for databases created before these were added
		ALTER TABLE superread.user_settings ADD COLUMN IF NOT EXISTS email TEXT DEFAULT '';
		ALTER TABLE superread.user_settings ADD COLUMN IF NOT EXISTS briefing_range TEXT DEFAULT '24h';

		CREATE TABLE IF NOT EXISTS superread.daily_briefs (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			date DATE NOT NULL DEFAULT CURRENT_DATE,
			content TEXT NOT NULL DEFAULT '',
			article_ids BIGINT[] DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, date)
		);
	`

	if _, err := db.Pool.Exec(ctx, migrationSQL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database migrations completed")
}
