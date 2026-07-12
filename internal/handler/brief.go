package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/llm"
	"github.com/XiaoleC05/SuperRead/internal/model"
	"github.com/XiaoleC05/SuperRead/internal/summarizer"
	"github.com/XiaoleC05/SuperRead/internal/config"
	"github.com/XiaoleC05/SuperRead/internal/mailer"
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

// GenerateDailyBrief POST /api/daily-brief/generate?range=24h
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

	// Parse range (default: from settings or 24h)
	rangeStr := c.Query("range")
	if rangeStr == "" {
		rangeStr = settings.BriefingRange
		if rangeStr == "" {
			rangeStr = "24h"
		}
	}
	duration, err := parseRange(rangeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range: " + err.Error()})
		return
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc)
	since := now.Add(-duration)

	// 1. Get unsummarized articles in the time window
	unsummarized, err := db.ListUnsummarizedArticles(c.Request.Context(), userID, since)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	// 2. Summarize each unsummarized article
	s := summarizer.New()
	for i := range unsummarized {
		summary, err := s.Summarize(c.Request.Context(), settings, &unsummarized[i])
		if err != nil {
			log.Printf("GenerateDailyBrief: summarize article %d failed: %v", unsummarized[i].ID, err)
			continue
		}
		if summary != "" {
			db.UpdateArticleSummary(c.Request.Context(), unsummarized[i].ID, summary)
			unsummarized[i].Summary = summary
		}
	}

	// 3. Get all summarized articles in the time window
	articles, err := db.ListSummarizedArticles(c.Request.Context(), userID, since, now)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if len(articles) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"date":    now.Format("2006-01-02"),
			"content": "",
			"total":   0,
		})
		return
	}

	// 4. Build consolidation prompt
	var sb strings.Builder
	for _, a := range articles {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", a.Title, a.Summary))
	}

	prompt := fmt.Sprintf(
		"Consolidate the following article summaries into a coherent daily brief. Group by theme, remove duplicates, write in fluent Chinese. One sentence per article. Output plain text only, no more than 800 characters.\n\nArticles:\n%s",
		sb.String(),
	)

	content, err := callLLM(c.Request.Context(), settings, prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM call failed: " + err.Error()})
		return
	}

	// 5. Store in daily_briefs
	articleIDs := make([]int64, 0, len(articles))
	for _, a := range articles {
		articleIDs = append(articleIDs, a.ID)
	}

	dateStr := now.Format("2006-01-02")
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

	// 6. Build response
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

func parseRange(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid range format")
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", numStr)
	}
	switch unit {
	case 'h':
		return time.Duration(num) * time.Hour, nil
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid unit: %c (use h or d)", unit)
	}
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
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("LLM error: %s", errResp.Error.Message)
		}
		return "", fmt.Errorf("LLM returned status %d", resp.StatusCode)
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
// SendBriefingToEmail POST /api/daily-brief/send
func SendBriefingToEmail(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	settings, err := db.GetSettings(c.Request.Context(), userID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if settings == nil || settings.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email not configured"})
		return
	}

	if config.Cfg.SMTPHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP not configured"})
		return
	}

	articles, err := db.ListRecentSummarizedArticles(c.Request.Context(), userID, 30)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	if len(articles) == 0 {
		c.JSON(http.StatusOK, gin.H{"sent": false, "message": "no summarized articles, generate summaries first"})
		return
	}

	briefingArticles := make([]mailer.BriefingArticle, 0, len(articles))
	for _, a := range articles {
		briefingArticles = append(briefingArticles, mailer.BriefingArticle{
			Title:     a.Title,
			FeedTitle: a.FeedTitle,
			Summary:   a.Summary,
			URL:       a.URL,
		})
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	dateStr := start.Format("2006-01-02")
	subject := fmt.Sprintf("SuperRead Daily Brief - %s", dateStr)
	htmlBody := renderBriefingHTML(dateStr, briefingArticles)

	if err := mailer.SendBriefing(settings.Email, subject, htmlBody); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "send failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sent": true, "count": len(articles), "to": settings.Email})
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

// ListDailyBriefs GET /api/daily-brief/list?limit=30
func ListDailyBriefs(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		return
	}

	limit := 30
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT date, content, article_ids, created_at
		 FROM superread.daily_briefs
		 WHERE user_id = $1
		 ORDER BY date DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	defer rows.Close()

	type briefItem struct {
		Date         string `json:"date"`
		Content      string `json:"content"`
		Preview      string `json:"preview"`
		ArticleCount int    `json:"article_count"`
		CreatedAt    string `json:"created_at"`
	}

	items := []briefItem{}
	for rows.Next() {
		var (
			dateStr    string
			content    string
			articleIDs []int64
			createdAt  time.Time
		)
		if err := rows.Scan(&dateStr, &content, &articleIDs, &createdAt); err != nil {
			continue
		}

		preview := content
		if len(preview) > 150 {
			preview = preview[:150] + "..."
		}

		items = append(items, briefItem{
			Date:         dateStr,
			Content:      content,
			Preview:      preview,
			ArticleCount: len(articleIDs),
			CreatedAt:    createdAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, items)
}