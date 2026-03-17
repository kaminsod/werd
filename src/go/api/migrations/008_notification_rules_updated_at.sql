-- +goose Up
ALTER TABLE notification_rules
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TRIGGER set_notification_rules_updated_at
    BEFORE UPDATE ON notification_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS set_notification_rules_updated_at ON notification_rules;
ALTER TABLE notification_rules DROP COLUMN IF EXISTS updated_at;
