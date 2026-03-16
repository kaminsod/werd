-- +goose Up
-- +goose NO TRANSACTION
ALTER TYPE monitor_type ADD VALUE IF NOT EXISTS 'bluesky';
ALTER TYPE notification_source_type ADD VALUE IF NOT EXISTS 'bluesky';

-- +goose Down
-- PostgreSQL does not support removing enum values. No-op.
