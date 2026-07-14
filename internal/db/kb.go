package db

import (
	"context"
	"fmt"
	"time"

	"github.com/XiaoleC05/SuperRead/internal/ingester"
)

// CreateOrGetDocument returns existing document ID or creates a new one.
func CreateOrGetDocument(ctx context.Context, source, title string) (int64, error) {
	var id int64
	err := Pool.QueryRow(ctx,
		`INSERT INTO smartkb.documents (source, title)
		 VALUES ($1, $2)
		 ON CONFLICT (source) DO UPDATE SET title = EXCLUDED.title, ingested_at = NOW()
		 RETURNING id`,
		source, title,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create document: %w", err)
	}
	return id, nil
}

// DeleteChunksForDocument removes all chunks for a document (for re-ingestion).
func DeleteChunksForDocument(ctx context.Context, docID int64) error {
	_, err := Pool.Exec(ctx,
		`DELETE FROM smartkb.chunks WHERE document_id = $1`, docID)
	return err
}

// InsertChunks inserts multiple chunks with auto-generated tsvector for full-text search.
func InsertChunks(ctx context.Context, docID int64, chunks []ingester.Chunk) error {
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, chunk := range chunks {
		_, err := tx.Exec(ctx,
			`INSERT INTO smartkb.chunks (document_id, content, source_line, tsv)
			 VALUES ($1, $2, $3, to_tsvector('simple', $2))`,
			docID, chunk.Content, chunk.SourceLine,
		)
		if err != nil {
			return fmt.Errorf("insert chunk %d: %w", i, err)
		}
	}

	return tx.Commit(ctx)
}

// ChunkResult represents a search hit.
type ChunkResult struct {
	ID         int64   `json:"id"`
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	SourceLine int     `json:"source_line"`
	Score      float64 `json:"score"`
}

// SearchChunks performs PostgreSQL full-text search using plainto_tsquery and ts_rank.
func SearchChunks(ctx context.Context, query string, limit int) ([]ChunkResult, error) {
	rows, err := Pool.Query(ctx, `
		SELECT c.id, c.content, d.source, c.source_line,
		       ts_rank(c.tsv, plainto_tsquery('simple', $1)) as score
		FROM smartkb.chunks c
		JOIN smartkb.documents d ON c.document_id = d.id
		WHERE c.tsv @@ plainto_tsquery('simple', $1)
		ORDER BY score DESC
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()

	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		if err := rows.Scan(&r.ID, &r.Content, &r.Source, &r.SourceLine, &r.Score); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// KBStats holds knowledge base statistics.
type KBStats struct {
	Documents  int    `json:"documents"`
	Chunks     int    `json:"chunks"`
	LastIngest string `json:"last_ingest"`
}

// GetKBStats returns document count, chunk count, and last ingest time.
func GetKBStats(ctx context.Context) (*KBStats, error) {
	var stats KBStats
	var lastIngest *time.Time

	err := Pool.QueryRow(ctx,
		`SELECT
			(SELECT COUNT(*) FROM smartkb.documents),
			(SELECT COUNT(*) FROM smartkb.chunks),
			(SELECT MAX(ingested_at) FROM smartkb.documents)`,
	).Scan(&stats.Documents, &stats.Chunks, &lastIngest)

	if lastIngest != nil {
		stats.LastIngest = lastIngest.Format(time.RFC3339)
	}

	if err != nil {
		return nil, fmt.Errorf("get kb stats: %w", err)
	}
	return &stats, nil
}

// GetDocumentIngestTime returns the last ingest epoch timestamp for a source file.
func GetDocumentIngestTime(ctx context.Context, source string) (int64, error) {
	var epoch int64
	err := Pool.QueryRow(ctx,
		`SELECT EXTRACT(EPOCH FROM ingested_at)::bigint FROM smartkb.documents WHERE source = $1`,
		source,
	).Scan(&epoch)
	if err != nil {
		return 0, nil
	}
	return epoch, nil
}