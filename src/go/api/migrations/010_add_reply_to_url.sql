-- +goose Up
ALTER TABLE published_posts ADD COLUMN reply_to_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE published_posts DROP COLUMN IF EXISTS reply_to_url;
