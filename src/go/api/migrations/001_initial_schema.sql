-- +goose Up

-- ============================================================================
-- Enum types
-- ============================================================================

CREATE TYPE project_role AS ENUM ('owner', 'admin', 'member', 'viewer');

CREATE TYPE service_name AS ENUM ('ntfy', 'changedetect', 'umami', 'rsshub');

CREATE TYPE service_status AS ENUM ('provisioning', 'active', 'error', 'deprovisioning');

CREATE TYPE monitor_type AS ENUM ('reddit', 'hn', 'web', 'rss', 'github');

CREATE TYPE keyword_match_type AS ENUM ('exact', 'substring', 'regex');

CREATE TYPE alert_severity AS ENUM ('low', 'medium', 'high', 'critical');

CREATE TYPE alert_status AS ENUM ('new', 'seen', 'triaged', 'dismissed', 'responded');

CREATE TYPE notification_source_type AS ENUM ('reddit', 'hn', 'web', 'rss', 'github', 'all');

CREATE TYPE notification_destination AS ENUM ('ntfy', 'email', 'webhook');

CREATE TYPE post_status AS ENUM ('draft', 'scheduled', 'publishing', 'published', 'failed');

-- ============================================================================
-- Trigger function: auto-update updated_at on row modification
-- ============================================================================

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- ============================================================================
-- Tables
-- ============================================================================

-- ── Core ──

CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    settings    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE project_members (
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        project_role NOT NULL DEFAULT 'member',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

-- ── Service integration ──

CREATE TABLE service_instances (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    service     service_name NOT NULL,
    external_id TEXT NOT NULL DEFAULT '',
    config      JSONB NOT NULL DEFAULT '{}',
    status      service_status NOT NULL DEFAULT 'provisioning',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER service_instances_updated_at
    BEFORE UPDATE ON service_instances
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Monitoring ──

CREATE TABLE monitor_sources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    type        monitor_type NOT NULL,
    config      JSONB NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER monitor_sources_updated_at
    BEFORE UPDATE ON monitor_sources
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE keywords (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    keyword     TEXT NOT NULL,
    match_type  keyword_match_type NOT NULL DEFAULT 'substring',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Alerts & notifications ──

CREATE TABLE alerts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_type         monitor_type NOT NULL,
    source_id           TEXT NOT NULL,
    title               TEXT NOT NULL DEFAULT '',
    content             TEXT NOT NULL DEFAULT '',
    url                 TEXT NOT NULL DEFAULT '',
    matched_keywords    TEXT[] NOT NULL DEFAULT '{}',
    severity            alert_severity NOT NULL DEFAULT 'low',
    status              alert_status NOT NULL DEFAULT 'new',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, source_type, source_id)
);

CREATE TRIGGER alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE notification_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_type     notification_source_type NOT NULL DEFAULT 'all',
    min_severity    alert_severity NOT NULL DEFAULT 'low',
    destination     notification_destination NOT NULL,
    config          JSONB NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Publishing ──

CREATE TABLE platform_connections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    platform    TEXT NOT NULL,
    credentials JSONB NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER platform_connections_updated_at
    BEFORE UPDATE ON platform_connections
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE published_posts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    content         TEXT NOT NULL DEFAULT '',
    platforms       TEXT[] NOT NULL DEFAULT '{}',
    scheduled_at    TIMESTAMPTZ,
    published_at    TIMESTAMPTZ,
    status          post_status NOT NULL DEFAULT 'draft',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER published_posts_updated_at
    BEFORE UPDATE ON published_posts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Audit ──

CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    action      TEXT NOT NULL,
    details     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- Indexes
-- ============================================================================

-- project_members: reverse lookup (which projects is user X in?)
CREATE INDEX idx_project_members_user_id ON project_members (user_id);

-- Per-project lookups on tenant-scoped tables (also speeds CASCADE deletes).
CREATE INDEX idx_service_instances_project_id ON service_instances (project_id);
CREATE INDEX idx_monitor_sources_project_id ON monitor_sources (project_id);
CREATE INDEX idx_keywords_project_id ON keywords (project_id);
CREATE INDEX idx_platform_connections_project_id ON platform_connections (project_id);
CREATE INDEX idx_notification_rules_project_id ON notification_rules (project_id);

-- Alerts: dashboard timeline (newest per project).
CREATE INDEX idx_alerts_project_created ON alerts (project_id, created_at DESC);

-- Alerts: filter by triage status within a project.
CREATE INDEX idx_alerts_project_status ON alerts (project_id, status);

-- Published posts: list by status within a project.
CREATE INDEX idx_published_posts_project_status ON published_posts (project_id, status);

-- Published posts: scheduler finds due posts (partial index, small and fast).
CREATE INDEX idx_published_posts_scheduled ON published_posts (status, scheduled_at)
    WHERE status = 'scheduled';

-- Audit log: per-project timeline.
CREATE INDEX idx_audit_log_project_created ON audit_log (project_id, created_at DESC);


-- +goose Down

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS published_posts;
DROP TABLE IF EXISTS platform_connections;
DROP TABLE IF EXISTS notification_rules;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS keywords;
DROP TABLE IF EXISTS monitor_sources;
DROP TABLE IF EXISTS service_instances;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS projects;

DROP FUNCTION IF EXISTS set_updated_at();

DROP TYPE IF EXISTS post_status;
DROP TYPE IF EXISTS notification_destination;
DROP TYPE IF EXISTS notification_source_type;
DROP TYPE IF EXISTS alert_status;
DROP TYPE IF EXISTS alert_severity;
DROP TYPE IF EXISTS keyword_match_type;
DROP TYPE IF EXISTS monitor_type;
DROP TYPE IF EXISTS service_status;
DROP TYPE IF EXISTS service_name;
DROP TYPE IF EXISTS project_role;
