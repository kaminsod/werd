-- name: CreatePlatformConnection :one
INSERT INTO platform_connections (project_id, platform, method, credentials, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, project_id, platform, method, credentials, enabled, created_at, updated_at;

-- name: ListPlatformConnections :many
SELECT id, project_id, platform, method, credentials, enabled, created_at, updated_at
FROM platform_connections
WHERE project_id = $1
ORDER BY platform, method, created_at;

-- name: GetPlatformConnectionByID :one
SELECT id, project_id, platform, method, credentials, enabled, created_at, updated_at
FROM platform_connections
WHERE id = $1 AND project_id = $2;

-- name: GetEnabledConnection :one
SELECT id, project_id, platform, method, credentials, enabled, created_at, updated_at
FROM platform_connections
WHERE project_id = $1 AND platform = $2 AND enabled = true
ORDER BY CASE method WHEN 'api' THEN 0 ELSE 1 END
LIMIT 1;

-- name: GetEnabledConnectionByMethod :one
SELECT id, project_id, platform, method, credentials, enabled, created_at, updated_at
FROM platform_connections
WHERE project_id = $1 AND platform = $2 AND method = $3 AND enabled = true;

-- name: UpdatePlatformConnection :one
UPDATE platform_connections
SET platform = $3, method = $4, credentials = $5, enabled = $6
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, platform, method, credentials, enabled, created_at, updated_at;

-- name: DeletePlatformConnection :exec
DELETE FROM platform_connections
WHERE id = $1 AND project_id = $2;
