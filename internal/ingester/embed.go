package ingester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func getEmbeddingConfig() (apiKey, apiBase, model string) {
	apiKey = os.Getenv("SMARTKB_EMBEDDING_API_KEY")
	apiBase = os.Getenv("SMARTKB_EMBEDDING_API_BASE")
	model = os.Getenv("SMARTKB_EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}
	return
}

// Embed returns a single embedding vector for the given text.
func Embed(text string) ([]float32, error) {
	embeddings, err := EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// EmbedBatch sends texts to the embedding API in batches of 100.
func EmbedBatch(texts []string) ([][]float32, error) {
	apiKey, apiBase, model := getEmbeddingConfig()
	if apiKey == "" || apiBase == "" {
		return nil, fmt.Errorf("embedding API not configured (SMARTKB_EMBEDDING_API_KEY / SMARTKB_EMBEDDING_API_BASE)")
	}

	const batchSize = 100
	var allEmbeddings [][]float32
	client := &http.Client{Timeout: 60 * time.Second}

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		reqBody := map[string]interface{}{
			"model": model,
			"input": batch,
		}
		jsonData, _ := json.Marshal(reqBody)

		apiURL := strings.TrimSuffix(apiBase, "/") + "/v1/embeddings"
		req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonData))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("embedding API returned status %d", resp.StatusCode)
		}

		var er embeddingResponse
		if err := json.Unmarshal(body, &er); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		for _, d := range er.Data {
			allEmbeddings = append(allEmbeddings, d.Embedding)
		}
	}

	return allEmbeddings, nil
}

// VectorString converts a float32 slice to pgvector string format: [0.1,0.2,...]
func VectorString(emb []float32) string {
	strs := make([]string, len(emb))
	for i, v := range emb {
		strs[i] = fmt.Sprintf("%v", v)
	}
	return "[" + strings.Join(strs, ",") + "]"
}