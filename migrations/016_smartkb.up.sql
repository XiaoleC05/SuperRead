-- 016_smartkb: pgvector extension + documents/chunks tables

CREATE EXTENSION IF NOT EXISTS vector;
CREATE SCHEMA IF NOT EXISTS smartkb;

CREATE TABLE IF NOT EXISTS smartkb.documents (
    id BIGSERIAL PRIMARY KEY,
    source TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    ingested_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS smartkb.chunks (
    id BIGSERIAL PRIMARY KEY,
    document_id BIGINT REFERENCES smartkb.documents(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    embedding vector(1536),
    source_line INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chunks_embedding ON smartkb.chunks
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE INDEX IF NOT EXISTS idx_chunks_document_id ON smartkb.chunks(document_id);