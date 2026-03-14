-- name: UpsertAlert :one
INSERT INTO alerts (project_id, source_type, source_id, title, content, url, matched_keywords, severity)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (project_id, source_type, source_id)
DO UPDATE SET
  title = EXCLUDED.title,
  content = EXCLUDED.content,
  url = EXCLUDED.url,
  matched_keywords = EXCLUDED.matched_keywords
RETURNING id, project_id, source_type, source_id, title, content, url, matched_keywords, severity, status, created_at, updated_at;

-- name: GetAlertByID :one
SELECT id, project_id, source_type, source_id, title, content, url, matched_keywords, severity, status, created_at, updated_at
FROM alerts
WHERE id = $1 AND project_id = $2;

-- name: ListAlerts :many
SELECT id, project_id, source_type, source_id, title, content, url, matched_keywords, severity, status, created_at, updated_at
FROM alerts
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAlertsByStatus :many
SELECT id, project_id, source_type, source_id, title, content, url, matched_keywords, severity, status, created_at, updated_at
FROM alerts
WHERE project_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAlertsBySourceType :many
SELECT id, project_id, source_type, source_id, title, content, url, matched_keywords, severity, status, created_at, updated_at
FROM alerts
WHERE project_id = $1 AND source_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateAlertStatus :one
UPDATE alerts
SET status = $3
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, source_type, source_id, title, content, url, matched_keywords, severity, status, created_at, updated_at;

-- name: CountAlerts :one
SELECT count(*) FROM alerts WHERE project_id = $1;

-- name: CountAlertsByStatus :one
SELECT count(*) FROM alerts WHERE project_id = $1 AND status = $2;
