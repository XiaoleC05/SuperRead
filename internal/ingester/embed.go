package ingester

import "fmt"

// Embed is deprecated 鈥?SmartKB now uses PostgreSQL full-text search.
// Returns an error to indicate embedding is no longer supported.
func Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("embedding is deprecated, use full-text search instead")
}

// EmbedBatch is deprecated 鈥?SmartKB now uses PostgreSQL full-text search.
func EmbedBatch(texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("embedding is deprecated, use full-text search instead")
}

// VectorString is deprecated 鈥?kept for backward compatibility.
func VectorString(emb []float32) string {
	return "[]"
}