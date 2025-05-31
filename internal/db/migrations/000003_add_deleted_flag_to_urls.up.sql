ALTER TABLE urls ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
CREATE INDEX idx_deleted ON urls(visitor_uuid, deleted_at)