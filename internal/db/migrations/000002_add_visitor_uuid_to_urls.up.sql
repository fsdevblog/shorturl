ALTER TABLE urls ADD COLUMN visitor_uuid VARCHAR(36) DEFAULT NULL;
CREATE INDEX idx_visitor_uuid ON urls(visitor_uuid);
