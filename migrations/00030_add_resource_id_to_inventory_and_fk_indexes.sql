-- +goose Up
-- +goose StatementBegin
ALTER TABLE inventory DROP CONSTRAINT IF EXISTS chk_inventory_item_type;
ALTER TABLE inventory ADD COLUMN IF NOT EXISTS resource_id UUID;
ALTER TABLE inventory
	ADD CONSTRAINT fk_inventory_resource FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE;
ALTER TABLE inventory ADD CONSTRAINT chk_inventory_item_type CHECK (
	(equipment_item_id IS NOT NULL AND potion_id IS NULL AND tool_item_id IS NULL AND resource_id IS NULL) OR
	(equipment_item_id IS NULL AND potion_id IS NOT NULL AND tool_item_id IS NULL AND resource_id IS NULL) OR
	(equipment_item_id IS NULL AND potion_id IS NULL AND tool_item_id IS NOT NULL AND resource_id IS NULL) OR
	(equipment_item_id IS NULL AND potion_id IS NULL AND tool_item_id IS NULL AND resource_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_tool_items_tool_category_id_fk ON tool_items (tool_category_id);
CREATE INDEX IF NOT EXISTS idx_equipment_items_equipment_category_id_fk ON equipment_items (equipment_category_id);

CREATE INDEX IF NOT EXISTS idx_users_avatar_id_fk ON users (avatar_id);
CREATE INDEX IF NOT EXISTS idx_users_location_id_fk ON users (location_id);
CREATE INDEX IF NOT EXISTS idx_users_chest_equipment_item_id_fk ON users (chest_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_belt_equipment_item_id_fk ON users (belt_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_head_equipment_item_id_fk ON users (head_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_neck_equipment_item_id_fk ON users (neck_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_weapon_equipment_item_id_fk ON users (weapon_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_shield_equipment_item_id_fk ON users (shield_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_legs_equipment_item_id_fk ON users (legs_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_feet_equipment_item_id_fk ON users (feet_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_arms_equipment_item_id_fk ON users (arms_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_hands_equipment_item_id_fk ON users (hands_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_ring1_equipment_item_id_fk ON users (ring1_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_ring2_equipment_item_id_fk ON users (ring2_equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_users_potion1_id_fk ON users (potion1_id);
CREATE INDEX IF NOT EXISTS idx_users_potion2_id_fk ON users (potion2_id);
CREATE INDEX IF NOT EXISTS idx_users_potion3_id_fk ON users (potion3_id);
CREATE INDEX IF NOT EXISTS idx_users_tool_item_id_fk ON users (tool_item_id);

CREATE INDEX IF NOT EXISTS idx_inventory_user_id_fk ON inventory (user_id);
CREATE INDEX IF NOT EXISTS idx_inventory_equipment_item_id_fk ON inventory (equipment_item_id);
CREATE INDEX IF NOT EXISTS idx_inventory_potion_id_fk ON inventory (potion_id);
CREATE INDEX IF NOT EXISTS idx_inventory_tool_item_id_fk ON inventory (tool_item_id);
CREATE INDEX IF NOT EXISTS idx_inventory_resource_id_fk ON inventory (resource_id);

CREATE INDEX IF NOT EXISTS idx_location_bots_location_id_fk ON location_bots (location_id);
CREATE INDEX IF NOT EXISTS idx_location_bots_bot_id_fk ON location_bots (bot_id);

CREATE INDEX IF NOT EXISTS idx_location_locations_location_id_fk ON location_locations (location_id);
CREATE INDEX IF NOT EXISTS idx_location_locations_near_location_id_fk ON location_locations (near_location_id);

CREATE INDEX IF NOT EXISTS idx_fights_user_id_fk ON fights (user_id);
CREATE INDEX IF NOT EXISTS idx_fights_bot_id_fk ON fights (bot_id);
CREATE INDEX IF NOT EXISTS idx_rounds_fight_id_fk ON rounds (fight_id);
CREATE INDEX IF NOT EXISTS idx_movement_logs_user_id_fk ON movement_logs (user_id);

CREATE INDEX IF NOT EXISTS idx_resources_tool_category_id_fk ON resources (tool_category_id);
CREATE INDEX IF NOT EXISTS idx_resources_tool_item_id_fk ON resources (tool_item_id);

CREATE INDEX IF NOT EXISTS idx_location_resources_location_id_fk ON location_resources (location_id);
CREATE INDEX IF NOT EXISTS idx_location_resources_resource_id_fk ON location_resources (resource_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_location_resources_resource_id_fk;
DROP INDEX IF EXISTS idx_location_resources_location_id_fk;
DROP INDEX IF EXISTS idx_resources_tool_item_id_fk;
DROP INDEX IF EXISTS idx_resources_tool_category_id_fk;
DROP INDEX IF EXISTS idx_movement_logs_user_id_fk;
DROP INDEX IF EXISTS idx_rounds_fight_id_fk;
DROP INDEX IF EXISTS idx_fights_bot_id_fk;
DROP INDEX IF EXISTS idx_fights_user_id_fk;
DROP INDEX IF EXISTS idx_location_locations_near_location_id_fk;
DROP INDEX IF EXISTS idx_location_locations_location_id_fk;
DROP INDEX IF EXISTS idx_location_bots_bot_id_fk;
DROP INDEX IF EXISTS idx_location_bots_location_id_fk;
DROP INDEX IF EXISTS idx_inventory_resource_id_fk;
DROP INDEX IF EXISTS idx_inventory_tool_item_id_fk;
DROP INDEX IF EXISTS idx_inventory_potion_id_fk;
DROP INDEX IF EXISTS idx_inventory_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_inventory_user_id_fk;
DROP INDEX IF EXISTS idx_users_tool_item_id_fk;
DROP INDEX IF EXISTS idx_users_potion3_id_fk;
DROP INDEX IF EXISTS idx_users_potion2_id_fk;
DROP INDEX IF EXISTS idx_users_potion1_id_fk;
DROP INDEX IF EXISTS idx_users_ring2_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_ring1_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_hands_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_arms_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_feet_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_legs_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_shield_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_weapon_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_neck_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_head_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_belt_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_chest_equipment_item_id_fk;
DROP INDEX IF EXISTS idx_users_location_id_fk;
DROP INDEX IF EXISTS idx_users_avatar_id_fk;
DROP INDEX IF EXISTS idx_equipment_items_equipment_category_id_fk;
DROP INDEX IF EXISTS idx_tool_items_tool_category_id_fk;

ALTER TABLE inventory DROP CONSTRAINT IF EXISTS chk_inventory_item_type;
ALTER TABLE inventory DROP CONSTRAINT IF EXISTS fk_inventory_resource;
ALTER TABLE inventory DROP COLUMN IF EXISTS resource_id;
ALTER TABLE inventory ADD CONSTRAINT chk_inventory_item_type CHECK (
	(equipment_item_id IS NOT NULL AND potion_id IS NULL AND tool_item_id IS NULL) OR
	(equipment_item_id IS NULL AND potion_id IS NOT NULL AND tool_item_id IS NULL) OR
	(equipment_item_id IS NULL AND potion_id IS NULL AND tool_item_id IS NOT NULL)
);
-- +goose StatementEnd
