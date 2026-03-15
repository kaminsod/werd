-- +goose Up

-- Post type enum.
CREATE TYPE post_type AS ENUM ('text', 'link');

-- Add structured fields to published_posts.
ALTER TABLE published_posts ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE published_posts ADD COLUMN url TEXT NOT NULL DEFAULT '';
ALTER TABLE published_posts ADD COLUMN post_type post_type NOT NULL DEFAULT 'text';

-- Track per-platform publish outcomes (essential for reply monitoring).
CREATE TABLE post_platform_results (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id          UUID NOT NULL REFERENCES published_posts(id) ON DELETE CASCADE,
    platform         TEXT NOT NULL,
    platform_post_id TEXT NOT NULL DEFAULT '',
    platform_url     TEXT NOT NULL DEFAULT '',
    success          BOOLEAN NOT NULL DEFAULT false,
    error_message    TEXT NOT NULL DEFAULT '',
    published_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_post_platform_results_post_id ON post_platform_results (post_id);

-- +goose Down

DROP TABLE IF EXISTS post_platform_results;
ALTER TABLE published_posts DROP COLUMN IF EXISTS post_type;
ALTER TABLE published_posts DROP COLUMN IF EXISTS url;
ALTER TABLE published_posts DROP COLUMN IF EXISTS title;
DROP TYPE IF EXISTS post_type;
