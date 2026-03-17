-- name: CreateNotificationRule :one
INSERT INTO notification_rules (project_id, source_type, min_severity, destination, config, enabled)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, project_id, source_type, min_severity, destination, config, enabled, created_at, updated_at;

-- name: ListNotificationRules :many
SELECT id, project_id, source_type, min_severity, destination, config, enabled, created_at, updated_at
FROM notification_rules
WHERE project_id = $1
ORDER BY created_at;

-- name: GetNotificationRuleByID :one
SELECT id, project_id, source_type, min_severity, destination, config, enabled, created_at, updated_at
FROM notification_rules
WHERE id = $1 AND project_id = $2;

-- name: UpdateNotificationRule :one
UPDATE notification_rules
SET source_type = $3, min_severity = $4, destination = $5, config = $6, enabled = $7
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, source_type, min_severity, destination, config, enabled, created_at, updated_at;

-- name: DeleteNotificationRule :exec
DELETE FROM notification_rules
WHERE id = $1 AND project_id = $2;

-- name: ListEnabledRulesForProject :many
SELECT id, project_id, source_type, min_severity, destination, config, enabled, created_at, updated_at
FROM notification_rules
WHERE project_id = $1 AND enabled = true
ORDER BY created_at;
