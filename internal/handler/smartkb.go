package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/db"
	"github.com/XiaoleC05/SuperRead/internal/ingester"
	"github.com/gin-gonic/gin"
)

// SmartKBIngest POST /api/smartkb/ingest
func SmartKBIngest(c *gin.Context) {
	start := time.Now()

	files, err := ingester.ScanFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed: " + err.Error()})
		return
	}

	docCount := 0
	chunkCount := 0

	for _, filePath := range files {
		lastIngest, _ := db.GetDocumentIngestTime(c.Request.Context(), filePath)
		if lastIngest > 0 && !ingester.FileModifiedSince(filePath, lastIngest) {
			continue
		}

		chunks, err := ingester.ChunkFile(filePath)
		if err != nil {
			log.Printf("SmartKB: chunk %s failed: %v", filePath, err)
			continue
		}
		if len(chunks) == 0 {
			continue
		}

		title := filepath.Base(filePath)
		docID, err := db.CreateOrGetDocument(c.Request.Context(), filePath, title)
		if err != nil {
			log.Printf("SmartKB: create document %s failed: %v", filePath, err)
			continue
		}

		if err := db.DeleteChunksForDocument(c.Request.Context(), docID); err != nil {
			log.Printf("SmartKB: delete old chunks for %s failed: %v", filePath, err)
		}

		if err := db.InsertChunks(c.Request.Context(), docID, chunks); err != nil {
			log.Printf("SmartKB: insert chunks for %s failed: %v", filePath, err)
			continue
		}

		docCount++
		chunkCount += len(chunks)
	}

	duration := time.Since(start)
	c.JSON(http.StatusOK, gin.H{
		"documents": docCount,
		"chunks":    chunkCount,
		"duration":  duration.String(),
	})
}

// SmartKBSearch POST /api/smartkb/search
func SmartKBSearch(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := 5
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	results, err := db.SearchChunks(c.Request.Context(), req.Query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   req.Query,
		"results": results,
		"total":   len(results),
	})
}

// SmartKBStatus GET /api/smartkb/status
func SmartKBStatus(c *gin.Context) {
	stats, err := db.GetKBStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// SmartKBChat POST /api/smartkb/chat - SSE streaming RAG answer
func SmartKBChat(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Full-text search top-5 chunks
	results, err := db.SearchChunks(c.Request.Context(), req.Query, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed: " + err.Error()})
		return
	}

	// 2. Build RAG prompt with context
	var contextBuilder strings.Builder
	sources := make([]map[string]interface{}, 0, len(results))
	for i, r := range results {
		contextBuilder.WriteString(fmt.Sprintf("[%d] %s (line %d):\n%s\n\n",
			i+1, r.Source, r.SourceLine, r.Content))
		sources = append(sources, map[string]interface{}{
			"index":  i + 1,
			"title":  r.Source,
			"source": r.Source,
			"line":   r.SourceLine,
			"score":  r.Score,
		})
	}

	systemPrompt := "You are the Oxelia51 project knowledge assistant. " +
		"Answer the question based on the following references. " +
		"Use [1] [2] etc. to cite sources in your answer. " +
		"If the references do not contain relevant info, say so.\n\n" +
		contextBuilder.String()

	// 3. Get LLM config
	apiKey := os.Getenv("SMARTKB_EMBEDDING_API_KEY")
	apiBase := os.Getenv("SMARTKB_EMBEDDING_API_BASE")
	chatModel := os.Getenv("SMARTKB_CHAT_MODEL")
	if chatModel == "" {
		chatModel = "gpt-4o-mini"
	}

	if apiKey == "" || apiBase == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "LLM API not configured"})
		return
	}

	// 4. Build chat request with stream
	reqBody := map[string]interface{}{
		"model": chatModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": req.Query},
		},
		"stream": true,
	}
	jsonData, _ := json.Marshal(reqBody)

	apiURL := strings.TrimSuffix(apiBase, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", apiURL, bytes.NewReader(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create request failed"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM error: " + errResp.Error.Message})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("LLM returned status %d", resp.StatusCode)})
		}
		return
	}

	// 5. SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// 6. Stream LLM response to client
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			sseData, _ := json.Marshal(map[string]string{
				"type":    "content",
				"content": chunk.Choices[0].Delta.Content,
			})
			fmt.Fprintf(c.Writer, "data: %s\n\n", sseData)
			c.Writer.Flush()
		}
	}

	// 7. Send sources
	sourcesData, _ := json.Marshal(map[string]interface{}{
		"type":    "sources",
		"sources": sources,
	})
	fmt.Fprintf(c.Writer, "data: %s\n\n", sourcesData)
	c.Writer.Flush()

	// 8. Send done
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}