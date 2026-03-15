-- name: CreatePublishedPost :one
INSERT INTO published_posts (project_id, content, platforms, status)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, content, platforms, scheduled_at, published_at, status, created_at, updated_at;

-- name: GetPublishedPostByID :one
SELECT id, project_id, content, platforms, scheduled_at, published_at, status, created_at, updated_at
FROM published_posts
WHERE id = $1 AND project_id = $2;

-- name: ListPublishedPosts :many
SELECT id, project_id, content, platforms, scheduled_at, published_at, status, created_at, updated_at
FROM published_posts
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPublishedPostsByStatus :many
SELECT id, project_id, content, platforms, scheduled_at, published_at, status, created_at, updated_at
FROM published_posts
WHERE project_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdatePublishedPost :one
UPDATE published_posts
SET content = $3, platforms = $4
WHERE id = $1 AND project_id = $2 AND status = 'draft'
RETURNING id, project_id, content, platforms, scheduled_at, published_at, status, created_at, updated_at;

-- name: UpdatePublishedPostStatus :one
UPDATE published_posts
SET status = $3
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, content, platforms, scheduled_at, published_at, status, created_at, updated_at;

-- name: SetPublishedPostPublished :one
UPDATE published_posts
SET status = 'published', published_at = now()
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, content, platforms, scheduled_at, published_at, status, created_at, updated_at;

-- name: DeletePublishedPost :exec
DELETE FROM published_posts
WHERE id = $1 AND project_id = $2 AND status = 'draft';

-- name: CountPublishedPosts :one
SELECT count(*) FROM published_posts WHERE project_id = $1;

-- name: CountPublishedPostsByStatus :one
SELECT count(*) FROM published_posts WHERE project_id = $1 AND status = $2;
