package db

import (
	"context"
	"fmt"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/jackc/pgx/v5"
)

func ListArticles(ctx context.Context, userID int64, feedID *int64, starred *bool, tag *string, limit int) ([]model.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.url, a.author, a.published_at,
		       a.content_text, a.summary, a.is_read, a.is_starred, a.tag,
		       a.guid, a.created_at
		FROM superread.articles a
		JOIN superread.feeds f ON a.feed_id = f.id
		WHERE f.user_id = $1
	`
	args := []interface{}{userID}
	argIdx := 2

	if feedID != nil {
		query += fmt.Sprintf(" AND a.feed_id = $%d", argIdx)
		args = append(args, *feedID)
		argIdx++
	}

	if starred != nil {
		query += fmt.Sprintf(" AND a.is_starred = $%d", argIdx)
		args = append(args, *starred)
		argIdx++
	}

	if tag != nil && *tag != "" {
		query += fmt.Sprintf(" AND a.tag = $%d", argIdx)
		args = append(args, *tag)
		argIdx++
	}

	query += " ORDER BY a.published_at DESC, a.created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
	}

	rows, err := Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query articles: %w", err)
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		var a model.Article
		err := rows.Scan(
			&a.ID, &a.FeedID, &a.Title, &a.URL, &a.Author, &a.PublishedAt,
			&a.ContentText, &a.Summary, &a.IsRead, &a.IsStarred, &a.Tag,
			&a.GUID, &a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, a)
	}

	return articles, rows.Err()
}

func CreateArticle(ctx context.Context, article *model.Article) error {
	query := `
		INSERT INTO superread.articles 
		(feed_id, title, url, author, published_at, content_text, summary, guid)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (feed_id, guid) DO NOTHING
	`
	_, err := Pool.Exec(ctx, query,
		article.FeedID, article.Title, article.URL, article.Author,
		article.PublishedAt, article.ContentText, article.Summary, article.GUID,
	)
	if err != nil {
		return fmt.Errorf("create article: %w", err)
	}
	return nil
}

func UpdateArticle(ctx context.Context, id int64, userID int64, req model.UpdateArticleRequest) (*model.Article, error) {
	query := `
		UPDATE superread.articles a
		SET
	`
	updates := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.IsRead != nil {
		updates = append(updates, fmt.Sprintf("is_read = $%d", argIdx))
		args = append(args, *req.IsRead)
		argIdx++
	}

	if req.IsStarred != nil {
		updates = append(updates, fmt.Sprintf("is_starred = $%d", argIdx))
		args = append(args, *req.IsStarred)
		argIdx++
	}

	if req.Tag != nil {
		updates = append(updates, fmt.Sprintf("tag = $%d", argIdx))
		args = append(args, *req.Tag)
		argIdx++
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	query += fmt.Sprintf("%s WHERE a.id = $%d", joinStrings(updates, ", "), argIdx)
	args = append(args, id)
	argIdx++

	query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM superread.feeds f WHERE f.id = a.feed_id AND f.user_id = $%d)", argIdx)
	args = append(args, userID)

	_, err := Pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update article: %w", err)
	}

	// Fetch updated article
	return GetArticle(ctx, id)
}

func GetArticle(ctx context.Context, id int64) (*model.Article, error) {
	query := `
		SELECT id, feed_id, title, url, author, published_at,
		       content_text, summary, is_read, is_starred, tag, guid, created_at
		FROM superread.articles
		WHERE id = $1
	`
	var a model.Article
	err := Pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.FeedID, &a.Title, &a.URL, &a.Author, &a.PublishedAt,
		&a.ContentText, &a.Summary, &a.IsRead, &a.IsStarred, &a.Tag,
		&a.GUID, &a.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get article: %w", err)
	}
	return &a, nil
}

func ListArticlesByDateRange(ctx context.Context, userID int64, start, end time.Time) ([]model.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.url, a.author, a.published_at,
		       a.content_text, a.summary, a.is_read, a.is_starred, a.tag,
		       a.guid, a.created_at
		FROM superread.articles a
		JOIN superread.feeds f ON a.feed_id = f.id
		WHERE f.user_id = $1
		  AND a.published_at >= $2
		  AND a.published_at < $3
		ORDER BY a.published_at DESC
	`
	rows, err := Pool.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query articles by date: %w", err)
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		var a model.Article
		err := rows.Scan(
			&a.ID, &a.FeedID, &a.Title, &a.URL, &a.Author, &a.PublishedAt,
			&a.ContentText, &a.Summary, &a.IsRead, &a.IsStarred, &a.Tag,
			&a.GUID, &a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, a)
	}

	return articles, rows.Err()
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
