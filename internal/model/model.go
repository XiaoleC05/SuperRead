package model

import "time"

type Feed struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	Title         string     `json:"title"`
	FeedURL       string     `json:"feed_url"`
	SiteURL       string     `json:"site_url"`
	LastFetchedAt *time.Time `json:"last_fetched_at,omitempty"`
	FetchError    string     `json:"fetch_error"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Article struct {
	ID          int64      `json:"id"`
	FeedID      int64      `json:"feed_id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Author      string     `json:"author"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	ContentText string     `json:"content_text"`
	Summary     string     `json:"summary"`
	IsRead      bool       `json:"is_read"`
	IsStarred   bool       `json:"is_starred"`
	Tag         string     `json:"tag"`
	GUID        string     `json:"guid"`
	CreatedAt   time.Time  `json:"created_at"`
	FeedTitle   string     `json:"feed_title"`
}

type UserSettings struct {
	UserID          int64     `json:"user_id"`
	APIKey          string    `json:"api_key"`
	APIBase         string    `json:"api_base"`
	Model           string    `json:"model"`
	FetchIntervalMin int      `json:"fetch_interval_min"`
	Email           string    `json:"email"`
	BriefingRange   string    `json:"briefing_range"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type DailyBrief struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Date       string    `json:"date"`
	Content    string    `json:"content"`
	ArticleIDs []int64   `json:"article_ids"`
	CreatedAt  time.Time `json:"created_at"`
}
// Request/Response DTOs

type CreateFeedRequest struct {
	Title   string `json:"title" binding:"required"`
	FeedURL string `json:"feed_url" binding:"required"`
	SiteURL string `json:"site_url"`
}

type UpdateArticleRequest struct {
	IsRead    *bool   `json:"is_read,omitempty"`
	IsStarred *bool   `json:"is_starred,omitempty"`
	Tag       *string `json:"tag,omitempty"`
}

type UpdateSettingsRequest struct {
	APIKey           *string `json:"api_key,omitempty"`
	APIBase          *string `json:"api_base,omitempty"`
	Model            *string `json:"model,omitempty"`
	FetchIntervalMin *int    `json:"fetch_interval_min,omitempty"`
	Email            *string `json:"email,omitempty"`
	BriefingRange   *string `json:"briefing_range,omitempty"`
}

type OPMLImportResponse struct {
	Imported int      `json:"imported"`
	Errors   []string `json:"errors,omitempty"`
}
