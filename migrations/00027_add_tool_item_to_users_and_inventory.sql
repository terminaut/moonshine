-- +goose Up
ALTER TABLE tool_items ADD COLUMN slug VARCHAR(255);
UPDATE tool_items SET slug = LOWER(REPLACE(name, ' ', '_')) WHERE slug IS NULL;
ALTER TABLE tool_items ALTER COLUMN slug SET NOT NULL;
ALTER TABLE tool_items ADD CONSTRAINT uq_tool_items_slug UNIQUE (slug);

ALTER TABLE users ADD COLUMN tool_item_id UUID;
ALTER TABLE users ADD CONSTRAINT fk_users_tool_item FOREIGN KEY (tool_item_id) REFERENCES tool_items(id) ON DELETE SET NULL;

ALTER TABLE inventory DROP CONSTRAINT IF EXISTS chk_inventory_item_type;
ALTER TABLE inventory ADD COLUMN tool_item_id UUID;
ALTER TABLE inventory ADD CONSTRAINT fk_inventory_tool_item FOREIGN KEY (tool_item_id) REFERENCES tool_items(id) ON DELETE CASCADE;
ALTER TABLE inventory ADD CONSTRAINT chk_inventory_item_type CHECK (
    (equipment_item_id IS NOT NULL AND potion_id IS NULL AND tool_item_id IS NULL) OR
    (equipment_item_id IS NULL AND potion_id IS NOT NULL AND tool_item_id IS NULL) OR
    (equipment_item_id IS NULL AND potion_id IS NULL AND tool_item_id IS NOT NULL)
);

-- +goose Down
ALTER TABLE inventory DROP CONSTRAINT IF EXISTS chk_inventory_item_type;
ALTER TABLE inventory DROP CONSTRAINT IF EXISTS fk_inventory_tool_item;
ALTER TABLE inventory DROP COLUMN IF EXISTS tool_item_id;
ALTER TABLE inventory ADD CONSTRAINT chk_inventory_item_type CHECK (
    (equipment_item_id IS NOT NULL AND potion_id IS NULL) OR
    (equipment_item_id IS NULL AND potion_id IS NOT NULL)
);
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_tool_item;
ALTER TABLE users DROP COLUMN IF EXISTS tool_item_id;
ALTER TABLE tool_items DROP CONSTRAINT IF EXISTS uq_tool_items_slug;
ALTER TABLE tool_items DROP COLUMN IF EXISTS slug;
