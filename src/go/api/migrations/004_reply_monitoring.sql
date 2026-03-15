-- +goose Up

-- Add reply monitoring fields to post_platform_results.
ALTER TABLE post_platform_results ADD COLUMN monitor_replies BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE post_platform_results ADD COLUMN last_reply_check TIMESTAMPTZ;
ALTER TABLE post_platform_results ADD COLUMN last_known_reply_id TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_post_platform_results_monitor
    ON post_platform_results (monitor_replies, last_reply_check)
    WHERE monitor_replies = true;

-- +goose Down

DROP INDEX IF EXISTS idx_post_platform_results_monitor;
ALTER TABLE post_platform_results DROP COLUMN IF EXISTS last_known_reply_id;
ALTER TABLE post_platform_results DROP COLUMN IF EXISTS last_reply_check;
ALTER TABLE post_platform_results DROP COLUMN IF EXISTS monitor_replies;
