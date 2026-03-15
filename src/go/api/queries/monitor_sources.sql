-- name: CreateMonitorSource :one
INSERT INTO monitor_sources (project_id, type, config, enabled)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, type, config, enabled, created_at, updated_at;

-- name: ListMonitorSources :many
SELECT id, project_id, type, config, enabled, created_at, updated_at
FROM monitor_sources
WHERE project_id = $1
ORDER BY created_at;

-- name: GetMonitorSourceByID :one
SELECT id, project_id, type, config, enabled, created_at, updated_at
FROM monitor_sources
WHERE id = $1 AND project_id = $2;

-- name: UpdateMonitorSource :one
UPDATE monitor_sources
SET type = $3, config = $4, enabled = $5
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, type, config, enabled, created_at, updated_at;

-- name: DeleteMonitorSource :exec
DELETE FROM monitor_sources
WHERE id = $1 AND project_id = $2;

-- name: ListEnabledMonitorSources :many
SELECT id, project_id, type, config, enabled, created_at, updated_at
FROM monitor_sources
WHERE project_id = $1 AND enabled = true
ORDER BY created_at;
