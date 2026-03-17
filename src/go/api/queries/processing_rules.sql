-- name: CreateProcessingRule :one
INSERT INTO processing_rules (project_id, source_id, name, phase, rule_type, config, priority, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetProcessingRuleByID :one
SELECT * FROM processing_rules
WHERE id = $1 AND project_id = $2;

-- name: ListProcessingRules :many
SELECT * FROM processing_rules
WHERE project_id = $1
ORDER BY phase, priority, created_at;

-- name: UpdateProcessingRule :one
UPDATE processing_rules
SET source_id = $3, name = $4, phase = $5, rule_type = $6, config = $7, priority = $8, enabled = $9
WHERE id = $1 AND project_id = $2
RETURNING *;

-- name: DeleteProcessingRule :exec
DELETE FROM processing_rules
WHERE id = $1 AND project_id = $2;

-- name: ListRulesForSource :many
SELECT * FROM processing_rules
WHERE (source_id = $1 OR source_id IS NULL)
  AND project_id = $2
  AND enabled = true
ORDER BY priority;
