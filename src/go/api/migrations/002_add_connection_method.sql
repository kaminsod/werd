-- +goose Up

ALTER TABLE platform_connections ADD COLUMN method TEXT NOT NULL DEFAULT 'api';

-- Allow one API + one browser connection per platform per project.
CREATE UNIQUE INDEX idx_platform_connections_project_platform_method
    ON platform_connections (project_id, platform, method);

-- +goose Down

DROP INDEX IF EXISTS idx_platform_connections_project_platform_method;
ALTER TABLE platform_connections DROP COLUMN IF EXISTS method;
