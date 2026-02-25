-- +goose Up
-- +goose StatementBegin
ALTER TABLE inventory ALTER COLUMN equipment_item_id DROP NOT NULL;
ALTER TABLE inventory ADD COLUMN potion_id UUID;
ALTER TABLE inventory ADD CONSTRAINT fk_inventory_potion FOREIGN KEY (potion_id) REFERENCES potions(id) ON DELETE CASCADE;
ALTER TABLE inventory ADD CONSTRAINT chk_inventory_item_type CHECK (
    (equipment_item_id IS NOT NULL AND potion_id IS NULL) OR
    (equipment_item_id IS NULL AND potion_id IS NOT NULL)
);
CREATE INDEX idx_inventory_potion_id ON inventory (potion_id) WHERE potion_id IS NOT NULL;
CREATE INDEX idx_inventory_user_id ON inventory (user_id) WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_inventory_user_id;
DROP INDEX IF EXISTS idx_inventory_potion_id;
ALTER TABLE inventory DROP CONSTRAINT IF EXISTS chk_inventory_item_type;
ALTER TABLE inventory DROP CONSTRAINT IF EXISTS fk_inventory_potion;
ALTER TABLE inventory DROP COLUMN IF EXISTS potion_id;
ALTER TABLE inventory ALTER COLUMN equipment_item_id SET NOT NULL;
-- +goose StatementEnd
