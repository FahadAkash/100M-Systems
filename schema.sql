-- ==============================================================================
-- Schema for URL Shortener
-- ==============================================================================

CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

-- Index for ultra-fast lookup by short code
CREATE INDEX IF NOT EXISTS idx_urls_code ON urls (code);
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls (expires_at);

-- Click events table for stream processing analytics
CREATE TABLE IF NOT EXISTS url_clicks (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(16) NOT NULL,
    original_url TEXT NOT NULL,
    ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    referer TEXT NOT NULL DEFAULT '',
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for lightning-fast aggregated and recent clicks querying
CREATE INDEX IF NOT EXISTS idx_url_clicks_code ON url_clicks (code);
CREATE INDEX IF NOT EXISTS idx_url_clicks_clicked_at ON url_clicks (clicked_at);