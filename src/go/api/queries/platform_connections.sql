-- name: CreatePlatformConnection :one
INSERT INTO platform_connections (project_id, platform, credentials, enabled)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, platform, credentials, enabled, created_at, updated_at;

-- name: ListPlatformConnections :many
SELECT id, project_id, platform, credentials, enabled, created_at, updated_at
FROM platform_connections
WHERE project_id = $1
ORDER BY created_at;

-- name: GetPlatformConnectionByID :one
SELECT id, project_id, platform, credentials, enabled, created_at, updated_at
FROM platform_connections
WHERE id = $1 AND project_id = $2;

-- name: GetPlatformConnectionByPlatform :one
SELECT id, project_id, platform, credentials, enabled, created_at, updated_at
FROM platform_connections
WHERE project_id = $1 AND platform = $2 AND enabled = true;

-- name: UpdatePlatformConnection :one
UPDATE platform_connections
SET platform = $3, credentials = $4, enabled = $5
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, platform, credentials, enabled, created_at, updated_at;

-- name: DeletePlatformConnection :exec
DELETE FROM platform_connections
WHERE id = $1 AND project_id = $2;
