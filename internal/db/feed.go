package db

import (
	"context"
	"fmt"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/jackc/pgx/v5"
)

func ListFeeds(ctx context.Context, userID int64) ([]model.Feed, error) {
	query := `
		SELECT id, user_id, title, feed_url, site_url, last_fetched_at, 
		       fetch_error, created_at, updated_at
		FROM superread.feeds
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []model.Feed
	for rows.Next() {
		var f model.Feed
		err := rows.Scan(
			&f.ID, &f.UserID, &f.Title, &f.FeedURL, &f.SiteURL,
			&f.LastFetchedAt, &f.FetchError, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan feed: %w", err)
		}
		feeds = append(feeds, f)
	}

	return feeds, rows.Err()
}

func GetFeed(ctx context.Context, id int64) (*model.Feed, error) {
	query := `
		SELECT id, user_id, title, feed_url, site_url, last_fetched_at,
		       fetch_error, created_at, updated_at
		FROM superread.feeds
		WHERE id = $1
	`
	var f model.Feed
	err := Pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.UserID, &f.Title, &f.FeedURL, &f.SiteURL,
		&f.LastFetchedAt, &f.FetchError, &f.CreatedAt, &f.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get feed: %w", err)
	}
	return &f, nil
}

func CreateFeed(ctx context.Context, userID int64, req model.CreateFeedRequest) (*model.Feed, error) {
	query := `
		INSERT INTO superread.feeds (user_id, title, feed_url, site_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, feed_url) DO NOTHING
		RETURNING id, user_id, title, feed_url, site_url, last_fetched_at,
		          fetch_error, created_at, updated_at
	`
	var f model.Feed
	err := Pool.QueryRow(ctx, query, userID, req.Title, req.FeedURL, req.SiteURL).Scan(
		&f.ID, &f.UserID, &f.Title, &f.FeedURL, &f.SiteURL,
		&f.LastFetchedAt, &f.FetchError, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create feed: %w", err)
	}
	return &f, nil
}

func DeleteFeed(ctx context.Context, id int64, userID int64) error {
	query := `DELETE FROM superread.feeds WHERE id = $1 AND user_id = $2`
	_, err := Pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	return nil
}

func UpdateFeedFetchTime(ctx context.Context, id int64, fetchErr string) error {
	now := time.Now()
	query := `
		UPDATE superread.feeds
		SET last_fetched_at = $2, fetch_error = $3, updated_at = $2
		WHERE id = $1
	`
	_, err := Pool.Exec(ctx, query, id, now, fetchErr)
	if err != nil {
		return fmt.Errorf("update feed fetch time: %w", err)
	}
	return nil
}

func ListAllFeeds(ctx context.Context) ([]model.Feed, error) {
	query := `
		SELECT id, user_id, title, feed_url, site_url, last_fetched_at,
		       fetch_error, created_at, updated_at
		FROM superread.feeds
		ORDER BY user_id, created_at
	`
	rows, err := Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all feeds: %w", err)
	}
	defer rows.Close()

	var feeds []model.Feed
	for rows.Next() {
		var f model.Feed
		err := rows.Scan(
			&f.ID, &f.UserID, &f.Title, &f.FeedURL, &f.SiteURL,
			&f.LastFetchedAt, &f.FetchError, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan feed: %w", err)
		}
		feeds = append(feeds, f)
	}

	return feeds, rows.Err()
}
