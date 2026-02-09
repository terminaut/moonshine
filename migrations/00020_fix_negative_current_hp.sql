-- +goose Up
-- +goose StatementBegin
-- Fix existing negative HP values
UPDATE users SET current_hp = 0 WHERE current_hp < 0;

-- Add check constraint to prevent negative HP
ALTER TABLE users ADD CONSTRAINT check_current_hp_non_negative CHECK (current_hp >= 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_current_hp_non_negative;
-- +goose StatementEnd
