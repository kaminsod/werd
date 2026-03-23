-- name: CreateServiceInstance :one
INSERT INTO service_instances (project_id, service, external_id, config, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, project_id, service, external_id, config, status, created_at, updated_at;

-- name: GetServiceInstanceByExternalID :one
SELECT id, project_id, service, external_id, config, status, created_at, updated_at
FROM service_instances
WHERE project_id = $1 AND service = $2 AND external_id = $3;

-- name: UpdateServiceInstance :exec
UPDATE service_instances
SET config = $2, status = $3
WHERE id = $1;

-- name: DeleteServiceInstanceByExternalID :exec
DELETE FROM service_instances
WHERE project_id = $1 AND service = $2 AND external_id = $3;
