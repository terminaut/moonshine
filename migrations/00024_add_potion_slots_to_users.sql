-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN potion1_id UUID;
ALTER TABLE users ADD COLUMN potion2_id UUID;
ALTER TABLE users ADD COLUMN potion3_id UUID;
ALTER TABLE users ADD CONSTRAINT fk_users_potion1 FOREIGN KEY (potion1_id) REFERENCES potions(id) ON DELETE SET NULL;
ALTER TABLE users ADD CONSTRAINT fk_users_potion2 FOREIGN KEY (potion2_id) REFERENCES potions(id) ON DELETE SET NULL;
ALTER TABLE users ADD CONSTRAINT fk_users_potion3 FOREIGN KEY (potion3_id) REFERENCES potions(id) ON DELETE SET NULL;
CREATE INDEX idx_users_potion1 ON users (potion1_id) WHERE potion1_id IS NOT NULL;
CREATE INDEX idx_users_potion2 ON users (potion2_id) WHERE potion2_id IS NOT NULL;
CREATE INDEX idx_users_potion3 ON users (potion3_id) WHERE potion3_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_potion3;
DROP INDEX IF EXISTS idx_users_potion2;
DROP INDEX IF EXISTS idx_users_potion1;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_potion3;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_potion2;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_potion1;
ALTER TABLE users DROP COLUMN IF EXISTS potion3_id;
ALTER TABLE users DROP COLUMN IF EXISTS potion2_id;
ALTER TABLE users DROP COLUMN IF EXISTS potion1_id;
-- +goose StatementEnd
