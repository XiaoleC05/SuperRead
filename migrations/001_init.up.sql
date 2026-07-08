CREATE SCHEMA IF NOT EXISTS superread;

CREATE TABLE IF NOT EXISTS superread.feeds (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    feed_url TEXT NOT NULL,
    site_url TEXT DEFAULT '',
    last_fetched_at TIMESTAMPTZ,
    fetch_error TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, feed_url)
);

CREATE TABLE IF NOT EXISTS superread.articles (
    id BIGSERIAL PRIMARY KEY,
    feed_id BIGINT NOT NULL REFERENCES superread.feeds(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    author TEXT DEFAULT '',
    published_at TIMESTAMPTZ,
    content_text TEXT DEFAULT '',
    summary TEXT DEFAULT '',
    is_read BOOLEAN DEFAULT FALSE,
    is_starred BOOLEAN DEFAULT FALSE,
    tag TEXT DEFAULT '',
    guid TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(feed_id, guid)
);

CREATE TABLE IF NOT EXISTS superread.user_settings (
    user_id BIGINT PRIMARY KEY,
    api_key TEXT DEFAULT '',
    api_base TEXT DEFAULT '',
    model TEXT DEFAULT 'gpt-4o-mini',
    fetch_interval_min INT DEFAULT 30,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
