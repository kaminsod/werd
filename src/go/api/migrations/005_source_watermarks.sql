-- +goose Up

ALTER TABLE monitor_sources ADD COLUMN watermark JSONB NOT NULL DEFAULT '{}';
ALTER TABLE monitor_sources ADD COLUMN last_poll_at TIMESTAMPTZ;

-- +goose Down

ALTER TABLE monitor_sources DROP COLUMN IF EXISTS last_poll_at;
ALTER TABLE monitor_sources DROP COLUMN IF EXISTS watermark;
