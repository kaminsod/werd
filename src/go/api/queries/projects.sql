-- name: CreateProject :one
INSERT INTO projects (name, slug, settings)
VALUES ($1, $2, $3)
RETURNING id, name, slug, settings, created_at, updated_at;

-- name: GetProjectByID :one
SELECT id, name, slug, settings, created_at, updated_at
FROM projects
WHERE id = $1;

-- name: UpdateProject :one
UPDATE projects
SET name = $2, slug = $3, settings = $4
WHERE id = $1
RETURNING id, name, slug, settings, created_at, updated_at;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;

-- name: ProjectExistsBySlug :one
SELECT EXISTS(SELECT 1 FROM projects WHERE slug = $1);

-- name: ProjectExistsBySlugExcludingID :one
SELECT EXISTS(SELECT 1 FROM projects WHERE slug = $1 AND id != $2);

-- name: ListProjectsForUser :many
SELECT p.id, p.name, p.slug, p.settings, p.created_at, p.updated_at
FROM projects p
JOIN project_members pm ON pm.project_id = p.id
WHERE pm.user_id = $1
ORDER BY p.created_at DESC;

-- name: CreateProjectMember :one
INSERT INTO project_members (project_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING project_id, user_id, role, created_at;

-- name: GetProjectMember :one
SELECT project_id, user_id, role, created_at
FROM project_members
WHERE project_id = $1 AND user_id = $2;

-- name: ListProjectMembers :many
SELECT pm.project_id, pm.user_id, pm.role, pm.created_at,
       u.email AS user_email, u.name AS user_name
FROM project_members pm
JOIN users u ON u.id = pm.user_id
WHERE pm.project_id = $1
ORDER BY pm.created_at;

-- name: UpdateProjectMemberRole :exec
UPDATE project_members
SET role = $3
WHERE project_id = $1 AND user_id = $2;

-- name: DeleteProjectMember :exec
DELETE FROM project_members
WHERE project_id = $1 AND user_id = $2;
