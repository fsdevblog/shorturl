ALTER TABLE urls ADD COLUMN visitor_uuid VARCHAR(36) NOT NULL DEFAULT '';
DROP INDEX idx_urls_url;
CREATE UNIQUE INDEX idx_visitor_uuid_url ON urls(visitor_uuid, url);


