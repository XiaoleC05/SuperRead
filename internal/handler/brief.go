package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/llm"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/gin-gonic/gin"
)

type BriefArticle struct {
	ID        int64  `json:"id"`
	FeedID    int64  `json:"feed_id"`
	FeedTitle string `json:"feed_title"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Author    string `json:"author"`
	Summary   string `json:"summary"`
	Published string `json:"published"`
}

// GetDailyBrief GET /api/daily-brief?date=2026-07-13
func GetDailyBrief(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().In(loc).Format("2006-01-02")
	}

	// Try to load existing brief from daily_briefs table
	var (
		content    string
		articleIDs []int64
	)
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT content, article_ids FROM superread.daily_briefs WHERE user_id = $1 AND date = $2`,
		userID, dateStr,
	).Scan(&content, &articleIDs)

	if err == nil && content != "" {
		articles := fetchBriefArticles(c, userID, articleIDs)
		c.JSON(http.StatusOK, gin.H{
			"date":     dateStr,
			"content":  content,
			"articles": articles,
			"total":    len(articles),
		})
		return
	}

	// No brief for this date 鈥?fallback to recent summarized articles
	articles, err := db.ListRecentSummarizedArticles(c.Request.Context(), userID, 30)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	brief := make([]BriefArticle, 0, len(articles))
	for _, a := range articles {
		published := ""
		if a.PublishedAt != nil {
			published = a.PublishedAt.Format("2006-01-02 15:04")
		}
		brief = append(brief, BriefArticle{
			ID:        a.ID,
			FeedID:    a.FeedID,
			FeedTitle: a.FeedTitle,
			Title:     a.Title,
			URL:       a.URL,
			Author:    a.Author,
			Summary:   a.Summary,
			Published: published,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"date":     dateStr,
		"content":  "",
		"articles": brief,
		"total":    len(brief),
	})
}

// GenerateDailyBrief POST /api/daily-brief/generate
func GenerateDailyBrief(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	settings, err := db.GetSettings(c.Request.Context(), userID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if settings == nil || settings.APIKey == "" || settings.APIBase == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API key not configured"})
		return
	}

	articles, err := db.ListRecentSummarizedArticles(c.Request.Context(), userID, 30)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if len(articles) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"date":    time.Now().Format("2006-01-02"),
			"content": "",
			"total":   0,
		})
		return
	}

	// Build prompt from article summaries
	var sb strings.Builder
	for _, a := range articles {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", a.Title, a.Summary))
	}

	prompt := fmt.Sprintf(
		"You are a news briefing editor. Consolidate the following article summaries into a coherent daily brief. Group by theme, remove duplicates, write in fluent Chinese. Output plain text only, no more than 1000 characters.\n\nArticles:\n%s",
		sb.String(),
	)

	content, err := callLLM(c.Request.Context(), settings, prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM call failed: " + err.Error()})
		return
	}

	// Collect article IDs
	articleIDs := make([]int64, 0, len(articles))
	for _, a := range articles {
		articleIDs = append(articleIDs, a.ID)
	}

	// Store in daily_briefs (upsert)
	loc, _ := time.LoadLocation("Asia/Shanghai")
	dateStr := time.Now().In(loc).Format("2006-01-02")

	_, err = db.Pool.Exec(c.Request.Context(),
		`INSERT INTO superread.daily_briefs (user_id, date, content, article_ids)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, date) DO UPDATE SET
			content = EXCLUDED.content,
			article_ids = EXCLUDED.article_ids,
			created_at = NOW()`,
		userID, dateStr, content, articleIDs,
	)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	// Build article list for response
	brief := make([]BriefArticle, 0, len(articles))
	for _, a := range articles {
		published := ""
		if a.PublishedAt != nil {
			published = a.PublishedAt.Format("2006-01-02 15:04")
		}
		brief = append(brief, BriefArticle{
			ID:        a.ID,
			FeedID:    a.FeedID,
			FeedTitle: a.FeedTitle,
			Title:     a.Title,
			URL:       a.URL,
			Author:    a.Author,
			Summary:   a.Summary,
			Published: published,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"date":     dateStr,
		"content":  content,
		"articles": brief,
		"total":    len(brief),
	})
}

func fetchBriefArticles(c *gin.Context, userID int64, articleIDs []int64) []BriefArticle {
	if len(articleIDs) == 0 {
		return []BriefArticle{}
	}

	ids := make([]string, len(articleIDs))
	for i, id := range articleIDs {
		ids[i] = strconv.FormatInt(id, 10)
	}
	query := fmt.Sprintf(
		`SELECT a.id, a.feed_id, a.title, a.url, a.author, a.published_at,
		        a.summary, f.title as feed_title
		 FROM superread.articles a
		 JOIN superread.feeds f ON a.feed_id = f.id
		 WHERE f.user_id = $1 AND a.id IN (%s)
		 ORDER BY a.published_at DESC`,
		strings.Join(ids, ","),
	)

	rows, err := db.Pool.Query(c.Request.Context(), query, userID)
	if err != nil {
		return []BriefArticle{}
	}
	defer rows.Close()

	var result []BriefArticle
	for rows.Next() {
		var a model.Article
		if err := rows.Scan(&a.ID, &a.FeedID, &a.Title, &a.URL, &a.Author, &a.PublishedAt, &a.Summary, &a.FeedTitle); err != nil {
			continue
		}
		published := ""
		if a.PublishedAt != nil {
			published = a.PublishedAt.Format("2006-01-02 15:04")
		}
		result = append(result, BriefArticle{
			ID:        a.ID,
			FeedID:    a.FeedID,
			FeedTitle: a.FeedTitle,
			Title:     a.Title,
			URL:       a.URL,
			Author:    a.Author,
			Summary:   a.Summary,
			Published: published,
		})
	}
	return result
}

func callLLM(ctx context.Context, settings *model.UserSettings, prompt string) (string, error) {
	if err := llm.ValidateAPIBase(settings.APIBase); err != nil {
		return "", fmt.Errorf("invalid api base: %w", err)
	}

	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type chatReq struct {
		Model    string    `json:"model"`
		Messages []chatMsg `json:"messages"`
	}
	type chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	reqBody := chatReq{
		Model: settings.Model,
		Messages: []chatMsg{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	apiURL := strings.TrimSuffix(settings.APIBase, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var cr chatResp
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}