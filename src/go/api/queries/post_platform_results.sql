-- name: CreatePostPlatformResult :one
INSERT INTO post_platform_results (post_id, platform, platform_post_id, platform_url, success, error_message, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, post_id, platform, platform_post_id, platform_url, success, error_message, published_at, created_at;

-- name: ListPostPlatformResults :many
SELECT id, post_id, platform, platform_post_id, platform_url, success, error_message, published_at, created_at
FROM post_platform_results
WHERE post_id = $1
ORDER BY created_at;
