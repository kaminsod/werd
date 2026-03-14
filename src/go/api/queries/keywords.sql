-- name: CreateKeyword :one
INSERT INTO keywords (project_id, keyword, match_type)
VALUES ($1, $2, $3)
RETURNING id, project_id, keyword, match_type, created_at;

-- name: ListKeywords :many
SELECT id, project_id, keyword, match_type, created_at
FROM keywords
WHERE project_id = $1
ORDER BY created_at;

-- name: GetKeywordByID :one
SELECT id, project_id, keyword, match_type, created_at
FROM keywords
WHERE id = $1 AND project_id = $2;

-- name: DeleteKeyword :exec
DELETE FROM keywords
WHERE id = $1 AND project_id = $2;
