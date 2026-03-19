-- +goose Up

-- Ensure admin@werd.io exists with password 'password' (bcrypt cost 12).
-- Idempotent: INSERT does upsert; every other statement is a no-op on re-run.

-- Upsert the admin user. If the row already exists, reset the password hash.
INSERT INTO users (email, password_hash, name)
VALUES (
  'admin@werd.io',
  '$2a$12$uzNJZZl6WMlARtDZY8FgNeHybS.0Vorp.aI08sm602KtXbLFjes5S',
  'Admin'
)
ON CONFLICT (email) DO UPDATE SET
  password_hash = EXCLUDED.password_hash;

-- Migrate project memberships from the old admin to the new one.
INSERT INTO project_members (project_id, user_id, role, created_at)
SELECT pm.project_id, new_user.id, pm.role, pm.created_at
FROM project_members pm
JOIN users old_user ON pm.user_id = old_user.id AND old_user.email = 'admin@yourdomain.com'
CROSS JOIN (SELECT id FROM users WHERE email = 'admin@werd.io') new_user
ON CONFLICT (project_id, user_id) DO NOTHING;

DELETE FROM users WHERE email = 'admin@yourdomain.com';

-- Claim any orphaned projects (projects with zero members).
INSERT INTO project_members (project_id, user_id, role)
SELECT p.id, u.id, 'owner'
FROM projects p
CROSS JOIN (SELECT id FROM users WHERE email = 'admin@werd.io') u
WHERE NOT EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id)
ON CONFLICT (project_id, user_id) DO NOTHING;

-- +goose Down

-- No-op: cannot restore the old password hash.
