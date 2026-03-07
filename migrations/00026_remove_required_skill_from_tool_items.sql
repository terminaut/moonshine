-- +goose Up
ALTER TABLE tool_items DROP COLUMN IF EXISTS required_skill;

-- +goose Down
ALTER TABLE tool_items ADD COLUMN required_skill INTEGER NOT NULL DEFAULT 0;
