ALTER TABLE urls DROP COLUMN visitor_uuid;
CREATE UNIQUE INDEX idx_urls_url ON urls (url);