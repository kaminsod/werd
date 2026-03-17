-- +goose Up

CREATE TABLE processing_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_id   UUID REFERENCES monitor_sources(id) ON DELETE CASCADE,  -- NULL = project-wide
    name        TEXT NOT NULL DEFAULT '',
    phase       TEXT NOT NULL CHECK (phase IN ('filter', 'classify')),
    rule_type   TEXT NOT NULL CHECK (rule_type IN ('keyword', 'regex', 'llm')),
    config      JSONB NOT NULL DEFAULT '{}',
    priority    INT NOT NULL DEFAULT 0,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_processing_rules_project ON processing_rules(project_id);
CREATE INDEX idx_processing_rules_source ON processing_rules(source_id) WHERE source_id IS NOT NULL;
CREATE INDEX idx_processing_rules_lookup ON processing_rules(project_id, enabled, priority)
    WHERE enabled = true;

-- Trigger to auto-update updated_at.
CREATE TRIGGER set_processing_rules_updated_at
    BEFORE UPDATE ON processing_rules
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- New columns on alerts for classification results and source tracking.
ALTER TABLE alerts ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE alerts ADD COLUMN classification_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE alerts ADD COLUMN monitor_source_id UUID REFERENCES monitor_sources(id) ON DELETE SET NULL;

CREATE INDEX idx_alerts_monitor_source ON alerts(monitor_source_id) WHERE monitor_source_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_alerts_monitor_source;
ALTER TABLE alerts DROP COLUMN IF EXISTS monitor_source_id;
ALTER TABLE alerts DROP COLUMN IF EXISTS classification_reason;
ALTER TABLE alerts DROP COLUMN IF EXISTS tags;

DROP TRIGGER IF EXISTS set_processing_rules_updated_at ON processing_rules;
DROP INDEX IF EXISTS idx_processing_rules_lookup;
DROP INDEX IF EXISTS idx_processing_rules_source;
DROP INDEX IF EXISTS idx_processing_rules_project;
DROP TABLE IF EXISTS processing_rules;
