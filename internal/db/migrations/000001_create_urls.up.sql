CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    url VARCHAR(512) NOT NULL,
    short_identifier VARCHAR(8) NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_url ON urls (url);