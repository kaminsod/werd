-- name: CreatePostPlatformResult :one
INSERT INTO post_platform_results (post_id, platform, platform_post_id, platform_url, success, error_message, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, post_id, platform, platform_post_id, platform_url, success, error_message, published_at, created_at, monitor_replies, last_reply_check, last_known_reply_id;

-- name: ListPostPlatformResults :many
SELECT id, post_id, platform, platform_post_id, platform_url, success, error_message, published_at, created_at, monitor_replies, last_reply_check, last_known_reply_id
FROM post_platform_results
WHERE post_id = $1
ORDER BY created_at;

-- name: SetMonitorReplies :exec
UPDATE post_platform_results
SET monitor_replies = $2
WHERE id = $1;

-- name: ListMonitoredResults :many
SELECT ppr.id, ppr.post_id, ppr.platform, ppr.platform_post_id, ppr.platform_url,
       ppr.success, ppr.error_message, ppr.published_at, ppr.created_at,
       ppr.monitor_replies, ppr.last_reply_check, ppr.last_known_reply_id,
       pp.project_id
FROM post_platform_results ppr
JOIN published_posts pp ON pp.id = ppr.post_id
WHERE ppr.monitor_replies = true
  AND ppr.success = true
  AND ppr.platform_post_id != ''
ORDER BY ppr.last_reply_check NULLS FIRST
LIMIT $1;

-- name: UpdateReplyCheckpoint :exec
UPDATE post_platform_results
SET last_reply_check = now(), last_known_reply_id = $2
WHERE id = $1;
