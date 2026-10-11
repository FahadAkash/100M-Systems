-- ==============================================================================
-- Schema for URL Shortener
-- ==============================================================================

CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(16) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for ultra-fast lookup by short code
CREATE INDEX IF NOT EXISTS idx_urls_code ON urls (code);